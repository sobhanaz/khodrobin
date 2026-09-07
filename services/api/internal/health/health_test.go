package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type snap struct {
	specs int
	built time.Time
}

func (s snap) Specs() int         { return s.specs }
func (s snap) BuiltAt() time.Time { return s.built }

func TestLiveReturnsOK(t *testing.T) {
	rec := httptest.NewRecorder()
	Live(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func run(t *testing.T, s snap) (int, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	Ready(s)(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return rec.Code, body
}

func TestReadyWithFreshData(t *testing.T) {
	code, body := run(t, snap{specs: 1200, built: time.Now().Add(-30 * time.Minute)})
	if code != http.StatusOK {
		t.Errorf("status = %d, want 200", code)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %v, want ok", body["status"])
	}
}

func TestAnEmptyIndexIsNotReady(t *testing.T) {
	// Serving zero results while reporting healthy is worse than reporting
	// failure: nobody investigates a green probe.
	code, body := run(t, snap{specs: 0, built: time.Now()})
	if code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", code)
	}
	checks := body["checks"].(map[string]any)
	if checks["index"] != "empty" {
		t.Errorf("checks.index = %v, want empty", checks["index"])
	}
}

func TestAStaleIndexIsNotReady(t *testing.T) {
	// The crawler runs every three hours, so past six hours two cycles have
	// been missed and the prices on screen are not current.
	code, body := run(t, snap{specs: 1200, built: time.Now().Add(-7 * time.Hour)})
	if code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", code)
	}
	checks := body["checks"].(map[string]any)
	if checks["freshness"] != "stale" {
		t.Errorf("checks.freshness = %v, want stale", checks["freshness"])
	}
}

func TestJustInsideTheStalenessWindowIsReady(t *testing.T) {
	code, _ := run(t, snap{specs: 1200, built: time.Now().Add(-StaleAfter + time.Minute)})
	if code != http.StatusOK {
		t.Errorf("status = %d, want 200", code)
	}
}

func TestAnUnknownBuildTimeIsNotReady(t *testing.T) {
	code, _ := run(t, snap{specs: 1200})
	if code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", code)
	}
}
