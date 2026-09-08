// Package store is the only place that talks to Postgres.
//
// Every query is parameterised — pgx does not interpolate, so string building
// never enters the picture. The methods are deliberately narrow: handlers get
// verbs like "CreateUser" rather than a connection to write SQL against, so
// there is one place to audit when something about the data model changes.
package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrDuplicate = errors.New("already exists")
)

type Store struct{ pool *pgxpool.Pool }

func New(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// Migrate applies the schema. Idempotent by construction — every statement is
// IF NOT EXISTS — so it can run on every boot without a migration table.
func (s *Store) Migrate(ctx context.Context, sqlText string) error {
	_, err := s.pool.Exec(ctx, sqlText)
	return err
}

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	DisplayName  *string
	VerifiedAt   *time.Time
	IsAdmin      bool
	CreatedAt    time.Time
	LastLoginAt  *time.Time
	FailedLogins int
	LockedUntil  *time.Time
	// Opt-in for promotional mail, which is a different question from having an
	// account: the register page promises we send none without it.
	MarketingConsent bool
}

func (u *User) Verified() bool { return u.VerifiedAt != nil }

// Locked reports whether login is currently refused for this account.
func (u *User) Locked(now time.Time) bool {
	return u.LockedUntil != nil && now.Before(*u.LockedUntil)
}

// NormalizeEmail is the single definition of account identity.
//
// Lowercased and trimmed so «Sobhan@Gmail.com » and «sobhan@gmail.com» cannot
// become two accounts — and so a password reset always finds the row a user
// expects.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// CreateUser takes the marketing opt-in alongside the credentials because the
// timestamp has to be stamped in the same statement that records the choice.
// Written afterwards it would be a second thing to remember, and the one time
// it is forgotten the row says "consented" with no date to defend it.
func (s *Store) CreateUser(ctx context.Context, email, hash string, name *string, marketing bool) (*User, error) {
	u := &User{Email: NormalizeEmail(email), PasswordHash: hash, DisplayName: name,
		MarketingConsent: marketing}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, display_name, marketing_consent, marketing_consent_at)
		 VALUES ($1, $2, $3, $4, CASE WHEN $4 THEN now() END)
		 RETURNING id, created_at, is_admin`,
		u.Email, hash, name, marketing,
	).Scan(&u.ID, &u.CreatedAt, &u.IsAdmin)
	if isUnique(err) {
		return nil, ErrDuplicate
	}
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

const userCols = `id, email, password_hash, display_name, verified_at, is_admin,
                  created_at, last_login_at, failed_logins, locked_until, marketing_consent`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.VerifiedAt,
		&u.IsAdmin, &u.CreatedAt, &u.LastLoginAt, &u.FailedLogins, &u.LockedUntil,
		&u.MarketingConsent)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return &u, nil
}

func (s *Store) UserByEmail(ctx context.Context, email string) (*User, error) {
	return scanUser(s.pool.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE email = $1`, NormalizeEmail(email)))
}

func (s *Store) UserByID(ctx context.Context, id int64) (*User, error) {
	return scanUser(s.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id = $1`, id))
}

func (s *Store) MarkVerified(ctx context.Context, userID int64) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET verified_at = now() WHERE id = $1 AND verified_at IS NULL`, userID)
	return err
}

func (s *Store) SetPassword(ctx context.Context, userID int64, hash string) error {
	// Clearing the lockout is deliberate: someone who proved control of the
	// mailbox has demonstrated more than a password attempt ever does.
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET password_hash = $2, failed_logins = 0, locked_until = NULL
		 WHERE id = $1`, userID, hash)
	return err
}

// RecordLoginSuccess resets the brute-force counter.
func (s *Store) RecordLoginSuccess(ctx context.Context, userID int64) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET last_login_at = now(), failed_logins = 0, locked_until = NULL
		 WHERE id = $1`, userID)
	return err
}

// RecordLoginFailure counts a bad attempt and locks the account once they pile
// up. Persisted rather than in memory so a restart is not a free reset.
func (s *Store) RecordLoginFailure(ctx context.Context, userID int64, threshold int, lockFor time.Duration) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users
		    SET failed_logins = failed_logins + 1,
		        locked_until = CASE WHEN failed_logins + 1 >= $2
		                            THEN now() + $3::interval ELSE locked_until END
		  WHERE id = $1`,
		userID, threshold, fmt.Sprintf("%d seconds", int(lockFor.Seconds())))
	return err
}

// --- one-time tokens -------------------------------------------------------

func (s *Store) CreateToken(ctx context.Context, userID int64, purpose, hash string, ttl time.Duration) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO tokens (user_id, purpose, token_hash, expires_at)
		 VALUES ($1, $2, $3, now() + $4::interval)`,
		userID, purpose, hash, fmt.Sprintf("%d seconds", int(ttl.Seconds())))
	return err
}

