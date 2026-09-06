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

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health.Live)
	mux.HandleFunc("GET /readyz", health.Ready)

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
