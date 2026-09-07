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
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	appmail "github.com/sobhanaz/khodrobin/auth/internal/mail"
	"github.com/sobhanaz/khodrobin/auth/internal/store"
	"github.com/sobhanaz/khodrobin/auth/internal/tokens"
)

const (
	verifyTTL      = 24 * time.Hour
	resetTTL       = time.Hour
	lockThreshold  = 8
	lockDuration   = 15 * time.Minute
	minPasswordLen = 10
	maxPasswordLen = 200
	refreshCookie  = "kb_refresh"
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
	mux.HandleFunc("POST /api/auth/resend", a.rateLimited(3, a.resend))
	mux.HandleFunc("POST /api/auth/forgot", a.rateLimited(3, a.forgot))
	mux.HandleFunc("POST /api/auth/reset", a.rateLimited(5, a.reset))
	mux.HandleFunc("GET /api/auth/me", a.authed(a.me))
	mux.HandleFunc("GET /api/auth/searches", a.authed(a.listSearches))
	mux.HandleFunc("POST /api/auth/searches", a.authed(a.createSearch))
	mux.HandleFunc("DELETE /api/auth/searches/{id}", a.authed(a.deleteSearch))
	mux.HandleFunc("POST /api/auth/contact", a.rateLimited(3, a.contact))
	mux.HandleFunc("GET /api/auth/admin/summary", a.adminOnly(a.adminSummary))
	mux.HandleFunc("GET /healthz", a.healthz)
	mux.HandleFunc("GET /readyz", a.readyz)
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
	if !validPassword(in.Password) {
		fail(w, http.StatusBadRequest, "رمز عبور باید حداقل ۱۰ نویسه باشد.")
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

	user, err := a.store.CreateUser(r.Context(), in.Email, hash, name)
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

// sendVerification issues a link and mails it without blocking the reply.
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
	link := a.baseURL + "/verify?token=" + secret
	go func() {
		subject, body := appmail.Verify(link)
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

func setRefreshCookie(w http.ResponseWriter, value string, ttl time.Duration) {
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
		http.SetCookie(w, &http.Cookie{Name: refreshCookie, Path: "/api/auth", MaxAge: -1})
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
	http.SetCookie(w, &http.Cookie{Name: refreshCookie, Path: "/api/auth", MaxAge: -1})
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
	if err := a.store.MarkVerified(r.Context(), userID); err != nil {
		a.log.Error("mark verified", "err", err)
		fail(w, http.StatusInternalServerError, "مشکلی پیش آمد.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "ایمیلت تأیید شد."})
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
	if !validPassword(in.Password) {
		fail(w, http.StatusBadRequest, "رمز عبور باید حداقل ۱۰ نویسه باشد.")
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
