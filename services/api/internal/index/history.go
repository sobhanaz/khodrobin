package index

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync/atomic"
	"time"
)

// Price history, appended by the crawler each cycle and served from memory.
//
// The file keeps its points as {t, m, n} because it is rewritten eight times a
// day for thousands of specs and the short keys keep it small; the API answers
// with spelled-out names because a response is a public contract. The mapping
// happens once, in the handler.
type HistoryPoint struct {
	T string `json:"t"`
	M int64  `json:"m"`
	N int    `json:"n"`
}

type historyFile struct {
	UpdatedAt string                    `json:"updated_at"`
	Specs     map[string][]HistoryPoint `json:"specs"`
}

// History is a hot-swappable set, reloaded when the crawler republishes. Same
// contract as Explanations: readers never block, a broken file keeps the
// previous set serving.
type History struct {
	path string
	cur  atomic.Pointer[historyFile]
	log  *slog.Logger

	mtime atomic.Int64
}

func NewHistory(path string, log *slog.Logger) *History {
	h := &History{path: path, log: log}
	h.cur.Store(&historyFile{Specs: map[string][]HistoryPoint{}})
	h.reloadIfChanged()
	return h
}

// Get returns the recorded points for a spec. Nil is normal: a spec the
// crawler has never found three reliable offers for has no history yet.
func (h *History) Get(key string) []HistoryPoint {
	return h.cur.Load().Specs[key]
}

func (h *History) UpdatedAt() string { return h.cur.Load().UpdatedAt }

// Watch polls for a republished file. Missing is normal, not an error: the
// history feature ships before the crawler has written its first cycle, so on
// a fresh volume every spec simply has no points yet.
func (h *History) Watch(every time.Duration, stop <-chan struct{}) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			h.reloadIfChanged()
		}
	}
}

func (h *History) reloadIfChanged() {
	fi, err := os.Stat(h.path)
	if err != nil {
		return
	}
	m := fi.ModTime().UnixNano()
	if m == h.mtime.Load() {
		return
	}
	raw, err := os.ReadFile(h.path)
	if err != nil {
		return
	}
	var f historyFile
	if err := json.Unmarshal(raw, &f); err != nil {
		h.log.Warn("history reload failed; keeping the previous set",
			"path", h.path, "err", err)
		h.mtime.Store(m)
		return
	}
	if f.Specs == nil {
		f.Specs = map[string][]HistoryPoint{}
	}
	h.cur.Store(&f)
	h.mtime.Store(m)
	h.log.Info("history reloaded", "specs", len(f.Specs), "updated_at", f.UpdatedAt)
}
