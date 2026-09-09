// Package handlers is the HTTP surface of the auth service.
//
// Two rules run through all of it.
//
// It never confirms whether an address has an account. Registration, login and
// password reset all answer the same way for a known and an unknown email,
// because a signup form that says "already registered" is an account
// enumeration oracle — and on a Persian car site the set of users is exactly
// the set of people who might be embarrassed by that.
//
// And it never blocks on mail. SMTP is slow and occasionally down; a
// registration that succeeds in the database must not fail because a message
// could not be sent.
package handlers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	appmail "github.com/sobhanaz/khodrobin/auth/internal/mail"
	"github.com/sobhanaz/khodrobin/auth/internal/store"
	"github.com/sobhanaz/khodrobin/auth/internal/tokens"
)

const (
	verifyTTL = 24 * time.Hour
	// Short on purpose: six digits against a rate-limited endpoint are safe for
	// a quarter of an hour, and a code that never expires is a code that gets
	// shoulder-surfed out of a phone's notification shade days later.
	codeTTL        = 15 * time.Minute
	resetTTL       = time.Hour
	lockThreshold  = 8
	lockDuration   = 15 * time.Minute
	minPasswordLen = 10
	maxPasswordLen = 200
	refreshCookie  = "kb_refresh"
	// A readable companion to the refresh token. Carries no authority; see
	// setRefreshCookie for why it exists.
	sessionHintCookie = "kb_session"
	// Admin lists are paged. Uncapped, one ?limit=1000000 turns a dashboard
	// into a full table scan streamed to a browser.
	defaultPageLimit = 50
	maxPageLimit     = 200
)

type API struct {
	store   *store.Store
	issuer  *tokens.Issuer
	mailer  *appmail.Mailer
	baseURL string
	log     *slog.Logger
	limiter *limiter
}

func New(st *store.Store, iss *tokens.Issuer, m *appmail.Mailer, baseURL string, log *slog.Logger) *API {
	return &API{store: st, issuer: iss, mailer: m,
		baseURL: strings.TrimRight(baseURL, "/"), log: log, limiter: newLimiter()}
}

func (a *API) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/register", a.rateLimited(5, a.register))
	mux.HandleFunc("POST /api/auth/login", a.rateLimited(10, a.login))
	mux.HandleFunc("POST /api/auth/logout", a.logout)
	mux.HandleFunc("POST /api/auth/refresh", a.rateLimited(60, a.refresh))
	mux.HandleFunc("GET /api/auth/verify", a.rateLimited(20, a.verify))
	// Typed by a human from a mail client, not clicked — the loose rate limit
	// that link verification uses is still the right one, because the code is
	// six digits, single-use, and fifteen minutes old.
	mux.HandleFunc("POST /api/auth/verify/code", a.rateLimited(10, a.verifyCode))
	mux.HandleFunc("POST /api/auth/resend", a.rateLimited(3, a.resend))
	mux.HandleFunc("POST /api/auth/forgot", a.rateLimited(3, a.forgot))
	mux.HandleFunc("POST /api/auth/reset", a.rateLimited(5, a.reset))
	mux.HandleFunc("GET /api/auth/me", a.authed(a.me))
	mux.HandleFunc("POST /api/auth/me/marketing", a.authed(a.setMarketing))
	mux.HandleFunc("GET /api/auth/searches", a.authed(a.listSearches))
	mux.HandleFunc("POST /api/auth/searches", a.authed(a.createSearch))
	mux.HandleFunc("DELETE /api/auth/searches/{id}", a.authed(a.deleteSearch))
	mux.HandleFunc("POST /api/auth/contact", a.rateLimited(3, a.contact))
	mux.HandleFunc("POST /api/auth/subscribe", a.rateLimited(5, a.subscribe))
	// Confirm and unsubscribe are clicked from a mail client, sometimes by its
	// link scanner before the human gets there, so the limit is the loose one
	// that verification already uses rather than the signup one.
	mux.HandleFunc("GET /api/auth/subscribe/confirm", a.rateLimited(20, a.confirmSubscribe))
	mux.HandleFunc("GET /api/auth/unsubscribe", a.rateLimited(20, a.unsubscribe))
	mux.HandleFunc("GET /api/auth/admin/summary", a.adminOnly(a.adminSummary))
	mux.HandleFunc("GET /api/auth/admin/users", a.adminOnly(a.adminUsers))
	mux.HandleFunc("GET /api/auth/admin/subscribers", a.adminOnly(a.adminSubscribers))
	mux.HandleFunc("GET /api/auth/admin/subscribers.csv", a.adminOnly(a.adminSubscribersCSV))
	// Registered twice on purpose. Compose reaches this service directly at
	// /healthz, while Caddy forwards the whole /api/auth/* path unchanged, so an
	// external monitor needs the prefixed form.
	mux.HandleFunc("GET /healthz", a.healthz)
	mux.HandleFunc("GET /readyz", a.readyz)
	mux.HandleFunc("GET /api/auth/healthz", a.healthz)
	mux.HandleFunc("GET /api/auth/readyz", a.readyz)
}