// ConsumeToken atomically claims a token and returns its owner.
//
// The UPDATE ... WHERE used_at IS NULL RETURNING is the whole point: two
// simultaneous clicks on the same verification link cannot both succeed,
// without a transaction or a lock.
func (s *Store) ConsumeToken(ctx context.Context, purpose, hash string) (int64, error) {
	var userID int64
	err := s.pool.QueryRow(ctx,
		`UPDATE tokens SET used_at = now()
		  WHERE token_hash = $1 AND purpose = $2
		    AND used_at IS NULL AND expires_at > now()
		 RETURNING user_id`, hash, purpose).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("consume token: %w", err)
	}
	return userID, nil
}

// --- sessions --------------------------------------------------------------

func (s *Store) CreateSession(ctx context.Context, userID int64, hash, userAgent string, ttl time.Duration) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO sessions (user_id, token_hash, user_agent, expires_at)
		 VALUES ($1, $2, $3, now() + $4::interval)`,
		userID, hash, userAgent, fmt.Sprintf("%d seconds", int(ttl.Seconds())))
	return err
}

// RotateSession swaps a refresh token for a new one and reports reuse.
//
// If the presented token was already rotated away, two parties hold it: the
// legitimate user and whoever copied it. We cannot tell which is asking, so
// every session for that user is revoked and both must log in again. Annoying
// once, versus an attacker holding a session for thirty days.
func (s *Store) RotateSession(ctx context.Context, oldHash, newHash, userAgent string, ttl time.Duration) (int64, error) {
	var userID int64
	var revokedAt *time.Time
	err := s.pool.QueryRow(ctx,
		`SELECT user_id, revoked_at FROM sessions
		  WHERE token_hash = $1 AND expires_at > now()`, oldHash).Scan(&userID, &revokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("lookup session: %w", err)
	}

	if revokedAt != nil {
		_, _ = s.pool.Exec(ctx,
			`UPDATE sessions SET reused_at = now() WHERE token_hash = $1`, oldHash)
		_ = s.RevokeAllSessions(ctx, userID)
		return userID, fmt.Errorf("refresh token reuse detected for user %d", userID)
	}

	batch := &pgx.Batch{}
	batch.Queue(`UPDATE sessions SET revoked_at = now() WHERE token_hash = $1`, oldHash)
	batch.Queue(`INSERT INTO sessions (user_id, token_hash, user_agent, expires_at)
	             VALUES ($1, $2, $3, now() + $4::interval)`,
		userID, newHash, userAgent, fmt.Sprintf("%d seconds", int(ttl.Seconds())))
	if err := s.pool.SendBatch(ctx, batch).Close(); err != nil {
		return 0, fmt.Errorf("rotate session: %w", err)
	}
	return userID, nil
}

func (s *Store) RevokeSession(ctx context.Context, hash string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE sessions SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL`, hash)
	return err
}

func (s *Store) RevokeAllSessions(ctx context.Context, userID int64) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}

// --- saved searches --------------------------------------------------------

type SavedSearch struct {
	ID         int64      `json:"id"`
	Label      *string    `json:"label"`
	Query      string     `json:"query"`
	Mode       string     `json:"mode"`
	AlertPct   *float64   `json:"alert_pct"`
	CreatedAt  time.Time  `json:"created_at"`
	LastRunAt  *time.Time `json:"last_run_at"`
	LastMedian *int64     `json:"last_median"`
}

func (s *Store) CreateSavedSearch(ctx context.Context, userID int64, label *string, query, mode string, alertPct *float64) (*SavedSearch, error) {
	ss := &SavedSearch{Label: label, Query: query, Mode: mode, AlertPct: alertPct}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO saved_searches (user_id, label, query, mode, alert_pct)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
		userID, label, query, mode, alertPct).Scan(&ss.ID, &ss.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create saved search: %w", err)
	}
	return ss, nil
}

