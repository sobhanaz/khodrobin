package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

// Explanations are fetched on demand, not folded into /search.
//
// Search answers in about a millisecond and a model answers in seconds, so
// putting them on one path would make every result page wait for text most
// visitors never open. Fetching lazily also means we only pay for explanations
// somebody actually reads, which is the difference between a cost per search
// and a cost per interested user.
type explainClient struct {
	base string
	http *http.Client
}

func newExplainClient() *explainClient {
	base := os.Getenv("KHODROBIN_AI_URL")
	if base == "" {
		base = "http://ai:8000"
	}
	return &explainClient{
		base: strings.TrimRight(base, "/"),
		// Generous: a 7B model on CPU is slow, and the alternative to waiting
		// is a templated sentence the service can already produce itself.
		http: &http.Client{Timeout: 45 * time.Second},
	}
}

func (c *explainClient) explain(ctx context.Context, spec any) (map[string]any, error) {
	body, err := json.Marshal(map[string]any{"spec": spec})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/explain", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

// handleExplain returns the "why this one" text for a single spec.
//
// The AI service being unreachable is not an error the user should see: it
// falls back to a sentence assembled from the spec's own numbers. The response
// always says which path produced it, so the UI can be honest about it and the
// «پشت صحنه» panel can show the difference.
func (s *Server) handleExplain(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.PathValue("key"), "/")
	idx := s.store.Get()

	var spec *interface{}
	for i := range idx.Specs {
		if idx.Specs[i].Key == key {
			var v interface{} = idx.Specs[i]
			spec = &v
			break
		}
	}
	if spec == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "spec not found"})
		return
	}

	start := time.Now()
	out, err := s.ai.explain(r.Context(), *spec)
	ms := time.Since(start).Seconds() * 1000
	if err != nil {
		s.log.Warn("explain.unavailable", "key", key, "err", err)
		writeJSON(w, http.StatusOK, map[string]any{
			"key":    key,
			"text":   "",
			"source": "unavailable",
		})
		return
	}

	out["key"] = key
	out["total_ms"] = ms
	s.log.Info("explain.served",
		"key", key, "source", out["source"], "cached", out["cached"], "total_ms", ms)
	writeJSON(w, http.StatusOK, out)
}
