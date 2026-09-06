// Package health serves the liveness and readiness probes.
//
// Live answers "the process is up". Ready answers "the process can serve
// traffic" — it will grow dependency checks (Postgres, Elasticsearch, Redis)
// as those are wired in, which is why they are separate endpoints from the
// start rather than one shared handler.
package health

import (
	"encoding/json"
	"net/http"
)

func write(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

// Live reports that the process is running.
func Live(w http.ResponseWriter, _ *http.Request) {
	write(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready reports that the process can serve traffic.
func Ready(w http.ResponseWriter, _ *http.Request) {
	write(w, http.StatusOK, map[string]any{"status": "ok", "checks": map[string]string{}})
}
