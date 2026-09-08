package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sobhanaz/khodrobin/api/internal/index"
)

const knownKey = "peugeot/206/panorama/mt/1405/0"

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// newTestServer wires a server around a one-spec index and a history file that
// may not exist — which is exactly the state of a first deploy.
func newTestServer(t *testing.T, historyJSON string) *Server {
	t.Helper()
	dir := t.TempDir()
	historyPath := filepath.Join(dir, "history.json")
	if historyJSON != "" {
		if err := os.WriteFile(historyPath, []byte(historyJSON), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	idx := &index.Index{BuiltAt: "2026-01-01T00:00:00Z",
		Specs: []index.Spec{{Key: knownKey, Brand: "peugeot", Model: "206"}}}
	store := index.NewStore(filepath.Join(dir, "index.json"), idx, quiet())
	explains := index.NewExplanations(filepath.Join(dir, "explanations.json"), quiet())
	history := index.NewHistory(historyPath, quiet())
	return New(store, explains, history, quiet())
}

func TestHistoryEndpoint(t *testing.T) {
	recorded := `{"updated_at":"2026-09-08T00:00:00Z","specs":{"` + knownKey +
		`":[{"t":"2026-09-07T00:00:00Z","m":600000000,"n":4}]}}`

	cases := []struct {
		name        string
		historyJSON string
		path        string
		wantCode    int
		wantBody    string
	}{
		{
			// The file does not exist on first deploy; a known spec must still
			// answer 200 with a present-and-empty points list, not an error.
			name:     "known key with no history file",
			path:     "/api/v1/history/" + knownKey,
			wantCode: 200,
			wantBody: `"points":[]`,
		},
		{
			name:     "unknown key is a 404 even with no file",
			path:     "/api/v1/history/peugeot/207/nope/mt/1400/0",
			wantCode: 404,
		},
		{
			// Pins the field mapping: the file's compact {t,m,n} must come out
			// as the contract's {t, median, offers}.
			name:        "recorded points come back under contract names",
			historyJSON: recorded,
			path:        "/api/v1/history/" + knownKey,
			wantCode:    200,
			wantBody:    `"median":600000000`,
		},
		{
			name:        "a key the index dropped is 404 even if history remembers it",
			historyJSON: `{"updated_at":"x","specs":{"gone/spec/na/mt/1390/0":[]}}`,
			path:        "/api/v1/history/gone/spec/na/mt/1390/0",
			wantCode:    404,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newTestServer(t, c.historyJSON)
			rec := httptest.NewRecorder()
			s.ServeHTTP(rec, httptest.NewRequest("GET", c.path, nil))
			if rec.Code != c.wantCode {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, c.wantCode, rec.Body.String())
			}
			if c.wantBody != "" && !strings.Contains(rec.Body.String(), c.wantBody) {
				t.Errorf("body %s does not contain %s", rec.Body.String(), c.wantBody)
			}
		})
	}
}

func TestSearchHandlerAppliesOverrides(t *testing.T) {
	// The merge itself is tested in the search package; this pins that the
	// handler actually routes URL parameters through it and reports the
	// merged intent, so a shared URL renders the same chips for everyone.
	s := newTestServer(t, "")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest("GET",
		"/api/v1/search?q=%D8%B2%DB%8C%D8%B1+%DB%B5%DB%B0%DB%B0&price_max=700000000", nil))
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Result struct {
			Intent struct {
				PriceMax int64 `json:"price_max"`
			} `json:"intent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Result.Intent.PriceMax != 700_000_000 {
		t.Errorf("intent.price_max = %d, want the override 700000000", body.Result.Intent.PriceMax)
	}
}
