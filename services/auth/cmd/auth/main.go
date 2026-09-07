// Command auth runs the accounts service: registration, sessions, saved
// searches, price alerts and the contact form.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sobhanaz/khodrobin/auth/internal/handlers"
	"github.com/sobhanaz/khodrobin/auth/internal/mail"
	"github.com/sobhanaz/khodrobin/auth/internal/store"
	"github.com/sobhanaz/khodrobin/auth/internal/tokens"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Error("DATABASE_URL is required")
		os.Exit(1)
	}
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		// A short signing key is a forgeable one, and it fails silently — every
		// token still validates against itself. Refuse to start instead.
		log.Error("JWT_SECRET must be at least 32 characters")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	st, err := store.New(ctx, dsn)
	if err != nil {
		log.Error("connect to database", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	if err := st.Migrate(ctx, store.Schema); err != nil {
		log.Error("apply schema", "err", err)
		os.Exit(1)
	}
	log.Info("schema applied")

	mailer := mail.FromEnv(log)
	if !mailer.Configured() {
		// A warning, not a failure. Search and accounts work without mail; only
		// verification links and alerts wait for it.
		log.Warn("smtp is not configured — verification and alert mail will not send")
	}

	baseURL := os.Getenv("PUBLIC_BASE_URL")
	if baseURL == "" {
		baseURL = "https://khodrobin.noxioai.com"
	}

	api := handlers.New(st, tokens.NewIssuer(secret, "khodrobin"), mailer, baseURL, log)
	mux := http.NewServeMux()
	api.Routes(mux)

	addr := os.Getenv("AUTH_ADDR")
	if addr == "" {
		addr = ":8081"
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		// Longer than the others: argon2id verification is deliberately slow.
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	idle := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		shutdownCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("graceful shutdown failed", "err", err)
		}
		close(idle)
	}()

	log.Info("auth listening", "addr", addr, "mail", mailer.Configured(),
		"base_url", baseURL, "smtp_user", maskUser(os.Getenv("SMTP_USER")))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
	<-idle
}

// maskUser keeps the sending account identifiable in logs without printing it.
func maskUser(s string) string {
	at := strings.LastIndex(s, "@")
	if at <= 1 {
		return "***"
	}
	return s[:1] + "***" + s[at:]
}
