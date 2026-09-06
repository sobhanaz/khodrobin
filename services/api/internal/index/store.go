package index

import (
	"log/slog"
	"os"
	"sync/atomic"
	"time"
)

// Store holds the current index and swaps it atomically when the crawler
// publishes a new one.
//
// Readers never block and never see a half-written index: the pointer swap is
// atomic and each Index is immutable once built. A failed reload is a no-op —
// the previous index keeps serving, which is the behaviour we want when a
// crawl produces garbage at 3am.
type Store struct {
	path string
	cur  atomic.Pointer[Index]
	log  *slog.Logger

	mtime atomic.Int64
}

func NewStore(path string, idx *Index, log *slog.Logger) *Store {
	s := &Store{path: path, log: log}
	s.cur.Store(idx)
	if fi, err := os.Stat(path); err == nil {
		s.mtime.Store(fi.ModTime().UnixNano())
	}
	return s
}

// Get returns the index currently being served.
func (s *Store) Get() *Index { return s.cur.Load() }

// Watch polls the index file and reloads it when its modification time moves.
//
// Polling rather than inotify: the file arrives via a Docker volume shared with
// the crawler, where filesystem events are unreliable across the boundary, and
// a stat every 30 seconds costs nothing.
func (s *Store) Watch(every time.Duration, stop <-chan struct{}) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			s.reloadIfChanged()
		}
	}
}

func (s *Store) reloadIfChanged() {
	fi, err := os.Stat(s.path)
	if err != nil {
		return
	}
	m := fi.ModTime().UnixNano()
	if m == s.mtime.Load() {
		return
	}

	// The crawler writes to a temp file and renames, so seeing a new mtime
	// should mean a complete file. If it does not parse, keep the old one.
	idx, err := Load(s.path)
	if err != nil {
		s.log.Warn("index reload failed; keeping the previous index",
			"path", s.path, "err", err)
		// Record the mtime anyway so a permanently broken file is not retried
		// every tick.
		s.mtime.Store(m)
		return
	}

	prev := s.cur.Load()
	s.cur.Store(idx)
	s.mtime.Store(m)
	s.log.Info("index reloaded",
		"specs", len(idx.Specs), "offers", idx.TotalOffers(),
		"was_specs", len(prev.Specs), "built_at", idx.BuiltAt)
}