// --- plumbing --------------------------------------------------------------

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

// fail sends a Persian message the UI can show directly. The English detail
// stays in the logs: an error string is for the operator, not the visitor.
func fail(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func decode(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

// clientIP prefers the proxy header because Caddy always terminates in front.
func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if i := strings.IndexByte(fwd, ','); i > 0 {
			return strings.TrimSpace(fwd[:i])
		}
		return strings.TrimSpace(fwd)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ipPrefix truncates to a /24 so abuse can be correlated without storing the
// address of a person who filled in a contact form.
func ipPrefix(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}
	if v4 := parsed.To4(); v4 != nil {
		return net.IP{v4[0], v4[1], v4[2], 0}.String() + "/24"
	}
	return parsed.Mask(net.CIDRMask(48, 128)).String() + "/48"
}

// limiter is a fixed-window per-IP counter.
//
// In memory because these limits exist to blunt scripted abuse, not to be
// exact, and a restart clearing them is acceptable — unlike the failed-login
// lockout, which is in the database precisely because it must survive one.
type limiter struct {
	mu     sync.Mutex
	counts map[string]int
	window time.Time
}

func newLimiter() *limiter {
	return &limiter{counts: map[string]int{}, window: time.Now().Truncate(time.Minute)}
}

func (l *limiter) allow(key string, perMinute int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now().Truncate(time.Minute)
	if now.After(l.window) {
		l.counts = map[string]int{}
		l.window = now
	}
	l.counts[key]++
	return l.counts[key] <= perMinute
}

func (a *API) rateLimited(perMinute int, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path + "|" + clientIP(r)
		if !a.limiter.allow(key, perMinute) {
			w.Header().Set("Retry-After", "60")
			fail(w, http.StatusTooManyRequests, "تعداد درخواست‌ها زیاد بود. یک دقیقه صبر کن.")
			return
		}
		next(w, r)
	}
}

// page reads the two list parameters every admin table sends, with the cap.
func page(r *http.Request) (limit, offset int) {
	limit, offset = defaultPageLimit, 0
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 {
		limit = min(v, maxPageLimit)
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && v > 0 {
		offset = v
	}
	return limit, offset
}

type ctxKey struct{}

func (a *API) authed(next func(http.ResponseWriter, *http.Request, *tokens.Claims)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if raw == "" {
			fail(w, http.StatusUnauthorized, "برای این کار باید وارد شوی.")
			return
		}
		claims, err := a.issuer.Parse(raw)
		if err != nil {
			fail(w, http.StatusUnauthorized, "نشست معتبر نیست. دوباره وارد شو.")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, claims)), claims)
	}
}

func (a *API) adminOnly(next func(http.ResponseWriter, *http.Request, *tokens.Claims)) http.HandlerFunc {
	return a.authed(func(w http.ResponseWriter, r *http.Request, c *tokens.Claims) {
		if !c.IsAdmin {
			// 404, not 403: a 403 confirms the route exists and that this
			// account is merely not privileged enough.
			fail(w, http.StatusNotFound, "یافت نشد.")
			return
		}
		next(w, r, c)
	})
}

// --- registration and login ------------------------------------------------

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
	// Absent means no. Consent has to be an act, and a field that defaults to
	// true the moment a caller forgets it is not one.
	MarketingConsent bool `json:"marketing_consent,omitempty"`
}

func validEmail(s string) bool {
	addr, err := mail.ParseAddress(strings.TrimSpace(s))
	return err == nil && strings.Contains(addr.Address, ".")
}

// validPassword enforces length only.
//
// Length is what actually resists guessing; composition rules mostly push people
// toward "Password1!" and away from a long Persian phrase, which is stronger and
// easier to remember. Counted in runes so a Persian passphrase is not penalised
// for its multi-byte characters.
// passwordProblem names the rule a password fails, or returns "" if it passes.
//
// The three rules are length, one symbol, one capital. They were enforced in
// the browser and nowhere else: composables/usePasswordRules.ts checked all
// three and its comment claimed "the Go service enforces it independently",
// which was simply untrue. Anything posting straight at this endpoint got the
// length rule alone, so the policy was a suggestion to people using the form
// and absent for everyone else.
//
// Runes, not bytes, and the same definitions the browser uses: a symbol is
// anything that is not a letter, a digit or a space, so a Persian «؟» counts;
// a capital is unicode.IsUpper rather than A-Z, so the two sides agree instead
// of disagreeing at the moment of submit.
//
// It returns which rule failed. "Invalid password" makes someone guess, and
// guessing at a rule they cannot see is how people give up on a signup form.
func passwordProblem(s string) string {
	n := utf8.RuneCountInString(s)
	switch {
	case n < minPasswordLen:
		return fmt.Sprintf("رمز عبور باید دست‌کم %d نویسه باشد.", minPasswordLen)
	case n > maxPasswordLen:
		return fmt.Sprintf("رمز عبور نمی‌تواند بیشتر از %d نویسه باشد.", maxPasswordLen)
	}
	var symbol, upper bool
	for _, r := range s {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r):
			symbol = true
		}
	}
	if !symbol {
		return "رمز عبور باید دست‌کم یک نماد داشته باشد، مثل ! یا @ یا ؟."
	}
	if !upper {
		return "رمز عبور باید دست‌کم یک حرف بزرگ لاتین داشته باشد، مثل A."
	}
	return ""
}

