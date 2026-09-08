// Command api serves the KhodroBin core API: search, ranking, caching and metrics.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sobhanaz/khodrobin/api/internal/health"
	"github.com/sobhanaz/khodrobin/api/internal/index"
	"github.com/sobhanaz/khodrobin/api/internal/server"
)

func main() {
	// The image is distroless, so it has no shell and no curl. Docker's
	// healthcheck therefore re-executes this binary with -healthcheck, which
	// probes the live server over HTTP and exits with the verdict.
	probe := flag.Bool("healthcheck", false, "probe a running server and exit")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	addr := os.Getenv("KHODROBIN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	if *probe {
		os.Exit(healthcheck(addr))
	}

	indexPath := os.Getenv("KHODROBIN_INDEX")
	if indexPath == "" {
		indexPath = "/data/index.json"
	}
	// The live index lives on a volume the crawler writes to. On a cold start
	// that volume is empty, so fall back to the snapshot baked into the image:
	// the site is never blank, it is just as fresh as the last release.
	fallback := os.Getenv("KHODROBIN_INDEX_FALLBACK")
	if fallback == "" {
		fallback = "/seed/index.json"
	}
	idx, err := index.Load(indexPath)
	if err != nil {
		log.Warn("live index unavailable; falling back to the baked snapshot",
			"path", indexPath, "err", err)
		idx, err = index.Load(fallback)
	}
	if err != nil {
		// An API with no index has nothing to say. Reporting healthy while
		// serving zero results would be worse than refusing to start.
		log.Error("cannot load any index", "live", indexPath, "fallback", fallback, "err", err)
		os.Exit(1)
	}
	log.Info("index loaded",
		"path", indexPath, "specs", len(idx.Specs), "offers", idx.TotalOffers(),
		"built_at", idx.BuiltAt)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health.Live)
	// The crawler republishes the index on a schedule into a shared volume;
	// the API picks it up without a restart.
	store := index.NewStore(indexPath, idx, log)
	stopWatch := make(chan struct{})
	go store.Watch(30*time.Second, stopWatch)

	explanationsPath := os.Getenv("KHODROBIN_EXPLANATIONS")
	if explanationsPath == "" {
		explanationsPath = "/data/explanations.json"
	}
	explains := index.NewExplanations(explanationsPath, log)
	go explains.Watch(30*time.Second, stopWatch)
	log.Info("explanations loaded", "path", explanationsPath, "count", explains.Count())

	// History sits beside the index on the same crawler volume. Missing is
	// normal on first deploy: the endpoint answers empty lists until the
	// crawler writes its first cycle.
	historyPath := os.Getenv("KHODROBIN_HISTORY")
	if historyPath == "" {
		historyPath = "/data/history.json"
	}
	history := index.NewHistory(historyPath, log)
	go history.Watch(30*time.Second, stopWatch)

	// Readiness needs the store, so it is registered after the store exists.
	mux.HandleFunc("GET /readyz", health.Ready(store))
	mux.Handle("/", server.New(store, explains, history, log))

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Shut down on SIGINT/SIGTERM so `docker compose down` does not sever
	// in-flight requests.
	idle := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Error("graceful shutdown failed", "err", err)
		}
		close(stopWatch)
		close(idle)
	}()

	log.Info("api listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
	<-idle
}

// healthcheck probes a running server's liveness endpoint. It returns a process
// exit code: 0 when the server answers 200, 1 otherwise.
func healthcheck(addr string) int {
	host := addr
	if len(host) > 0 && host[0] == ':' {
		host = "127.0.0.1" + host
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + host + "/healthz")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
