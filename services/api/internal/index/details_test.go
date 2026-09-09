package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeAged replaces the file and stamps a distinct mtime, because
// reloadIfChanged skips a file whose mtime it has already seen and two writes
// inside one test can land on the same filesystem timestamp.
func writeAged(t *testing.T, path, body string, age time.Duration) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := time.Now().Add(-age)
	if err := os.Chtimes(path, ts, ts); err != nil {
		t.Fatal(err)
	}
}

func TestDetailsMissingFile(t *testing.T) {
	d := NewDetails(filepath.Join(t.TempDir(), "details.json"), quiet())
	if _, ok := d.Get("divar:abc"); ok {
		t.Error("a missing file must not report a hit")
	}
	if d.Count() != 0 || d.BuiltAt() != "" {
		t.Errorf("count = %d, built_at = %q, want an empty set", d.Count(), d.BuiltAt())
	}
	// Polling a file that never appears must stay quiet rather than panic.
	d.reloadIfChanged()
}

func TestDetailsCorruptReloadKeepsPreviousSet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "details.json")
	writeAged(t, path, `{"built_at":"2026-09-09T00:00:00Z","details":{"divar:abc":{"color":"سفید"}}}`, time.Minute)
	d := NewDetails(path, quiet())
	if _, ok := d.Get("divar:abc"); !ok {
		t.Fatal("the good file did not load")
	}

	// Truncated mid-object, which is what catching the crawler between write
	// and rename would look like.
	writeAged(t, path, `{"built_at":"2026-09-09T01:00:00Z","details":{"divar:abc":`, 0)
	d.reloadIfChanged()

	got, ok := d.Get("divar:abc")
	if !ok || got.Color == nil || *got.Color != "سفید" {
		t.Fatalf("corrupt reload lost the previous set: %+v (ok=%v)", got, ok)
	}
	if d.BuiltAt() != "2026-09-09T00:00:00Z" {
		t.Errorf("built_at = %q, want the last good file's", d.BuiltAt())
	}
}

func TestDetailCaps(t *testing.T) {
	long := strings.Repeat("ط", maxDescriptionRunes+500)
	extras := map[string]string{}
	for i := range maxExtras + 10 {
		extras[string(rune('a'+i%26))+strings.Repeat("x", i)] = "v"
	}
	imgs := make([]string, 0, maxImages+3)
	for range maxImages + 2 {
		imgs = append(imgs, "https://example.com/1.jpg")
	}
	imgs = append(imgs, "https://example.com/"+strings.Repeat("z", maxImageURLBytes))

	in := Detail{Description: &long, Color: &long, Images: imgs, Extras: extras,
		Inspection: []byte(`{"score":` + strings.Repeat("9", maxInspectionBytes) + `}`)}
	got := in.capped()

	if n := len([]rune(*got.Description)); n != maxDescriptionRunes {
		t.Errorf("description = %d runes, want %d", n, maxDescriptionRunes)
	}
	// Runes, not bytes: a two-byte Persian letter cut by byte count would come
	// back as a replacement character, and the count below would not match.
	if n := len([]rune(*got.Color)); n != maxFieldRunes {
		t.Errorf("color = %d runes, want %d", n, maxFieldRunes)
	}
	if len(got.Images) != maxImages {
		t.Errorf("images = %d, want %d (the over-long URL must be dropped, not counted)",
			len(got.Images), maxImages)
	}
	if len(got.Extras) != maxExtras {
		t.Errorf("extras = %d, want %d", len(got.Extras), maxExtras)
	}
	if got.Inspection != nil {
		t.Error("an oversized inspection report must be dropped whole, not truncated into invalid JSON")
	}

	// Empty in, present-and-empty out: the UI reads these without a nil check.
	empty := Detail{}.capped()
	if empty.Images == nil || empty.Extras == nil {
		t.Errorf("images = %v, extras = %v, want empty collections not nil", empty.Images, empty.Extras)
	}
}