// validPassword is the login-side check and stays length-only on purpose.
//
// Accounts created before the symbol and capital rules existed satisfy neither,
// the first admin account among them. Enforcing the new rules at login would
// lock out exactly the people who have been here longest, with no way back:
// fixing it requires logging in.
func validPassword(s string) bool {
	n := utf8.RuneCountInString(s)
	return n >= minPasswordLen && n <= maxPasswordLen
}

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if err := decode(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "درخواست نامعتبر است.")
		return
	}
	if !validEmail(in.Email) {
		fail(w, http.StatusBadRequest, "ایمیل معتبر وارد کن.")
		return
	}
	if problem := passwordProblem(in.Password); problem != "" {
		fail(w, http.StatusBadRequest, problem)
		return
	}

	hash, err := tokens.HashPassword(in.Password)
	if err != nil {
		a.log.Error("hash password", "err", err)
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد. دوباره تلاش کن.")
		return
	}

	var name *string
	if n := strings.TrimSpace(in.Name); n != "" {
		name = &n
	}

	user, err := a.store.CreateUser(r.Context(), in.Email, hash, name, in.MarketingConsent)
	switch {
	case errors.Is(err, store.ErrDuplicate):
		// Same reply as success. Telling an anonymous caller which addresses are
		// registered is an enumeration oracle, and the real owner still gets a
		// mail — theirs says someone tried to register their address.
		a.log.Info("register.duplicate", "email_domain", domainOf(in.Email))
		writeJSON(w, http.StatusAccepted, map[string]string{
			"status": "ok", "message": "اگر این ایمیل قبلاً ثبت نشده باشد، لینک تأیید برایت ارسال می‌شود."})
		return
	case err != nil:
		a.log.Error("create user", "err", err)
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد. دوباره تلاش کن.")
		return
	}

	// The opt-in is NOT acted on here. It is stored on the users row and
	// honoured at verification, because anyone can type anyone's address into a
	// registration form — which is the entire reason verification exists.
	a.sendVerification(r.Context(), user.ID, user.Email)
	writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "ok", "message": "اگر این ایمیل قبلاً ثبت نشده باشد، لینک تأیید برایت ارسال می‌شود."})
}

func domainOf(email string) string {
	if i := strings.LastIndex(email, "@"); i >= 0 {
		return email[i+1:]
	}
	return "?"
}

// sendVerification issues a link and a six-digit code, and mails both without
// blocking the reply. Two independent token rows: using one path does not
// invalidate the other, which is what makes «the link did nothing» recoverable
// by typing the code that arrived in the same message.
func (a *API) sendVerification(ctx context.Context, userID int64, email string) {
	secret, err := tokens.Secret()
	if err != nil {
		a.log.Error("verification secret", "err", err)
		return
	}
	if err := a.store.CreateToken(ctx, userID, "verify_email", tokens.Fingerprint(secret), verifyTTL); err != nil {
		a.log.Error("store verification token", "err", err)
		return
	}
	code, err := tokens.Code()
	if err != nil {
		a.log.Error("verification code", "err", err)
		return
	}
	if err := a.store.CreateToken(ctx, userID, "verify_email_code", tokens.Fingerprint(code), codeTTL); err != nil {
		a.log.Error("store verification code", "err", err)
		return
	}
	link := a.baseURL + "/verify?token=" + secret
	go func() {
		subject, body := appmail.Verify(link, code)
		if err := a.mailer.Send(email, subject, body); err != nil {
			// Not fatal: the account exists and the link can be resent.
			a.log.Warn("verification mail failed", "err", err)
		}
	}()
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if err := decode(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "درخواست نامعتبر است.")
		return
	}

	user, err := a.store.UserByEmail(r.Context(), in.Email)
	if err != nil {
		// Hash anyway so a missing account and a wrong password take the same
		// time. Skipping it turns response latency into an account oracle.
		_, _ = tokens.HashPassword(in.Password)
		fail(w, http.StatusUnauthorized, "ایمیل یا رمز عبور درست نیست.")
		return
	}
	if user.Locked(time.Now()) {
		fail(w, http.StatusTooManyRequests, "به دلیل تلاش‌های ناموفق، ورود موقتاً بسته است.")
		return
	}

	ok, err := tokens.VerifyPassword(in.Password, user.PasswordHash)
	if err != nil || !ok {
		_ = a.store.RecordLoginFailure(r.Context(), user.ID, lockThreshold, lockDuration)
		fail(w, http.StatusUnauthorized, "ایمیل یا رمز عبور درست نیست.")
		return
	}

	_ = a.store.RecordLoginSuccess(r.Context(), user.ID)
	a.issueSession(w, r, user)
}

