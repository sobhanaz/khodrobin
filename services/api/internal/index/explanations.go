package index

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync/atomic"
	"time"
)

// Precomputed explanations, written by the crawler and served from memory.
//
// Generating one takes about ten seconds on CPU, which is fine on a schedule
// and unacceptable in a request. This is the same trade the rest of the
// pipeline makes: cluster, normalize and now explain offline, so the request
// path only reads.
type Explanation struct {
	Text        string `json:"text"`
	Source      string `json:"source"`
	Fingerprint string `json:"fingerprint"`
	// float64, not int. The guard reports decimals — «۱۱.۶٪» is one number, not
	// 116 — and an int here made every reload fail with an unmarshal error,
	// which the reload path then handled gracefully by keeping the previous
	// set. The previous set was empty, so the service served zero precomputed
	// explanations indefinitely while logging a warning nobody was reading.
	RejectedNumbers []float64      `json:"rejected_numbers"`
	RejectedTopics  []string       `json:"rejected_topics"`
	Usage           map[string]any `json:"usage"`
}

type explanationFile struct {
	BuiltAt      string                 `json:"built_at"`
	Count        int                    `json:"count"`
	Explanations map[string]Explanation `json:"explanations"`
}

// Explanations is a hot-swappable set, reloaded when the crawler republishes.
type Explanations struct {
	path string
	cur  atomic.Pointer[explanationFile]
	log  *slog.Logger

	mtime atomic.Int64
}

func NewExplanations(path string, log *slog.Logger) *Explanations {
	e := &Explanations{path: path, log: log}
	e.cur.Store(&explanationFile{Explanations: map[string]Explanation{}})
	e.reloadIfChanged()
	return e
}

// Get returns the precomputed explanation for a spec, if there is one.
func (e *Explanations) Get(key string) (Explanation, bool) {
	f := e.cur.Load()
	v, ok := f.Explanations[key]
	return v, ok
}

func (e *Explanations) Count() int      { return len(e.cur.Load().Explanations) }
func (e *Explanations) BuiltAt() string { return e.cur.Load().BuiltAt }

// Watch polls for a republished file. Missing is normal, not an error: on a
// fresh volume nothing has been warmed yet and every request falls through to
// on-demand generation.
func (e *Explanations) Watch(every time.Duration, stop <-chan struct{}) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			e.reloadIfChanged()
		}
	}
}

func (e *Explanations) reloadIfChanged() {
	fi, err := os.Stat(e.path)
	if err != nil {
		return
	}
	m := fi.ModTime().UnixNano()
	if m == e.mtime.Load() {
		return
	}
	raw, err := os.ReadFile(e.path)
	if err != nil {
		return
	}
	var f explanationFile
	if err := json.Unmarshal(raw, &f); err != nil {
		e.log.Warn("explanations reload failed; keeping the previous set",
			"path", e.path, "err", err)
		e.mtime.Store(m)
		return
	}
	if f.Explanations == nil {
		f.Explanations = map[string]Explanation{}
	}
	e.cur.Store(&f)
	e.mtime.Store(m)
	e.log.Info("explanations reloaded", "count", len(f.Explanations), "built_at", f.BuiltAt)
}
