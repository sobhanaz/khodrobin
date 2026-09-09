package index

import (
	"encoding/json"
	"log/slog"
	"os"
	"sort"
	"sync/atomic"
	"time"
)

// Per-listing detail, fetched by the crawler from the source's own ad page and
// served from memory.
//
// The list pages the index is built from do not carry a seller's description,
// colour, damage report or gallery; the detail pages do. This file is the only
// place those extra fields live, and it deliberately holds nothing the index
// already holds — no price, no mileage. Divar publishes rials and Bama publishes
// tomans while both label the field IRR, so a second writer of prices is a
// second chance to be off by ten.
//
// Nullable fields are pointers rather than strings-with-omitempty because the
// UI distinguishes "the source did not publish a colour" from "the colour is
// empty", and a missing key and a null are different answers to that.
type Detail struct {
	Description *string  `json:"description"`
	Color       *string  `json:"color"`
	Damages     *string  `json:"damages"`
	Condition   *string  `json:"condition"`
	Fuel        *string  `json:"fuel"`
	Body        *string  `json:"body"`
	Engine      *string  `json:"engine"`
	Images      []string `json:"images"`
	// hamrah-mechanic publishes a professional inspection; the other four do
	// not. Its shape is that source's business, so it is carried through
	// verbatim instead of being modelled here — a struct would silently drop
	// every field the crawler starts collecting next month. The outer
	// json.Unmarshal has already proved these bytes are well-formed JSON, so
	// echoing them into a response cannot produce a malformed one.
	Inspection json.RawMessage   `json:"inspection"`
	Extras     map[string]string `json:"extras"`
	FetchedAt  string            `json:"fetched_at"`
}

type detailFile struct {
	BuiltAt string            `json:"built_at"`
	Details map[string]Detail `json:"details"`
}

// Response ceilings, applied once per reload rather than per request.
//
// A description is the seller's own free text: nothing on the writing side
// bounds it, and the crawler cannot know what a seller pasted into the box. An
// unbounded field on the read path is an unbounded response on every request
// for that listing, so the ceiling belongs here, where it costs one pass per
// reload instead of one pass per request. The values are chosen to be far
// above any plausible ad and far below anything that would matter to a
// response body; they are limits, not measurements, and nothing derived from
// them is shown to a user.
const (
	maxDescriptionRunes = 4000
	maxFieldRunes       = 200
	maxImages           = 8
	maxImageURLBytes    = 512
	maxExtras           = 20
	maxInspectionBytes  = 16 << 10
)

// Details is a hot-swappable set, reloaded when the crawler republishes. Same
// contract as Explanations and History: readers never block, a broken file
// keeps the previous set serving.
// Empty reports a tombstone: an entry the crawler wrote to record that it
// tried and got nothing, so a dead listing is not re-requested every cycle.
//
// It carries a fetched_at and no content. Serving it as a detail would give the
// page an object full of nulls to render, which is worse than a 404 because the
// UI would open a panel to display nothing.
func (d Detail) Empty() bool {
	return d.Description == nil && d.Color == nil && d.Damages == nil &&
		d.Condition == nil && d.Fuel == nil && d.Body == nil && d.Engine == nil &&
		len(d.Images) == 0 && len(d.Inspection) == 0 && len(d.Extras) == 0
}

type Details struct {
	path string
	cur  atomic.Pointer[detailFile]
	log  *slog.Logger

	mtime atomic.Int64
}

func NewDetails(path string, log *slog.Logger) *Details {
	d := &Details{path: path, log: log}
	d.cur.Store(&detailFile{Details: map[string]Detail{}})
	if _, err := os.Stat(path); err != nil {
		// Said once, at startup, and never by the poll loop. On a first deploy
		// this file does not exist yet — the crawler writes it a cycle later —
		// and a line repeated every thirty seconds for the life of the process
		// is how a log stops being read.
		log.Info("details file absent; detail requests answer 404 until the crawler writes it",
			"path", path)
	}
	d.reloadIfChanged()
	return d
}

