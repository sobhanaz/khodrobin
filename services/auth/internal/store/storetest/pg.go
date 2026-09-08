// Package storetest gives the store and handler tests a real Postgres.
//
// The rules that make this list legal are written in SQL, not in Go: the
// ON CONFLICT clause that resets a returning subscriber, the COALESCE that
// makes unsubscribing idempotent, the WHERE that keeps unconfirmed addresses
// out of the export. A hand-written fake store would only prove that the fake
// agrees with itself — and the first time one of those predicates was typed
// wrong, every test would still pass.
//
// So the tests talk to a server. Not a container and not a service that has to
// be running first: initdb the developer already has, a data directory in a
// temp folder, fsync off, thrown away at the end. Where there is no initdb —
// most CI images — TEST_DATABASE_URL is used instead, and failing both, the
// tests that need a database skip rather than fail.
package storetest

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/sobhanaz/khodrobin/auth/internal/store"
)

var (
	dsn      string
	skipWhy  = "no database: set TEST_DATABASE_URL, or install postgres so initdb is on PATH"
	teardown func()
)

// Run boots the cluster around a package's tests. Call it from TestMain.
func Run(m *testing.M) int {
	if err := start(); err != nil {
		skipWhy = "no database: " + err.Error()
	}
	code := m.Run()
	if teardown != nil {
		teardown()
	}
	return code
}

func start() error {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		dsn = u
		return nil
	}
	initdb, err := exec.LookPath("initdb")
	if err != nil {
		return fmt.Errorf("initdb not on PATH")
	}

	dir, err := os.MkdirTemp("", "kbpg")
	if err != nil {
		return err
	}
	// A unix socket path is capped around 100 bytes by the kernel, and some
	// TMPDIRs are long enough on their own to blow through it. /tmp always is
	// not.
	if len(dir) > 70 {
		_ = os.RemoveAll(dir)
		if dir, err = os.MkdirTemp("/tmp", "kbpg"); err != nil {
			return err
		}
	}
	data := filepath.Join(dir, "data")

	// LC_ALL is not politeness. Without it the macOS build of postgres detects
	// that the postmaster went multithreaded during locale lookup and refuses
	// to start, with a message that says nothing about locales being the cause.
	env := append(os.Environ(), "LC_ALL=C", "LANG=C")

	cmd := exec.Command(initdb, "-D", data, "-U", "postgres", "--auth=trust", "-E", "UTF8", "--no-sync")
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.RemoveAll(dir)
		return fmt.Errorf("initdb: %v: %s", err, out)
	}

	// fsync off because the whole cluster is deleted in a few seconds; durability
	// here would only buy slower tests.
	server := exec.Command(filepath.Join(filepath.Dir(initdb), "postgres"),
		"-D", data, "-k", dir, "-h", "", "-c", "fsync=off", "-c", "full_page_writes=off")
	server.Env = env
	if err := server.Start(); err != nil {
		_ = os.RemoveAll(dir)
		return fmt.Errorf("start postgres: %w", err)
	}
	teardown = func() {
		// SIGINT is postgres's fast shutdown, and os.Interrupt spells it without
		// dragging syscall into a test helper.
		_ = server.Process.Signal(os.Interrupt)
		_ = server.Wait()
		_ = os.RemoveAll(dir)
	}

	candidate := "postgres://postgres@/postgres?host=" + url.QueryEscape(dir)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for {
		st, err := store.New(ctx, candidate)
		if err == nil {
			st.Close()
			dsn = candidate
			return nil
		}
		select {
		case <-ctx.Done():
			teardown()
			teardown = nil
			return fmt.Errorf("postgres never accepted a connection: %w", err)
		case <-time.After(200 * time.Millisecond):
		}
	}
}

// DSN is for the one assertion the store's own API cannot make: that
// marketing_consent_at is set exactly when the flag is. A column with no reader
// is not worth a method on Store, but it is worth a test.
func DSN() string { return dsn }

// New returns a migrated, empty store, or skips the test when no server could
// be reached.
func New(t *testing.T) *store.Store {
	t.Helper()
	if dsn == "" {
		t.Skip(skipWhy)
	}
	ctx := context.Background()
	st, err := store.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx, store.Schema); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// One cluster serves the whole package, so each test starts from empty
	// rather than from whatever the previous one left behind. Migrate is the
	// store's only raw-SQL door and a second one is not worth widening the
	// package's API for.
	if err := st.Migrate(ctx, `TRUNCATE users, subscribers, sessions, tokens,
		saved_searches, alerts_sent, contact_messages RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return st
}
