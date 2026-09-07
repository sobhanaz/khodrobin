// Package health serves the liveness and readiness probes.
//
// The distinction is load-bearing here. Live answers "the process is up" and
// nothing more. Ready answers "this process can serve a useful response", which
// for a search product means it holds data that is both present and recent.
//
// Ready began as a hardcoded 200 with an empty checks map, which made it
// indistinguishable from Live and meant the deploy smoke test was asserting a
// constant. A probe that cannot fail is not a probe.
package health

import (
	"encoding/json"
	"net/http"
	"time"
)

// StaleAfter is how old an index may get before readiness fails.
//
// The crawler runs every three hours, so six hours means two consecutive cycles
// were missed — long enough to rule out a slow crawl, short enough that nobody
// browses day-old prices believing they are current.
const StaleAfter = 6 * time.Hour

// Snapshot is what readiness needs to know about the served index. It is an
// interface so health does not import index and index does not import health.
type Snapshot interface {
	Specs() int
	BuiltAt() time.Time
}

func write(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

// Live reports that the process is running.
func Live(w http.ResponseWriter, _ *http.Request) {
	write(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready reports whether this process can serve a useful search.
//
// It returns 503 with the failing check named, so an external monitor sees the
// reason rather than a bare failure — and so a bad deploy is caught by the
// smoke test instead of by a visitor.
func Ready(snap Snapshot) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		checks := map[string]any{}
		ok := true

		specs := snap.Specs()
		checks["index_specs"] = specs
		if specs == 0 {
			checks["index"] = "empty"
			ok = false
		} else {
			checks["index"] = "ok"
		}

		built := snap.BuiltAt()
		if built.IsZero() {
			checks["freshness"] = "unknown build time"
			ok = false
		} else {
			age := time.Since(built)
			checks["index_age_seconds"] = int(age.Seconds())
			if age > StaleAfter {
				checks["freshness"] = "stale"
				ok = false
			} else {
				checks["freshness"] = "ok"
			}
		}

		status, code := "ok", http.StatusOK
		if !ok {
			status, code = "not ready", http.StatusServiceUnavailable
		}
		write(w, code, map[string]any{"status": status, "checks": checks})
	}
}