// Get returns the detail for "<source>:<source_id>". Not-found is the normal
// case, not an error: details are warmed for the top specs only, so most
// offers in the index will never have one.
func (d *Details) Get(key string) (Detail, bool) {
	v, ok := d.cur.Load().Details[key]
	return v, ok
}

func (d *Details) Count() int      { return len(d.cur.Load().Details) }
func (d *Details) BuiltAt() string { return d.cur.Load().BuiltAt }

// Watch polls for a republished file. Missing stays normal for the life of the
// process: the crawler may never get past a soft block on a given cycle.
func (d *Details) Watch(every time.Duration, stop <-chan struct{}) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			d.reloadIfChanged()
		}
	}
}

func (d *Details) reloadIfChanged() {
	fi, err := os.Stat(d.path)
	if err != nil {
		return
	}
	m := fi.ModTime().UnixNano()
	if m == d.mtime.Load() {
		return
	}
	raw, err := os.ReadFile(d.path)
	if err != nil {
		return
	}
	var f detailFile
	if err := json.Unmarshal(raw, &f); err != nil {
		// A half-written file is a normal thing to catch a crawler mid-rename.
		// Keeping the previous set means the worst case is stale detail, not
		// no detail; the mtime is stored so the same broken bytes are not
		// re-read and re-warned about every thirty seconds.
		d.log.Warn("details reload failed; keeping the previous set",
			"path", d.path, "err", err)
		d.mtime.Store(m)
		return
	}
	capped := make(map[string]Detail, len(f.Details))
	for k, v := range f.Details {
		capped[k] = v.capped()
	}
	f.Details = capped
	d.cur.Store(&f)
	d.mtime.Store(m)
	d.log.Info("details reloaded", "count", len(f.Details), "built_at", f.BuiltAt)
}

// capped returns d with every unbounded field cut to its ceiling, and with the
// two collection fields present-but-empty rather than null, because the UI
// distinguishes "no images" from "no images field".
func (d Detail) capped() Detail {
	d.Description = truncPtr(d.Description, maxDescriptionRunes)
	d.Color = truncPtr(d.Color, maxFieldRunes)
	d.Damages = truncPtr(d.Damages, maxFieldRunes)
	d.Condition = truncPtr(d.Condition, maxFieldRunes)
	d.Fuel = truncPtr(d.Fuel, maxFieldRunes)
	d.Body = truncPtr(d.Body, maxFieldRunes)
	d.Engine = truncPtr(d.Engine, maxFieldRunes)
	d.FetchedAt = truncate(d.FetchedAt, maxFieldRunes)

	imgs := d.Images
	if len(imgs) > maxImages {
		imgs = imgs[:maxImages]
	}
	out := make([]string, 0, len(imgs))
	for _, u := range imgs {
		// A string this long is not an image URL. Dropping it beats echoing
		// whatever the crawler picked up by mistake.
		if len(u) <= maxImageURLBytes {
			out = append(out, u)
		}
	}
	d.Images = out

	extras := make(map[string]string, len(d.Extras))
	if len(d.Extras) > 0 {
		keys := make([]string, 0, len(d.Extras))
		for k := range d.Extras {
			keys = append(keys, k)
		}
		// Sorted before the cut: Go randomises map iteration, so taking an
		// unsorted prefix would drop a different subset on every reload and
		// the same listing would answer differently after a restart.
		sort.Strings(keys)
		if len(keys) > maxExtras {
			keys = keys[:maxExtras]
		}
		for _, k := range keys {
			extras[truncate(k, maxFieldRunes)] = truncate(d.Extras[k], maxFieldRunes)
		}
	}
	d.Extras = extras

	if len(d.Inspection) > maxInspectionBytes {
		// Truncating JSON produces invalid JSON, so an oversized report is
		// dropped whole rather than cut.
		d.Inspection = nil
	}
	return d
}

// truncate cuts s to n runes, not bytes. Every one of these fields is Persian,
// where a byte slice lands mid-codepoint and the browser renders the tail as a
// replacement character.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func truncPtr(p *string, n int) *string {
	if p == nil {
		return nil
	}
	s := truncate(*p, n)
	return &s
}