func (s *Store) ListSavedSearches(ctx context.Context, userID int64) ([]SavedSearch, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, label, query, mode, alert_pct, created_at, last_run_at, last_median
		   FROM saved_searches WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list saved searches: %w", err)
	}
	defer rows.Close()
	out := []SavedSearch{}
	for rows.Next() {
		var ss SavedSearch
		if err := rows.Scan(&ss.ID, &ss.Label, &ss.Query, &ss.Mode, &ss.AlertPct,
			&ss.CreatedAt, &ss.LastRunAt, &ss.LastMedian); err != nil {
			return nil, fmt.Errorf("scan saved search: %w", err)
		}
		out = append(out, ss)
	}
	return out, rows.Err()
}

// DeleteSavedSearch scopes the delete by user_id as well as id, so an
// authenticated user cannot delete somebody else's row by guessing a number.
func (s *Store) DeleteSavedSearch(ctx context.Context, userID, id int64) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM saved_searches WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete saved search: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- contact ---------------------------------------------------------------

func (s *Store) CreateContactMessage(ctx context.Context, name, email, subject, body, ipPrefix string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO contact_messages (name, email, subject, body, ip_prefix)
		 VALUES ($1, $2, $3, $4, $5)`, name, email, subject, body, ipPrefix)
	return err
}

// --- subscribers -----------------------------------------------------------

type Subscriber struct {
	ID             int64      `json:"id"`
	Email          string     `json:"email"`
	ConfirmedAt    *time.Time `json:"confirmed_at"`
	UnsubscribedAt *time.Time `json:"unsubscribed_at"`
	Source         string     `json:"source"`
	CreatedAt      time.Time  `json:"created_at"`
}

const subscriberCols = `id, email, confirmed_at, unsubscribed_at, source, created_at`

func scanSubscribers(rows pgx.Rows) ([]Subscriber, error) {
	defer rows.Close()
	out := []Subscriber{}
	for rows.Next() {
		var sub Subscriber
		if err := rows.Scan(&sub.ID, &sub.Email, &sub.ConfirmedAt, &sub.UnsubscribedAt,
			&sub.Source, &sub.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan subscriber: %w", err)
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

// resendWindow is how long a pending address is left in peace before another
// confirmation mail is owed. Long enough that a form retyped in frustration
// sends nothing, short enough that somebody who genuinely lost the message can
// ask for it again over a coffee.
const resendWindow = 15 * time.Minute

// maxUnansweredSends is the ceiling the window alone did not provide.
//
// The window throttles; it does not stop. The reset arm of the upsert carried
// no time gate at all, so an address that had confirmed and later left could be
// pushed back to pending once per window indefinitely — measured at four mails
// across four windows, which is ninety-six a day to somebody who had already
// asked to go. Somebody who ignores three confirmation mails is answering.
const maxUnansweredSends = 3

// SubscribePending records an unconfirmed subscription and reports whether a
// confirmation mail is owed.
//
// The conflict clause carries every rule that makes this list legal, and each
// one is a case the previous version got wrong or right for a reason:
//
// Confirmed and still subscribed — nothing happens. Without that guard anyone
// typing a stranger's address into the form could clear their consent until
// they re-confirmed.
//
// Confirmed, then unsubscribed — a full reset, because somebody who left and
// came back is giving consent again rather than resuming the consent they
// withdrew. This is the only way back onto the list.
//
// Never confirmed, then unsubscribed — nothing happens, ever. That row belongs
// to someone who clicked «حذف کامل این نشانی» in a message they never asked
// for, and the mail called it complete removal. Lumping it in with the case
// above meant the next person to type that address into the box reset the row
// and mailed them again. The cost of getting this right is that the address is
// permanently suppressed: a genuine later signup from that person is answered
// with the same silent 202 as everything else, and they have to write in.
//
// Never confirmed, still subscribed — one mail per resendWindow. Before that
// clock existed every POST rotated the token and reported a mail owed, so the
// endpoint was an unbounded confirmation-mail amplifier pointed at whichever
// address the sender chose. Outside the window the token is left alone as
// well, which is what keeps the link in the message already sitting in their
// inbox working.
func (s *Store) SubscribePending(ctx context.Context, email, tokenHash, source string) (bool, error) {
	var id int64
	err := s.pool.QueryRow(ctx,
		`INSERT INTO subscribers (email, token_hash, source, last_sent_at, sends)
		 VALUES ($1, $2, $3, now(), 1)
		 ON CONFLICT (email) DO UPDATE
		    SET token_hash = EXCLUDED.token_hash,
		        confirmed_at = NULL,
		        unsubscribed_at = NULL,
		        source = EXCLUDED.source,
		        last_sent_at = now(),
		        sends = subscribers.sends + 1
		  WHERE subscribers.sends < `+strconv.Itoa(maxUnansweredSends)+`
		    AND ((subscribers.confirmed_at IS NOT NULL AND subscribers.unsubscribed_at IS NOT NULL)
		      OR (subscribers.confirmed_at IS NULL AND subscribers.unsubscribed_at IS NULL
		          AND (subscribers.last_sent_at IS NULL
		               OR subscribers.last_sent_at < now() - $4::interval)))
		 RETURNING id`,
		NormalizeEmail(email), tokenHash, source,
		fmt.Sprintf("%d seconds", int(resendWindow.Seconds()))).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("subscribe: %w", err)
	}
	return true, nil
}

// AddConfirmedSubscriber puts a consent that was given somewhere else onto the
// list, already confirmed.
//
// The one caller is registration with the marketing box ticked. That tick is
// the same act a double opt-in mail exists to collect — a form the person
// filled in themselves — so asking them to confirm it a second time would only
// mean sending a second message to an address that is already receiving its
// verification link.
//
// The conflict clause upgrades a pending row in place rather than replacing it:
// the token stays whatever was mailed out, because that link is the unsubscribe
// link in their inbox. An unsubscribed row is left alone for the reason spelled
// out above — nothing puts a person who left back on the list except their own
// return through the newsletter box.
func (s *Store) AddConfirmedSubscriber(ctx context.Context, email, tokenHash, source string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO subscribers (email, token_hash, source, confirmed_at)
		 VALUES ($1, $2, $3, now())
		 ON CONFLICT (email) DO UPDATE SET confirmed_at = now(), sends = 0
		  WHERE subscribers.confirmed_at IS NULL AND subscribers.unsubscribed_at IS NULL`,
		NormalizeEmail(email), tokenHash, source)
	if err != nil {
		return fmt.Errorf("add confirmed subscriber: %w", err)
	}
	return nil
}

