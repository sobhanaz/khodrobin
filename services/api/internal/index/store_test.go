package index

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func write(t *testing.T, path string, specs int) {
	t.Helper()
	idx := Index{BuiltAt: time.Now().Format(time.RFC3339)}
	for i := 0; i < specs; i++ {
		idx.Specs = append(idx.Specs, Spec{Key: string(rune('a' + i))})
	}
	b, err := json.Marshal(idx)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestReloadPicksUpANewIndex(t *testing.T) {
	p := filepath.Join(t.TempDir(), "index.json")
	write(t, p, 2)
	idx, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	s := NewStore(p, idx, quiet())

	write(t, p, 5)
	// Force a distinct mtime; some filesystems have coarse resolution.
	future := time.Now().Add(2 * time.Second)
	_ = os.Chtimes(p, future, future)

	s.reloadIfChanged()
	if got := len(s.Get().Specs); got != 5 {
		t.Errorf("specs = %d, want 5", got)
	}
}

func TestABrokenRebuildKeepsServingTheOldIndex(t *testing.T) {
	// The important property: a bad crawl must not take the site down.
	p := filepath.Join(t.TempDir(), "index.json")
	write(t, p, 3)
	idx, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	s := NewStore(p, idx, quiet())

	if err := os.WriteFile(p, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	_ = os.Chtimes(p, future, future)

	s.reloadIfChanged()
	if got := len(s.Get().Specs); got != 3 {
		t.Errorf("specs = %d, want the previous 3", got)
	}
}

func TestAnEmptyIndexIsRejected(t *testing.T) {
	// An index with no specs parses fine but means the crawler produced
	// nothing. Serving it would silently empty the site.
	p := filepath.Join(t.TempDir(), "index.json")
	write(t, p, 4)
	idx, _ := Load(p)
	s := NewStore(p, idx, quiet())

	if err := os.WriteFile(p, []byte(`{"specs":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	_ = os.Chtimes(p, future, future)

	s.reloadIfChanged()
	if got := len(s.Get().Specs); got != 4 {
		t.Errorf("specs = %d, want the previous 4", got)
	}
}

func TestUnchangedFileIsNotReloaded(t *testing.T) {
	p := filepath.Join(t.TempDir(), "index.json")
	write(t, p, 2)
	idx, _ := Load(p)
	s := NewStore(p, idx, quiet())
	before := s.Get()
	s.reloadIfChanged()
	if s.Get() != before {
		t.Error("index pointer changed although the file did not")
	}
}