// issueSession returns a short access token and sets the refresh cookie.
//
// The refresh token lives in an HttpOnly cookie so JavaScript cannot read it;
// the access token goes in the body because the front end must attach it to API
// calls. That split means an XSS bug can borrow a session for fifteen minutes
// rather than steal one for thirty days.
func (a *API) issueSession(w http.ResponseWriter, r *http.Request, user *store.User) {
	access, err := a.issuer.Access(user.ID, user.Email, user.Verified(), user.IsAdmin, time.Now())
	if err != nil {
		a.log.Error("issue access token", "err", err)
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد. دوباره تلاش کن.")
		return
	}
	refresh, err := tokens.Secret()
	if err != nil {
		a.log.Error("issue refresh token", "err", err)
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد. دوباره تلاش کن.")
		return
	}
	if err := a.store.CreateSession(r.Context(), user.ID, tokens.Fingerprint(refresh),
		r.UserAgent(), tokens.RefreshTTL); err != nil {
		a.log.Error("store session", "err", err)
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد. دوباره تلاش کن.")
		return
	}
	setRefreshCookie(w, refresh, tokens.RefreshTTL)
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": access,
		"expires_in":   int(tokens.AccessTTL.Seconds()),
		"user": map[string]any{
			"id": user.ID, "email": user.Email,
			"name": user.DisplayName, "verified": user.Verified(), "admin": user.IsAdmin,
		},
	})
}

// clearRefreshCookie drops both halves. Clearing only the token would leave the
// hint behind, and the front end would go on asking for a session that is gone.
func clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: refreshCookie, Path: "/api/auth", MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: sessionHintCookie, Path: "/", MaxAge: -1})
}

// setRefreshCookie writes the session, plus a readable flag saying one exists.
//
// The refresh token is HttpOnly on purpose, which means the browser cannot ask
// "am I signed in?" without a round trip. So the front end asked on every page
// load, and since search is anonymous by design, the overwhelming majority of
// those loads were anonymous: a wasted request on the critical path and a 401
// in the console of every visitor who never had an account.
//
// The flag carries no authority. It is readable, so anyone can forge it, and
// forging it buys nothing: all it permits is a refresh attempt that fails
// without the real token. It is a hint that saves a round trip, never a
// credential, and no handler reads it.
//
// Path is "/" because the hint is read on every page, while the token stays
// scoped to /api/auth, the only place it is needed.
func setRefreshCookie(w http.ResponseWriter, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionHintCookie,
		Value:    "1",
		Path:     "/",
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookie,
		Value:    value,
		Path:     "/api/auth",
		HttpOnly: true,
		Secure:   true,
		// Lax rather than Strict: the verification link arrives from an email
		// client, and Strict would drop the cookie on that navigation.
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	})
}

func (a *API) refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookie)
	if err != nil || cookie.Value == "" {
		fail(w, http.StatusUnauthorized, "نشستی برای تمدید وجود ندارد.")
		return
	}
	next, err := tokens.Secret()
	if err != nil {
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد.")
		return
	}
	userID, err := a.store.RotateSession(r.Context(), tokens.Fingerprint(cookie.Value),
		tokens.Fingerprint(next), r.UserAgent(), tokens.RefreshTTL)
	if err != nil {
		// RotateSession returns the user id alongside the error when it detects
		// reuse, having already revoked every session for that account.
		if userID != 0 {
			a.log.Warn("refresh token reuse; all sessions revoked", "user_id", userID)
		}
		clearRefreshCookie(w)
		fail(w, http.StatusUnauthorized, "نشست منقضی شده. دوباره وارد شو.")
		return
	}
	user, err := a.store.UserByID(r.Context(), userID)
	if err != nil {
		fail(w, http.StatusUnauthorized, "نشست معتبر نیست.")
		return
	}
	access, err := a.issuer.Access(user.ID, user.Email, user.Verified(), user.IsAdmin, time.Now())
	if err != nil {
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد.")
		return
	}
	setRefreshCookie(w, next, tokens.RefreshTTL)
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": access,
		"expires_in":   int(tokens.AccessTTL.Seconds()),
	})
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(refreshCookie); err == nil && cookie.Value != "" {
		_ = a.store.RevokeSession(r.Context(), tokens.Fingerprint(cookie.Value))
	}
	clearRefreshCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- verification and reset ------------------------------------------------

func (a *API) verify(w http.ResponseWriter, r *http.Request) {
	secret := r.URL.Query().Get("token")
	if secret == "" {
		fail(w, http.StatusBadRequest, "لینک نامعتبر است.")
		return
	}
	userID, err := a.store.ConsumeToken(r.Context(), "verify_email", tokens.Fingerprint(secret))
	if err != nil {
		fail(w, http.StatusBadRequest, "این لینک منقضی شده یا قبلاً استفاده شده است.")
		return
	}
	if err := a.verifyUser(r.Context(), userID); err != nil {
		a.log.Error("mark verified", "err", err)
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "ایمیلت تأیید شد."})
}

