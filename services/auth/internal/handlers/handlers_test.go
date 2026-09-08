package handlers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	appmail "github.com/sobhanaz/khodrobin/auth/internal/mail"
	"github.com/sobhanaz/khodrobin/auth/internal/store"
	"github.com/sobhanaz/khodrobin/auth/internal/store/storetest"
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

// --- newsletter, consent and the admin list --------------------------------
//
// These go through the real store against a throwaway Postgres. The rules worth
// testing here — who ends up in the export, what a returning subscriber resets —
// live in SQL, and a fake store would only prove that the fake agrees with the
// test that wrote it.

func TestMain(m *testing.M) { os.Exit(storetest.Run(m)) }

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func testIssuer() *tokens.Issuer {
	return tokens.NewIssuer(strings.Repeat("k", 32), "khodrobin")
}

func testAPI(t *testing.T) (*http.ServeMux, *store.Store) {
	t.Helper()
	st := storetest.New(t)
	// A zero Mailer reports itself unconfigured and fails fast, so a test run
	// cannot pick up SMTP credentials from the developer's environment and mail
	// a real address at khodrobin.test.
	api := New(st, testIssuer(), &appmail.Mailer{}, "https://khodrobin.test", quiet())
	mux := http.NewServeMux()
	api.Routes(mux)
	return mux, st
}

