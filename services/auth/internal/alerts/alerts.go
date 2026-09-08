// Package alerts turns a saved search into a price watch.
//
// The schema for this shipped on day one and the runtime never did: alerts_sent
// was never written by any code, last_median appeared in exactly one SELECT and
// was never assigned, and no scheduler existed. The checkbox on the account page
// saved successfully, the row persisted, the API returned 200, and nothing was
// ever going to arrive. It failed by staying quiet, which is why it survived
// until somebody asked what the feature was for.
//
// Nothing here uses a model. A saved query is re-run through the same
// deterministic parser and ranking the live search uses, and comparing two
// medians is arithmetic.
package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"time"

	appmail "github.com/sobhanaz/khodrobin/auth/internal/mail"
	"github.com/sobhanaz/khodrobin/auth/internal/store"
)

// Runner re-runs watched searches against the search API on a schedule.
type Runner struct {
	store  *store.Store
	mail   Mailer
	apiURL string
	base   string
	log    *slog.Logger
	client *http.Client
}

// Mailer is the subset of the mail service this needs, so the runner can be
// tested without an SMTP server.
type Mailer interface {
	Send(to, subject, body string) error
}

// batch caps how many searches one cycle examines. The work is one HTTP call
// each against a service that answers in about a millisecond, so the ceiling is
// about politeness to the index rather than cost.
const batch = 200

func New(st *store.Store, m Mailer, apiURL, baseURL string, log *slog.Logger) *Runner {
	return &Runner{
		store: st, mail: m, apiURL: apiURL, base: baseURL, log: log,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Start runs a cycle on every tick until ctx is cancelled.
//
// The interval should be longer than the crawl interval. Running more often
// than the data changes cannot find anything new and only adds load.
func (r *Runner) Start(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if n, err := r.RunOnce(ctx); err != nil {
				r.log.Error("alert cycle failed", "err", err)
			} else if n > 0 {
				r.log.Info("alerts sent", "count", n)
			}
		}
	}
}

// topSpec is the shape this needs out of the search response. The rest of the
// payload is ignored on purpose: decoding only what is used means a new field
// in the API cannot break the watcher.
type topSpec struct {
	Result struct {
		Specs []struct {
			Key            string `json:"key"`
			NameFa         string `json:"name_fa"`
			MedianPrice    int64  `json:"median_price"`
			MedianReliable bool   `json:"median_reliable"`
			OfferCount     int    `json:"offer_count"`
		} `json:"specs"`
	} `json:"result"`
}

// RunOnce examines every watched search and returns how many alerts were sent.
//
// One failure does not stop the cycle. A search whose query no longer matches
// anything, or whose API call fails, is skipped and retried next time rather
// than aborting every other person's alerts.
func (r *Runner) RunOnce(ctx context.Context) (int, error) {
	watched, err := r.store.DueAlerts(ctx, batch)
	if err != nil {
		return 0, err
	}
	sent := 0
	for _, w := range watched {
		spec, err := r.top(ctx, w.Query, w.Mode)
		if err != nil {
			r.log.Warn("alert search failed", "search_id", w.ID, "err", err)
			continue
		}
		if spec == nil {
			continue
		}
		if r.consider(ctx, w, spec) {
			sent++
		}
	}
	return sent, nil
}

// consider decides whether this observation is worth an email, and reports
// whether one was sent.
func (r *Runner) consider(ctx context.Context, w store.AlertingSearch, spec *specView) bool {
	// A median over fewer than three offers is the mean of two asking prices
	// nobody is asking. The card refuses to quote it, so an alert must not
	// either: mailing somebody about a movement in a number the product itself
	// declines to stand behind would be worse than silence.
	if !spec.reliable {
		return false
	}

	prior := w.LastMedian
	// Always record what was seen, even when no mail goes out. Otherwise a slow
	// drift is measured against a stale baseline and eventually fires as one
	// dramatic alert that never actually happened in one step.
	defer func() {
		if err := r.store.MarkAlertRun(ctx, w.ID, spec.median); err != nil {
			r.log.Warn("mark alert run", "search_id", w.ID, "err", err)
		}
	}()

	// First observation establishes the baseline. There is nothing to compare
	// against yet, and mailing "the price is 500M" to somebody who just asked to
	// be told when it *changes* is not what they asked for.
	if prior == nil || *prior == 0 {
		return false
	}

	delta := percentChange(*prior, spec.median)
	if math.Abs(delta) < w.AlertPct {
		return false
	}

	fresh, err := r.store.RecordAlert(ctx, w.ID, spec.key, *prior, spec.median)
	if err != nil {
		r.log.Error("record alert", "search_id", w.ID, "err", err)
		return false
	}
	// Already mailed for this spec at this price recently. A price oscillating
	// across the threshold would otherwise mail on every cycle.
	if !fresh {
		return false
	}

	link := r.base + "/search?q=" + url.QueryEscape(w.Query)
	subject, body := appmail.PriceAlert(spec.name, w.Query, *prior, spec.median, link)
	if err := r.mail.Send(w.Email, subject, body); err != nil {
		r.log.Warn("alert mail failed", "search_id", w.ID, "err", err)
		return false
	}
	r.log.Info("alert.sent", "search_id", w.ID, "spec", spec.key,
		"old", *prior, "new", spec.median, "pct", delta)
	return true
}

type specView struct {
	key, name string
	median    int64
	reliable  bool
	offers    int
}

// top runs the saved query through the live search API and returns its first
// result, which is the card the person was looking at when they saved it.
func (r *Runner) top(ctx context.Context, query, mode string) (*specView, error) {
	u := fmt.Sprintf("%s/api/v1/search?q=%s&mode=%s",
		r.apiURL, url.QueryEscape(query), url.QueryEscape(mode))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	res, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search returned %d", res.StatusCode)
	}
	var body topSpec
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, err
	}
	if len(body.Result.Specs) == 0 {
		return nil, nil
	}
	s := body.Result.Specs[0]
	return &specView{key: s.Key, name: s.NameFa, median: s.MedianPrice,
		reliable: s.MedianReliable, offers: s.OfferCount}, nil
}

// percentChange is signed: negative means the price came down, which for a
// buyer is the direction worth waking up for.
func percentChange(old, now int64) float64 {
	if old == 0 {
		return 0
	}
	return (float64(now) - float64(old)) / float64(old) * 100
}
