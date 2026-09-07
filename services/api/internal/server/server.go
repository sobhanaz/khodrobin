// Package server wires the HTTP surface: the JSON API and the embedded UI.
package server

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sobhanaz/khodrobin/api/internal/index"
	"github.com/sobhanaz/khodrobin/api/internal/search"
)

//go:embed ui/*
var uiFS embed.FS

type Server struct {
	store *index.Store
	log   *slog.Logger
	mux   *http.ServeMux
	ai    *explainClient
}

func New(store *index.Store, log *slog.Logger) *Server {
	s := &Server{store: store, log: log, mux: http.NewServeMux(), ai: newExplainClient()}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/v1/search", s.handleSearch)
	s.mux.HandleFunc("GET /api/v1/specs/{key...}", s.handleSpec)
	s.mux.HandleFunc("GET /api/v1/stats", s.handleStats)
	s.mux.HandleFunc("GET /api/v1/explain/{key...}", s.handleExplain)
	ui, err := fs.Sub(uiFS, "ui")
	if err != nil {
		// Embedded at compile time: if this fails the binary is malformed.
		panic("embedded ui missing: " + err.Error())
	}
	s.mux.Handle("GET /", http.FileServerFS(ui))
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

// handleSearch is the main path: parse the query, filter, rank, respond — and
// report how long each stage took, because the timings are part of the demo.
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	q := r.URL.Query().Get("q")
	mode := r.URL.Query().Get("mode")
	limit := 24
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	parseStart := time.Now()
	idx := s.store.Get()
	in := search.ParseQuery(q, idx.Vocab)
	parseMs := time.Since(parseStart).Seconds() * 1000

	rankStart := time.Now()
	res := search.Run(idx, in, mode, limit)
	rankMs := time.Since(rankStart).Seconds() * 1000

	totalMs := time.Since(start).Seconds() * 1000
	w.Header().Set("Server-Timing",
		"parse;dur="+strconv.FormatFloat(parseMs, 'f', 2, 64)+
			", rank;dur="+strconv.FormatFloat(rankMs, 'f', 2, 64))
	w.Header().Set("X-Intent-Source", in.Source)

	writeJSON(w, http.StatusOK, map[string]any{
		"query":  q,
		"result": res,
		"timing": map[string]float64{"parse_ms": parseMs, "rank_ms": rankMs, "total_ms": totalMs},
	})

	s.log.Info("search.served",
		"q", q, "mode", res.Mode, "brand", in.Brand, "model", in.Model,
		"total", res.Total, "total_ms", totalMs)
}

func (s *Server) handleSpec(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.PathValue("key"), "/")
	idx := s.store.Get()
	for i := range idx.Specs {
		if idx.Specs[i].Key == key {
			writeJSON(w, http.StatusOK, idx.Specs[i])
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "spec not found"})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	idx := s.store.Get()
	writeJSON(w, http.StatusOK, map[string]any{
		"stats":     idx.Stats,
		"built_at":  idx.BuiltAt,
		"loaded_at": idx.LoadedAt,
		"offers":    idx.TotalOffers(),
	})
}