func bearer(t *testing.T, admin bool) string {
	t.Helper()
	raw, err := testIssuer().Access(1, "admin@khodrobin.test", true, admin, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + raw
}

func call(t *testing.T, mux *http.ServeMux, method, target, body, auth string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	if auth != "" {
		r.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r)
	return rec
}

// seed puts a subscriber in a known state with a secret the test holds, the way
// the mail the handler sends would.
func seed(t *testing.T, st *store.Store, email, secret string, confirm, unsub bool) {
	t.Helper()
	ctx := context.Background()
	if _, err := st.SubscribePending(ctx, email, tokens.Fingerprint(secret), "landing"); err != nil {
		t.Fatal(err)
	}
	if confirm {
		if err := st.ConfirmSubscriber(ctx, tokens.Fingerprint(secret)); err != nil {
			t.Fatal(err)
		}
	}
	if unsub {
		if err := st.Unsubscribe(ctx, tokens.Fingerprint(secret)); err != nil {
			t.Fatal(err)
		}
	}
}

func exportedEmails(t *testing.T, mux *http.ServeMux) []string {
	t.Helper()
	rec := call(t, mux, http.MethodGet, "/api/auth/admin/subscribers.csv", "", bearer(t, true))
	if rec.Code != http.StatusOK {
		t.Fatalf("csv export status = %d", rec.Code)
	}
	rows, err := csv.NewReader(rec.Body).ReadAll()
	if err != nil {
		t.Fatalf("export is not valid csv: %v", err)
	}
	if len(rows) == 0 || rows[0][0] != "email" {
		t.Fatalf("export is missing its header row: %v", rows)
	}
	out := []string{}
	for _, row := range rows[1:] {
		out = append(out, row[0])
	}
	return out
}

func TestSubscribeAnswersTheSameWayForEveryAddress(t *testing.T) {
	// A signup box that answered differently for a known address would be the
	// account-enumeration oracle the 202 on register exists to remove, on a form
	// that needs no password at all.
	mux, st := testAPI(t)
	seed(t, st, "member@khodrobin.test", "already-in", true, false)
	seed(t, st, "left@khodrobin.test", "gone", true, true)

	var bodies []string
	for _, email := range []string{"new@khodrobin.test", "new@khodrobin.test",
		"member@khodrobin.test", "left@khodrobin.test"} {
		rec := call(t, mux, http.MethodPost, "/api/auth/subscribe",
			`{"email":"`+email+`","source":"landing"}`, "")
		if rec.Code != http.StatusAccepted {
			t.Fatalf("%s gave status %d, want 202", email, rec.Code)
		}
		bodies = append(bodies, rec.Body.String())
	}
	for i, b := range bodies {
		if b != bodies[0] {
			t.Errorf("reply %d differs from the first:\n%s\n%s", i, b, bodies[0])
		}
	}

	if rec := call(t, mux, http.MethodPost, "/api/auth/subscribe", `{"email":"nope"}`, ""); rec.Code != http.StatusBadRequest {
		t.Errorf("a malformed address gave status %d, want 400", rec.Code)
	}
}

func TestSubscribingIsNotConsentingUntilTheLinkIsClicked(t *testing.T) {
	mux, st := testAPI(t)

	// Straight through the form: the token only exists in the mail, so this
	// address can never reach the export by itself.
	if rec := call(t, mux, http.MethodPost, "/api/auth/subscribe",
		`{"email":"typed@khodrobin.test"}`, ""); rec.Code != http.StatusAccepted {
		t.Fatalf("subscribe status = %d", rec.Code)
	}
	if got := exportedEmails(t, mux); len(got) != 0 {
		t.Errorf("export = %v, want empty before anyone clicked", got)
	}

	for _, target := range []string{
		"/api/auth/subscribe/confirm",
		"/api/auth/subscribe/confirm?token=",
		"/api/auth/subscribe/confirm?token=not-the-one",
	} {
		if rec := call(t, mux, http.MethodGet, target, "", ""); rec.Code != http.StatusBadRequest {
			t.Errorf("%s gave status %d, want 400", target, rec.Code)
		}
	}

	seed(t, st, "clicked@khodrobin.test", "the-secret", false, false)
	if rec := call(t, mux, http.MethodGet,
		"/api/auth/subscribe/confirm?token=the-secret", "", ""); rec.Code != http.StatusOK {
		t.Fatalf("confirm status = %d, want 200", rec.Code)
	}
	if got := exportedEmails(t, mux); len(got) != 1 || got[0] != "clicked@khodrobin.test" {
		t.Errorf("export = %v, want only the address that clicked", got)
	}
}

func TestUnsubscribeNeedsNoLoginAndSurvivesADoubleClick(t *testing.T) {
	// Clicked from a mail client: no cookie, no bearer token, and often clicked
	// twice — once by the client's link scanner, once by the person.
	mux, st := testAPI(t)
	seed(t, st, "bye@khodrobin.test", "opt-out", true, false)

	for i := 1; i <= 3; i++ {
		rec := call(t, mux, http.MethodGet, "/api/auth/unsubscribe?token=opt-out", "", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("click %d gave status %d, want 200 every time", i, rec.Code)
		}
	}
	if got := exportedEmails(t, mux); len(got) != 0 {
		t.Errorf("export = %v, want empty after unsubscribing", got)
	}
	for _, target := range []string{"/api/auth/unsubscribe", "/api/auth/unsubscribe?token=made-up"} {
		if rec := call(t, mux, http.MethodGet, target, "", ""); rec.Code != http.StatusBadRequest {
			t.Errorf("%s gave status %d, want 400", target, rec.Code)
		}
	}
}

func TestComingBackMeansConfirmingAgain(t *testing.T) {
	mux, st := testAPI(t)
	const email = "returning@khodrobin.test"
	seed(t, st, email, "first-time", true, true)

	if rec := call(t, mux, http.MethodPost, "/api/auth/subscribe",
		`{"email":"`+email+`","source":"footer"}`, ""); rec.Code != http.StatusAccepted {
		t.Fatalf("re-subscribe status = %d", rec.Code)
	}
	if got := exportedEmails(t, mux); len(got) != 0 {
		t.Errorf("export = %v, want empty until the new link is clicked", got)
	}

	rec := call(t, mux, http.MethodGet, "/api/auth/admin/subscribers", "", bearer(t, true))
	var page struct {
		Subscribers []store.Subscriber `json:"subscribers"`
		Total       int64              `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Subscribers) != 1 {
		t.Fatalf("admin list = %d rows (total %d), want the one row reused", len(page.Subscribers), page.Total)
	}
	row := page.Subscribers[0]
	if row.UnsubscribedAt != nil {
		t.Error("unsubscribed_at was not cleared by the new signup")
	}
	if row.ConfirmedAt != nil {
		t.Error("the old consent was carried over instead of being asked for again")
	}
	// The old link is dead, which is what stops a stale mail from re-consenting
	// on the subscriber's behalf.
	if rec := call(t, mux, http.MethodGet,
		"/api/auth/subscribe/confirm?token=first-time", "", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("the retired link gave status %d, want 400", rec.Code)
	}
}

func TestAdminRoutesAreInvisibleWithoutAnAdminToken(t *testing.T) {
	// 404 rather than 403: a 403 confirms the route is there and that this
	// account is merely not privileged enough.
	mux, _ := testAPI(t)
	for _, target := range []string{
		"/api/auth/admin/summary",
		"/api/auth/admin/users",
		"/api/auth/admin/subscribers",
		"/api/auth/admin/subscribers.csv",
	} {
		for _, tc := range []struct {
			name, auth string
			want       int
		}{
			{"anonymous", "", http.StatusUnauthorized},
			{"signed in, not admin", bearer(t, false), http.StatusNotFound},
			{"admin", bearer(t, true), http.StatusOK},
		} {
			if rec := call(t, mux, http.MethodGet, target, "", tc.auth); rec.Code != tc.want {
				t.Errorf("%s as %s gave status %d, want %d", target, tc.name, rec.Code, tc.want)
			}
		}
	}
}

func TestTheExportCarriesOnlyMailableRows(t *testing.T) {
	// The point of the file: it is the list that is legal and deliverable to
	// mail. Anything wider is a spam run wearing a .csv extension.
	mux, st := testAPI(t)
	seed(t, st, "yes@khodrobin.test", "a", true, false)
	seed(t, st, "unconfirmed@khodrobin.test", "b", false, false)
	seed(t, st, "unsubscribed@khodrobin.test", "c", true, true)

	rec := call(t, mux, http.MethodGet, "/api/auth/admin/subscribers.csv", "", bearer(t, true))
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("content-type = %q, want text/csv", ct)
	}
	body := rec.Body.String()
	for _, unwanted := range []string{"unconfirmed@khodrobin.test", "unsubscribed@khodrobin.test"} {
		if strings.Contains(body, unwanted) {
			t.Errorf("%s reached the export", unwanted)
		}
	}
	if got := exportedEmails(t, mux); len(got) != 1 || got[0] != "yes@khodrobin.test" {
		t.Errorf("export = %v, want only the confirmed, still-subscribed address", got)
	}

	// The admin table is the wider view on purpose — the operator needs to see
	// the funnel the export leaves out.
	list := call(t, mux, http.MethodGet, "/api/auth/admin/subscribers", "", bearer(t, true))
	if !strings.Contains(list.Body.String(), "unsubscribed@khodrobin.test") {
		t.Error("the admin list hid a subscriber that the export is right to skip")
	}
}

func TestRegisterRecordsTheOptInEitherWay(t *testing.T) {
	// The register page promises no promotional mail. The checkbox is the only
	// thing that changes that, so a missing field has to mean no.
	mux, _ := testAPI(t)
	for _, body := range []string{
		`{"email":"in@khodrobin.test","password":"a-long-enough-one","marketing_consent":true}`,
		`{"email":"out@khodrobin.test","password":"a-long-enough-one","marketing_consent":false}`,
		`{"email":"silent@khodrobin.test","password":"a-long-enough-one"}`,
	} {
		if rec := call(t, mux, http.MethodPost, "/api/auth/register", body, ""); rec.Code != http.StatusAccepted {
			t.Fatalf("register gave status %d for %s", rec.Code, body)
		}
	}
	rec := call(t, mux, http.MethodGet, "/api/auth/admin/summary", "", bearer(t, true))
	var summary struct {
		Counts store.Counts `json:"counts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Counts.Users != 3 {
		t.Fatalf("users = %d, want 3", summary.Counts.Users)
	}
	if summary.Counts.MarketingOptin != 1 {
		t.Errorf("marketing_optin = %d, want only the account that ticked the box",
			summary.Counts.MarketingOptin)
	}
}
