// Package server wires the HTTP surface: the JSON API and the embedded UI.
package server

import (
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sobhanaz/khodrobin/api/internal/index"
	"github.com/sobhanaz/khodrobin/api/internal/lookup"
	"github.com/sobhanaz/khodrobin/api/internal/search"
)

//go:embed ui/*
var uiFS embed.FS

type Server struct {
	store    *index.Store
	explains *index.Explanations
	history  *index.History
	details  *index.Details
	log      *slog.Logger
	mux      *http.ServeMux
	ai       *explainClient
}

func New(store *index.Store, explains *index.Explanations, history *index.History, details *index.Details, log *slog.Logger) *Server {
	s := &Server{store: store, explains: explains, history: history, details: details, log: log,
		mux: http.NewServeMux(), ai: newExplainClient()}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/v1/search", s.handleSearch)
	s.mux.HandleFunc("POST /api/v1/lookup", s.handleLookup)
	s.mux.HandleFunc("GET /api/v1/specs/{key...}", s.handleSpec)
	s.mux.HandleFunc("GET /api/v1/stats", s.handleStats)
	s.mux.HandleFunc("GET /api/v1/explain/{key...}", s.handleExplain)
	s.mux.HandleFunc("GET /api/v1/history/{key...}", s.handleHistory)
	s.mux.HandleFunc("GET /api/v1/details/{source}/{id}", s.handleDetail)
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
	params := r.URL.Query()
	q := params.Get("q")
	mode := params.Get("mode")
	limit := 24
	if v := params.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	parseStart := time.Now()
	idx := s.store.Get()
	// Explicit filter parameters are merged after the parse, field by field,
	// and the merged intent goes down the exact same Run path a parsed one
	// does. One filtering implementation; the URL just gets the last word.
	in := search.Override(search.ParseQuery(q, idx.Vocab), params, idx.Vocab)
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

// The history file stores points as {t, m, n} to stay small; the response
// spells the names out because it is the public contract.
type historyPoint struct {
	T      string `json:"t"`
	Median int64  `json:"median"`
	Offers int    `json:"offers"`
}

// handleHistory serves the recorded median-price points for one spec.
//
// 404 means the *index* does not know the key. A known spec with no recorded
// points is a 200 with an empty list: the feature ships before the crawler has
// written its first history cycle, and "no history yet" is an answer, not an
// error.
func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.PathValue("key"), "/")
	idx := s.store.Get()
	known := false
	for i := range idx.Specs {
		if idx.Specs[i].Key == key {
			known = true
			break
		}
	}
	if !known {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "spec not found"})
		return
	}
	pts := s.history.Get(key)
	// make, not var: a spec with no points must serialise as [], because the
	// chart code distinguishes "no data yet" from a field that is missing.
	out := make([]historyPoint, 0, len(pts))
	for _, p := range pts {
		out = append(out, historyPoint{T: p.T, Median: p.M, Offers: p.N})
	}
	writeJSON(w, http.StatusOK, map[string]any{"key": key, "points": out})
}

// handleLookup answers the reverse question: given a listing URL, where else
// is this car being sold, and is the pasted one the expensive copy?
//
// It is POST with the URL in a JSON body rather than GET with a query param,
// because a listing URL pasted into an access log or a referrer header is a
// URL that belongs to somebody's ad; the body keeps it out of both.
func (s *Server) handleLookup(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<10)).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid_json", "message_fa": "درخواست نامعتبر است.",
		})
		return
	}

	source, id, err := lookup.Parse(body.URL)
	if err != nil {
		// Unknown-host and known-host-but-wrong-shape are different mistakes:
		// one needs "we only know these five sites", the other needs "this is
		// not an ad page". Both are 422 — the request is well-formed, the
		// content is not usable.
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"error":      "unknown_listing_url",
			"message_fa": "این نشانی را نمی‌شناسم. لینک آگهی را از یکی از این سایت‌ها بفرست: دیوار، باما، همراه‌مکانیک، خودرو۴۵ یا شیپور.",
		})
		return
	}

	idx := s.store.Get()
	res := lookup.Find(idx, source, id)
	if res == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":      "not_in_index",
			"message_fa": "این آگهی توی نمایه‌ی من نیست؛ یا تازه آگهی شده، یا حذفش کرده‌اند، یا دور بعدی خزیده می‌شود.",
		})
		return
	}

	ms := time.Since(start).Seconds() * 1000
	s.log.Info("lookup.served", "source", source, "id", id,
		"spec", res.Spec.Key, "other_sources", res.OtherSources, "total_ms", ms)
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	idx := s.store.Get()
	writeJSON(w, http.StatusOK, map[string]any{
		"stats":     idx.Stats,
		"built_at":  idx.BuiltAt,
		"loaded_at": idx.LoadedAt,
		"offers":    idx.TotalOffers(),
		"explanations": map[string]any{
			"warmed":   s.explains.Count(),
			"built_at": s.explains.BuiltAt(),
		},
	})
}

// handleDetail serves the per-listing extras the crawler pulled from a source's
// own ad page: the seller's description, colour, damage notes, gallery.
//
// Everything here is a map lookup against an in-memory file. The two path
// segments are never joined into a path, opened, or fetched — the source is
// checked against the five names the crawler knows and the id is only ever a
// map key, so there is nothing for a crafted segment to reach. An unknown
// source is a 404 for the same reason an unknown id is: the caller asked for a
// detail that does not exist, and telling them which half of the key was wrong
// only helps someone probing.
func (s *Server) handleDetail(w http.ResponseWriter, r *http.Request) {
	source := r.PathValue("source")
	id := r.PathValue("id")
	if !lookup.KnownSource(source) || id == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "detail not found"})
		return
	}
	key := source + ":" + id
	d, ok := s.details.Get(key)
	if ok && d.Empty() {
		// A tombstone. The crawler records that it tried and got nothing so a
		// dead listing is not re-fetched every cycle; that is a bookkeeping
		// entry, not a detail, and the caller must not be handed an object of
		// nulls to render a panel from.
		ok = false
	}
	if !ok {
		// The normal miss. Details are warmed for the top specs only, and the
		// file does not exist at all until the crawler's first detail cycle.
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "detail not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"key": key, "detail": d})
}
