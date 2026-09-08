package alerts

import (
	"testing"
)

func TestPercentChangeIsSignedSoDirectionSurvives(t *testing.T) {
	// A buyer cares which way it moved. An unsigned magnitude would mail
	// "the price changed by 12%" and make them open it to find out whether
	// that was good news.
	for _, tc := range []struct {
		name     string
		old, now int64
		want     float64
	}{
		{"came down", 500_000_000, 440_000_000, -12},
		{"went up", 500_000_000, 560_000_000, 12},
		{"unchanged", 500_000_000, 500_000_000, 0},
		// A zero baseline is not a 100% rise from nothing; it means there was
		// no observation, and dividing by it would produce Inf.
		{"no baseline", 0, 500_000_000, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := percentChange(tc.old, tc.now)
			if got != tc.want {
				t.Fatalf("percentChange(%d, %d) = %v, want %v", tc.old, tc.now, got, tc.want)
			}
		})
	}
}

func TestPercentChangeNeverProducesInfinityOrNaN(t *testing.T) {
	// Guarding this explicitly because the value goes straight into a subject
	// line, and "قیمت +Inf٪ تغییر کرد" is the kind of thing that ships.
	for _, old := range []int64{0} {
		v := percentChange(old, 900_000_000)
		if v != v || v > 1e9 || v < -1e9 {
			t.Fatalf("percentChange(%d, 900000000) = %v, want a finite number", old, v)
		}
	}
}