// verifyCode is the typing path of verification: the six-digit code from the
// same mail that carried the link.
func (a *API) verifyCode(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code string `json:"code"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "درخواست نامعتبر است.")
		return
	}
	code := strings.TrimSpace(in.Code)
	if !isSixDigits(code) {
		fail(w, http.StatusBadRequest, "کد ۶ رقمی را کامل وارد کن.")
		return
	}
	userID, err := a.store.ConsumeToken(r.Context(), "verify_email_code", tokens.Fingerprint(code))
	if err != nil {
		// Same shape of reply as the link path: nothing here tells a guesser
		// whether the code was real but used, or never existed.
		fail(w, http.StatusBadRequest, "این کد معتبر نیست یا منقضی شده است.")
		return
	}
	if err := a.verifyUser(r.Context(), userID); err != nil {
		a.log.Error("mark verified by code", "err", err)
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "ایمیلت تأیید شد."})
}

func isSixDigits(s string) bool {
	if len(s) != 6 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// verifyUser marks the account verified and sends the welcome mail only when
// this call is what made it verified.
//
// The guard matters: a second click on an old, still-valid link must not
// re-welcome an account that has been active for months.
// setMarketing is the authenticated way off the marketing list.
//
// The token route cannot serve everyone: rows backfilled for people who ticked
// the register box before tokens existed carry an unusable placeholder, so a
// secret will never match them. A signed-in request is proof of the same thing
// the secret proves, and it is the route the account page uses.
func (a *API) setMarketing(w http.ResponseWriter, r *http.Request, c *tokens.Claims) {
	var in struct {
		Consent bool `json:"consent"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "درخواست نامعتبر است.")
		return
	}
	if err := a.store.SetMarketingConsent(r.Context(), c.UserID, in.Consent); err != nil {
		a.log.Error("set marketing consent", "err", err)
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد.")
		return
	}
	msg := "دیگر ایمیل خبرنامه برایت نمی‌فرستیم."
	if in.Consent {
		msg = "از این به بعد خبرنامه برایت می‌فرستیم."
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "consent": in.Consent, "message": msg})
}

func (a *API) verifyUser(ctx context.Context, userID int64) error {
	user, err := a.store.UserByID(ctx, userID)
	if err != nil {
		return err
	}
	fresh := !user.Verified()
	if err := a.store.MarkVerified(ctx, userID); err != nil {
		return err
	}
	if fresh {
		a.sendWelcome(user.Email, a.recordMarketingOptIn(ctx, user))
	}
	return nil
}

// recordMarketingOptIn honours the register checkbox, and returns the secret
// that lets the person undo it.
//
// It ran in register() before this, straight after the account row was written
// and before anyone had clicked anything. Registering with a stranger's address
// and the box ticked therefore put that address into the mailable export marked
// confirmed, and worse, upgraded a pending double opt-in someone else had
// started. A tick on a form proves nothing about who owns the mailbox; the
// verification click is the only thing that does, so the consent is honoured
// exactly there.
//
// The secret is returned rather than discarded. The previous version generated
// one, hashed it into the row, and let it fall out of scope, so those rows were
// mailable with no token in existence that could unsubscribe them. It goes into
// the welcome mail, which makes «لغوش هم یک کلیک است، از پای هر ایمیل» true.
//
// A failure is logged, not returned: the account is verified either way, and
// losing the subscriber row costs less than failing a verification that worked.
func (a *API) recordMarketingOptIn(ctx context.Context, user *store.User) string {
	if !user.MarketingConsent {
		return ""
	}
	secret, err := tokens.Secret()
	if err != nil {
		a.log.Error("subscriber secret", "err", err)
		return ""
	}
	if err := a.store.AddConfirmedSubscriber(ctx, user.Email, tokens.Fingerprint(secret), "register"); err != nil {
		a.log.Error("register opt-in subscriber", "err", err)
		return ""
	}
	return secret
}

// sendWelcome mails the onboarding note without blocking the reply.
//
// optOut is empty for anyone who did not tick the marketing box, and the
// template omits the line entirely in that case rather than offering to
// unsubscribe someone from a list they are not on.
func (a *API) sendWelcome(email, optOut string) {
	link := a.baseURL + "/"
	unsub := ""
	if optOut != "" {
		unsub = a.baseURL + "/unsubscribe?token=" + url.QueryEscape(optOut)
	}
	go func() {
		subject, body := appmail.Welcome(link, unsub)
		if err := a.mailer.Send(email, subject, body); err != nil {
			a.log.Warn("welcome mail failed", "err", err)
		}
	}()
}

func (a *API) resend(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if err := decode(r, &in); err != nil || !validEmail(in.Email) {
		fail(w, http.StatusBadRequest, "ایمیل معتبر وارد کن.")
		return
	}
	if user, err := a.store.UserByEmail(r.Context(), in.Email); err == nil && !user.Verified() {
		a.sendVerification(r.Context(), user.ID, user.Email)
	}
	writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "ok", "message": "اگر حسابی با این ایمیل باشد، لینک تأیید ارسال شد."})
}