// ConfirmSubscriber turns a claimed address into a consented one.
//
// COALESCE rather than a used-once guard, because people double-click links in
// mail clients and being told "this link is already used" for something that
// worked is a support ticket. An unsubscribed row is excluded so an old
// confirmation link cannot resurrect someone who left.
func (s *Store) ConfirmSubscriber(ctx context.Context, tokenHash string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE subscribers SET confirmed_at = COALESCE(confirmed_at, now())
		  WHERE token_hash = $1 AND unsubscribed_at IS NULL`, tokenHash)
	if err != nil {
		return fmt.Errorf("confirm subscriber: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Unsubscribe is idempotent, and keeps the first timestamp.
//
// This link is clicked from a mail client, sometimes twice, sometimes by the
// client's own link scanner. The second click must behave exactly like the
// first — an error page here reads as "it didn't work" and the next step is a
// spam complaint.
func (s *Store) Unsubscribe(ctx context.Context, tokenHash string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE subscribers SET unsubscribed_at = COALESCE(unsubscribed_at, now())
		  WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return fmt.Errorf("unsubscribe: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetMarketingConsent is the exit for consent given by ticking the register box.
//
// Unsubscribe matches on a token, and the rows backfilled for people who ticked
// that box before tokens were issued carry a deliberately unusable placeholder,
// so no secret can ever reach them. They all belong to registered accounts
// though, and an authenticated request naming its own address proves as much as
// a mailed secret does. Without this they were mailable with no way off, which
// is worse than the state it replaced, where they at least reached no campaign.
//
// Both tables move together, so the checkbox on the account page and the export
// can never disagree about whether someone wants to hear from us.
func (s *Store) SetMarketingConsent(ctx context.Context, userID int64, on bool) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("marketing consent: %w", err)
	}
	defer tx.Rollback(ctx)

	var email string
	if err := tx.QueryRow(ctx,
		`UPDATE users SET marketing_consent = $2,
		        marketing_consent_at = CASE WHEN $2 THEN now() ELSE marketing_consent_at END
		  WHERE id = $1 RETURNING email`, userID, on).Scan(&email); err != nil {
		return fmt.Errorf("marketing consent: %w", err)
	}

	if on {
		// Turning it back on re-confirms rather than re-mailing: the account is
		// already verified, which is the thing a confirmation mail establishes.
		_, err = tx.Exec(ctx,
			`UPDATE subscribers SET unsubscribed_at = NULL, confirmed_at = COALESCE(confirmed_at, now()),
			        sends = 0
			  WHERE email = $1`, email)
	} else {
		_, err = tx.Exec(ctx,
			`UPDATE subscribers SET unsubscribed_at = COALESCE(unsubscribed_at, now())
			  WHERE email = $1`, email)
	}
	if err != nil {
		return fmt.Errorf("marketing consent: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *Store) ListSubscribers(ctx context.Context, limit, offset int) ([]Subscriber, int64, error) {
	var total int64
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM subscribers`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count subscribers: %w", err)
	}
	rows, err := s.pool.Query(ctx,
		`SELECT `+subscriberCols+` FROM subscribers
		  ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list subscribers: %w", err)
	}
	list, err := scanSubscribers(rows)
	return list, total, err
}

// mailableWhere is the definition of "may be mailed", written once.
//
// The export and the number on the dashboard both claim to be this list. Two
// copies of the predicate is one copy too many: the day they drift, the admin
// page promises a list the CSV does not deliver, and nothing in either place
// looks wrong.
const mailableWhere = `confirmed_at IS NOT NULL AND unsubscribed_at IS NULL`

// MailableSubscribers is the export, and the filter is the entire point of it.
//
// Confirmed and not unsubscribed: the list that can legally be mailed. Anything
// wider is a spam run wearing a CSV extension, and the sending domain pays for
// it long after the campaign.
func (s *Store) MailableSubscribers(ctx context.Context) ([]Subscriber, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+subscriberCols+` FROM subscribers
		  WHERE `+mailableWhere+`
		  ORDER BY confirmed_at`)
	if err != nil {
		return nil, fmt.Errorf("mailable subscribers: %w", err)
	}
	return scanSubscribers(rows)
}

// --- admin -----------------------------------------------------------------

// AdminUser is the account as an operator sees it: no password hash, no lockout
// counters, nothing that would turn the admin page into a credential leak.
type AdminUser struct {
	ID               int64      `json:"id"`
	Email            string     `json:"email"`
	DisplayName      *string    `json:"display_name"`
	Verified         bool       `json:"verified"`
	IsAdmin          bool       `json:"is_admin"`
	MarketingConsent bool       `json:"marketing_consent"`
	CreatedAt        time.Time  `json:"created_at"`
	LastLoginAt      *time.Time `json:"last_login_at"`
}

func (s *Store) ListUsers(ctx context.Context, limit, offset int) ([]AdminUser, int64, error) {
	var total int64
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, email, display_name, verified_at IS NOT NULL, is_admin,
		        marketing_consent, created_at, last_login_at
		   FROM users ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	out := []AdminUser{}
	for rows.Next() {
		var u AdminUser
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.Verified, &u.IsAdmin,
			&u.MarketingConsent, &u.CreatedAt, &u.LastLoginAt); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		out = append(out, u)
	}
	return out, total, rows.Err()
}

type Counts struct {
	Users          int64 `json:"users"`
	Verified       int64 `json:"verified"`
	SavedSearches  int64 `json:"saved_searches"`
	ActiveSessions int64 `json:"active_sessions"`
	UnreadContact  int64 `json:"unread_contact"`
	Subscribers    int64 `json:"subscribers"`
	// The length of the export, counted by the same predicate the export
	// selects on — this is the number that can actually be mailed, and it is
	// the only one worth putting on a dashboard. Total minus this is the funnel
	// loss, and it is visible from the two figures together.
	SubscribersConfirmed int64 `json:"subscribers_confirmed"`
	// Accounts that ticked the marketing box. Since registration started
	// writing a confirmed subscriber row, every one of these is already inside
	// SubscribersConfirmed: this is a breakdown of that number, never something
	// to add to it.
	MarketingOptin int64 `json:"marketing_optin"`
}

func (s *Store) Counts(ctx context.Context) (*Counts, error) {
	var c Counts
	err := s.pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM users),
		       (SELECT count(*) FROM users WHERE verified_at IS NOT NULL),
		       (SELECT count(*) FROM saved_searches),
		       (SELECT count(*) FROM sessions WHERE revoked_at IS NULL AND expires_at > now()),
		       (SELECT count(*) FROM contact_messages WHERE handled_at IS NULL),
		       (SELECT count(*) FROM subscribers),
		       (SELECT count(*) FROM subscribers WHERE `+mailableWhere+`),
		       (SELECT count(*) FROM users WHERE marketing_consent)`,
	).Scan(&c.Users, &c.Verified, &c.SavedSearches, &c.ActiveSessions, &c.UnreadContact,
		&c.Subscribers, &c.SubscribersConfirmed, &c.MarketingOptin)
	if err != nil {
		return nil, fmt.Errorf("counts: %w", err)
	}
	return &c, nil
}
