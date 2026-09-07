package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sobhanaz/khodrobin/auth/internal/tokens"
)

func TestEmailValidation(t *testing.T) {
	good := []string{"a@b.co", "sobhandevuk@gmail.com", " Sobhan@Gmail.com "}
	bad := []string{"", "no-at-sign", "a@b", "a@", "@b.co", "a b@c.co"}
	for _, s := range good {
		if !validEmail(s) {
			t.Errorf("valid address rejected: %q", s)
		}
	}
	for _, s := range bad {
		if validEmail(s) {
			t.Errorf("invalid address accepted: %q", s)
		}
	}
}

func TestPasswordLengthIsCountedInRunes(t *testing.T) {
	// A ten-character Persian passphrase is thirty-odd bytes. Counting bytes
	// would let a shorter Persian password through than an English one.
	fa := "رمزعبورمن۱" // 10 runes
	if !validPassword(fa) {
		t.Errorf("10-rune Persian password rejected (len bytes = %d)", len(fa))
	}
	if validPassword("کوتاه") {
		t.Error("5-rune password accepted")
	}
	if validPassword(strings.Repeat("a", maxPasswordLen+1)) {
		t.Error("over-long password accepted")
	}
}

func TestIPPrefixTruncates(t *testing.T) {
	// Stored for abuse triage only, so it must not identify a person.
	for in, want := range map[string]string{
		"5.160.12.34": "5.160.12.0/24",
		"127.0.0.1":   "127.0.0.0/24",
		"not-an-ip":   "",
	} {
		if got := ipPrefix(in); got != want {
			t.Errorf("ipPrefix(%q) = %q, want %q", in, got, want)
		}
	}
	if got := ipPrefix("2a05:f480:1400:2b2a::1"); !strings.HasSuffix(got, "/48") {
		t.Errorf("ipv6 not truncated to /48: %q", got)
	}
}

func TestClientIPPrefersTheProxyHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.5:1234"
	r.Header.Set("X-Forwarded-For", "5.160.12.34, 10.0.0.1")
	if got := clientIP(r); got != "5.160.12.34" {
		t.Errorf("clientIP = %q, want the first forwarded address", got)
	}
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.RemoteAddr = "10.0.0.5:1234"
	if got := clientIP(r2); got != "10.0.0.5" {
		t.Errorf("clientIP = %q, want the socket address", got)
	}
}

func TestRateLimiterAllowsThenBlocks(t *testing.T) {
	l := newLimiter()
	for i := 0; i < 3; i++ {
		if !l.allow("k", 3) {
			t.Fatalf("request %d blocked inside the limit", i+1)
		}
	}
	if l.allow("k", 3) {
		t.Error("request past the limit was allowed")
	}
	// Limits are per key, so one abusive IP must not lock everyone out.
	if !l.allow("other", 3) {
		t.Error("a different key was blocked by another key's usage")
	}
}

func TestUnauthenticatedRequestsAreRejected(t *testing.T) {
	api := &API{limiter: newLimiter(), issuer: tokens.NewIssuer(strings.Repeat("k", 32), "khodrobin")}
	called := false
	handler := api.authed(func(http.ResponseWriter, *http.Request, *tokens.Claims) { called = true })

	for _, header := range []string{"", "Bearer garbage", "Bearer "} {
		rec := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		if header != "" {
			r.Header.Set("Authorization", header)
		}
		handler(rec, r)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("header %q gave status %d, want 401", header, rec.Code)
		}
	}
	if called {
		t.Error("the protected handler ran without a valid token")
	}
}

func TestAdminRouteHidesItselfFromNonAdmins(t *testing.T) {
	// 404 rather than 403: a 403 confirms the route exists and that this account
	// is merely not privileged enough.
	iss := tokens.NewIssuer(strings.Repeat("k", 32), "khodrobin")
	api := &API{limiter: newLimiter(), issuer: iss}
	raw, err := iss.Access(1, "a@b.co", true, false, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/auth/admin/summary", nil)
	r.Header.Set("Authorization", "Bearer "+raw)
	api.adminOnly(func(http.ResponseWriter, *http.Request, *tokens.Claims) {
		t.Error("admin handler ran for a non-admin")
	})(rec, r)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