func (a *API) forgot(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if err := decode(r, &in); err != nil || !validEmail(in.Email) {
		fail(w, http.StatusBadRequest, "ایمیل معتبر وارد کن.")
		return
	}
	if user, err := a.store.UserByEmail(r.Context(), in.Email); err == nil {
		if secret, err := tokens.Secret(); err == nil {
			if err := a.store.CreateToken(r.Context(), user.ID, "reset_password",
				tokens.Fingerprint(secret), resetTTL); err == nil {
				link := a.baseURL + "/reset?token=" + secret
				email := user.Email
				go func() {
					subject, body := appmail.Reset(link)
					if err := a.mailer.Send(email, subject, body); err != nil {
						a.log.Warn("reset mail failed", "err", err)
					}
				}()
			}
		}
	}
	writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "ok", "message": "اگر حسابی با این ایمیل باشد، لینک بازیابی ارسال شد."})
}

func (a *API) reset(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := decode(r, &in); err != nil || in.Token == "" {
		fail(w, http.StatusBadRequest, "درخواست نامعتبر است.")
		return
	}
	if problem := passwordProblem(in.Password); problem != "" {
		fail(w, http.StatusBadRequest, problem)
		return
	}
	userID, err := a.store.ConsumeToken(r.Context(), "reset_password", tokens.Fingerprint(in.Token))
	if err != nil {
		fail(w, http.StatusBadRequest, "این لینک منقضی شده یا قبلاً استفاده شده است.")
		return
	}
	hash, err := tokens.HashPassword(in.Password)
	if err != nil {
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد.")
		return
	}
	if err := a.store.SetPassword(r.Context(), userID, hash); err != nil {
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد.")
		return
	}
	// Changing a password ends every existing session. If the reset happened
	// because someone else had access, leaving their session alive defeats it.
	_ = a.store.RevokeAllSessions(r.Context(), userID)
	// The security notice is worth sending even when the owner did the reset —
	// the one time it matters is the time the reset was not theirs, and a mail
	// that only fires when suspicious is a mail an attacker can suppress.
	if user, err := a.store.UserByID(r.Context(), userID); err == nil {
		login, forgot := a.baseURL+"/login", a.baseURL+"/forgot"
		email := user.Email
		go func() {
			subject, body := appmail.PasswordChanged(login, forgot)
			if err := a.mailer.Send(email, subject, body); err != nil {
				a.log.Warn("password-changed mail failed", "err", err)
			}
		}()
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "رمز عبور عوض شد. دوباره وارد شو."})
}

// --- account ---------------------------------------------------------------

func (a *API) me(w http.ResponseWriter, r *http.Request, c *tokens.Claims) {
	user, err := a.store.UserByID(r.Context(), c.UserID)
	if err != nil {
		fail(w, http.StatusUnauthorized, "حساب پیدا نشد.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": user.ID, "email": user.Email, "name": user.DisplayName,
		"verified": user.Verified(), "admin": user.IsAdmin, "created_at": user.CreatedAt,
	})
}

func (a *API) listSearches(w http.ResponseWriter, r *http.Request, c *tokens.Claims) {
	list, err := a.store.ListSavedSearches(r.Context(), c.UserID)
	if err != nil {
		a.log.Error("list saved searches", "err", err)
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"searches": list})
}

