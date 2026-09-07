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

func (s *Store) CreateUser(ctx context.Context, email, hash string, name *string) (*User, error) {
	u := &User{Email: NormalizeEmail(email), PasswordHash: hash, DisplayName: name}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, display_name)
		 VALUES ($1, $2, $3) RETURNING id, created_at, is_admin`,
		u.Email, hash, name,
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
                  created_at, last_login_at, failed_logins, locked_until`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.VerifiedAt,
		&u.IsAdmin, &u.CreatedAt, &u.LastLoginAt, &u.FailedLogins, &u.LockedUntil)
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

// --- admin -----------------------------------------------------------------

type Counts struct {
	Users          int64 `json:"users"`
	Verified       int64 `json:"verified"`
	SavedSearches  int64 `json:"saved_searches"`
	ActiveSessions int64 `json:"active_sessions"`
	UnreadContact  int64 `json:"unread_contact"`
}

func (s *Store) Counts(ctx context.Context) (*Counts, error) {
	var c Counts
	err := s.pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM users),
		       (SELECT count(*) FROM users WHERE verified_at IS NOT NULL),
		       (SELECT count(*) FROM saved_searches),
		       (SELECT count(*) FROM sessions WHERE revoked_at IS NULL AND expires_at > now()),
		       (SELECT count(*) FROM contact_messages WHERE handled_at IS NULL)`,
	).Scan(&c.Users, &c.Verified, &c.SavedSearches, &c.ActiveSessions, &c.UnreadContact)
	if err != nil {
		return nil, fmt.Errorf("counts: %w", err)
	}
	return &c, nil
}