func (a *API) createSearch(w http.ResponseWriter, r *http.Request, c *tokens.Claims) {
	var in struct {
		Label    string   `json:"label,omitempty"`
		Query    string   `json:"query"`
		Mode     string   `json:"mode,omitempty"`
		AlertPct *float64 `json:"alert_pct,omitempty"`
	}
	if err := decode(r, &in); err != nil || strings.TrimSpace(in.Query) == "" {
		fail(w, http.StatusBadRequest, "متن جست‌وجو را وارد کن.")
		return
	}
	// Alerts go by email, so an unverified address must not be able to arm one.
	// Otherwise the product becomes a way to mail arbitrary strangers.
	if in.AlertPct != nil && !c.Verified {
		fail(w, http.StatusForbidden, "برای هشدار قیمت، اول ایمیلت را تأیید کن.")
		return
	}
	if in.AlertPct != nil && (*in.AlertPct < 1 || *in.AlertPct > 90) {
		fail(w, http.StatusBadRequest, "درصد هشدار باید بین ۱ تا ۹۰ باشد.")
		return
	}
	mode := in.Mode
	if mode == "" {
		mode = "relevant"
	}
	var label *string
	if l := strings.TrimSpace(in.Label); l != "" {
		label = &l
	}
	saved, err := a.store.CreateSavedSearch(r.Context(), c.UserID, label,
		strings.TrimSpace(in.Query), mode, in.AlertPct)
	if err != nil {
		a.log.Error("create saved search", "err", err)
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد.")
		return
	}
	// Arming an alert gets a confirmation. An alert people do not know is on
	// gets deleted as spam the first time it fires — the mail states the same
	// search and threshold the alert will later fire with.
	if in.AlertPct != nil {
		pct, email := *in.AlertPct, c.Email
		resultsLink := a.baseURL + "/?q=" + url.QueryEscape(saved.Query)
		go func() {
			subject, body := appmail.SavedSearchCreated(saved.Query, pct, resultsLink)
			if err := a.mailer.Send(email, subject, body); err != nil {
				a.log.Warn("saved-search mail failed", "err", err)
			}
		}()
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (a *API) deleteSearch(w http.ResponseWriter, r *http.Request, c *tokens.Claims) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		fail(w, http.StatusBadRequest, "شناسه نامعتبر است.")
		return
	}
	if err := a.store.DeleteSavedSearch(r.Context(), c.UserID, id); err != nil {
		// ErrNotFound covers both "no such row" and "belongs to someone else",
		// which is the answer we want to give in either case.
		fail(w, http.StatusNotFound, "پیدا نشد.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- contact and admin -----------------------------------------------------

func (a *API) contact(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Subject string `json:"subject,omitempty"`
		Body    string `json:"body"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "درخواست نامعتبر است.")
		return
	}
	name, body := strings.TrimSpace(in.Name), strings.TrimSpace(in.Body)
	if name == "" || !validEmail(in.Email) || utf8.RuneCountInString(body) < 10 {
		fail(w, http.StatusBadRequest, "نام، ایمیل معتبر و پیام حداقل ۱۰ نویسه لازم است.")
		return
	}
	if utf8.RuneCountInString(body) > 5000 {
		fail(w, http.StatusBadRequest, "پیام خیلی طولانی است.")
		return
	}
	// Stored first, mailed second. A message must not be lost because SMTP is
	// down — which is exactly when someone is most likely writing in.
	if err := a.store.CreateContactMessage(r.Context(), name, strings.TrimSpace(in.Email),
		strings.TrimSpace(in.Subject), body, ipPrefix(clientIP(r))); err != nil {
		a.log.Error("store contact message", "err", err)
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد.")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "ok", "message": "پیامت رسید. ممنون."})
}

// --- newsletter ------------------------------------------------------------

// signupSource labels where a signup came from. It arrives from the page, so it
// is clamped rather than trusted — and an unusable label falls back to the
// default instead of failing the request, because a marketing tag is never
// worth losing a subscriber over.
//
// Trimming the ends is not enough. This string is written into the CSV export,
// and a newline in the middle of it survives: csv.Writer quotes the cell
// correctly across two physical lines, and the admin page counts exported
// addresses by splitting on newline — so one crafted label reports more people
// on the list than are on it. Every non-printable rune goes for the same
// reason, bidi overrides included, since the export is opened in a spreadsheet
// by a Persian reader and a character that reverses the run around it is a
// display trick, not a label.
func signupSource(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || utf8.RuneCountInString(s) > 40 || strings.ContainsFunc(s, unlabellable) {
		return "landing"
	}
	return s
}

// zwnj is the half-space Persian words are spelled with.
const zwnj = '\u200c'

// unlabellable rejects what cannot appear in a label. The half-space is the one
// invisible character that stays: «صفحه‌ی اصلی» is two words, and a rule that
// drops it on the floor is a Latin rule wearing a safety badge.
func unlabellable(r rune) bool { return !unicode.IsPrint(r) && r != zwnj }

// subscribe starts double opt-in, and answers the same way every time.
//
// 202 for a fresh address, for one already confirmed, and for one that left
// last month. Registration is enumeration-safe for the reason given at the top
// of this file, and a newsletter box that answered differently would hand back
// the same oracle through a form that needs no password at all.
//
// The list itself stays empty until the link in that one message is clicked, so
// typing a stranger's address in here costs them one mail and enrols nobody.
func (a *API) subscribe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email  string `json:"email"`
		Source string `json:"source,omitempty"`
	}
	if err := decode(r, &in); err != nil || !validEmail(in.Email) {
		fail(w, http.StatusBadRequest, "ایمیل معتبر وارد کن.")
		return
	}
	secret, err := tokens.Secret()
	if err != nil {
		a.log.Error("subscribe secret", "err", err)
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد. دوباره تلاش کن.")
		return
	}
	email := store.NormalizeEmail(in.Email)
	owed, err := a.store.SubscribePending(r.Context(), email, tokens.Fingerprint(secret),
		signupSource(in.Source))
	if err != nil {
		a.log.Error("subscribe", "err", err)
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد. دوباره تلاش کن.")
		return
	}
	if owed {
		// Only when the row actually needs a confirmation. Mailing an address
		// that is already confirmed every time somebody types it into the box
		// is how a sender earns a complaint rate.
		a.sendSubscribeConfirmation(email, secret)
	}
	writeJSON(w, http.StatusAccepted, map[string]string{
		"status":  "ok",
		"message": "اگر این نشانی تازه باشد، لینک تأیید برایت فرستاده می‌شود. تا تأییدش نکنی چیزی نمی‌فرستیم."})
}

// sendSubscribeConfirmation mails the double opt-in link without blocking.
//
// Both links carry the same secret because one row has one token: the link that
// confirms the address is the link that removes it, which is why the mail can
// offer a way out to someone who never asked to be in.
func (a *API) sendSubscribeConfirmation(email, secret string) {
	confirm := a.baseURL + "/subscribed?token=" + secret
	optOut := a.baseURL + "/unsubscribe?token=" + secret
	go func() {
		subject, body := appmail.ConfirmSubscription(confirm, optOut)
		if err := a.mailer.Send(email, subject, body); err != nil {
			a.log.Warn("subscribe confirmation mail failed", "err", err)
		}
	}()
}

// confirmSubscribe is the half of double opt-in that creates the consent.
//
// Until this runs the row is an address somebody typed into a form, possibly
// not their own. After it there is a timestamp to point at, which is the only
// useful answer to "why are you mailing me".
func (a *API) confirmSubscribe(w http.ResponseWriter, r *http.Request) {
	secret := r.URL.Query().Get("token")
	if secret == "" {
		fail(w, http.StatusBadRequest, "لینک نامعتبر است.")
		return
	}
	if err := a.store.ConfirmSubscriber(r.Context(), tokens.Fingerprint(secret)); err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			a.log.Error("confirm subscriber", "err", err)
		}
		fail(w, http.StatusBadRequest, "این لینک معتبر نیست. دوباره عضو شو.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "عضویتت تأیید شد."})
}

// unsubscribe is a GET and needs no session, because it is clicked from a mail
// client that carries neither our cookie nor a bearer token.
//
// Idempotent for the same reason it is a GET: mail clients prefetch links and
// people click twice. The second click has to look exactly like the first —
// «خطا» on an unsubscribe page is answered with the spam button, and that costs
// the sending domain far more than one address.
func (a *API) unsubscribe(w http.ResponseWriter, r *http.Request) {
	secret := r.URL.Query().Get("token")
	if secret == "" {
		fail(w, http.StatusBadRequest, "لینک نامعتبر است.")
		return
	}
	if err := a.store.Unsubscribe(r.Context(), tokens.Fingerprint(secret)); err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			a.log.Error("unsubscribe", "err", err)
		}
		fail(w, http.StatusBadRequest, "لینک نامعتبر است.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok", "message": "از فهرست خبرنامه حذف شدی. دیگر ایمیلی نمی‌فرستیم."})
}

func (a *API) adminSummary(w http.ResponseWriter, r *http.Request, _ *tokens.Claims) {
	counts, err := a.store.Counts(r.Context())
	if err != nil {
		a.log.Error("admin counts", "err", err)
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"counts": counts,
		"mail":   map[string]bool{"configured": a.mailer.Configured()},
	})
}

func (a *API) adminUsers(w http.ResponseWriter, r *http.Request, _ *tokens.Claims) {
	limit, offset := page(r)
	list, total, err := a.store.ListUsers(r.Context(), limit, offset)
	if err != nil {
		a.log.Error("admin users", "err", err)
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": list, "total": total})
}

func (a *API) adminSubscribers(w http.ResponseWriter, r *http.Request, _ *tokens.Claims) {
	limit, offset := page(r)
	list, total, err := a.store.ListSubscribers(r.Context(), limit, offset)
	if err != nil {
		a.log.Error("admin subscribers", "err", err)
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد.")
		return
	}
	// Everyone, including the people who left. The operator needs to see the
	// funnel; the export below is what may actually be mailed.
	writeJSON(w, http.StatusOK, map[string]any{"subscribers": list, "total": total})
}

// adminSubscribersCSV exports only the addresses that may legally be mailed.
//
// The filter is the entire point of the file, and it lives in the query rather
// than here: confirmed, and not unsubscribed. Anything wider is a spam run
// wearing a .csv extension, and the sending domain pays for it for months after
// the campaign is forgotten.
func (a *API) adminSubscribersCSV(w http.ResponseWriter, r *http.Request, _ *tokens.Claims) {
	list, err := a.store.MailableSubscribers(r.Context())
	if err != nil {
		a.log.Error("mailable subscribers", "err", err)
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد.")
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="khodrobin-subscribers.csv"`)
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"email", "source", "confirmed_at"})
	for _, sub := range list {
		confirmed := ""
		if sub.ConfirmedAt != nil {
			confirmed = sub.ConfirmedAt.UTC().Format(time.RFC3339)
		}
		_ = cw.Write([]string{csvCell(sub.Email), csvCell(sub.Source), confirmed})
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		a.log.Error("write subscribers csv", "err", err)
	}
}

// csvCell defuses a cell that a spreadsheet would treat as a formula. «=» is a
// legal first character of an email address, and Excel runs what follows it on
// the machine of whoever opens the export — which here is always the owner.
func csvCell(s string) string {
	if s != "" && strings.IndexByte("=+-@\t\r", s[0]) >= 0 {
		return "'" + s
	}
	return s
}

func (a *API) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// readyz asserts the database is reachable, because without it this service can
// do nothing at all — unlike mail, which it can run without.
func (a *API) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.store.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "not ready", "checks": map[string]string{"database": "unreachable"}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"checks": map[string]any{"database": "ok", "mail_configured": a.mailer.Configured()},
	})
}
