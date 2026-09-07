-- Accounts, and the two product features that justify having them.
--
-- Decision ۷ said this product needs no user accounts, and for pure search that
-- was right. What changes it is saved searches and price alerts: Torob has
-- «پیگیری قیمت», and a used-car price that moves is exactly the thing a buyer
-- wants told to them rather than having to come back and check. Accounts exist
-- to carry that, not to gate the search.
--
-- Search itself stays anonymous. Nothing here is required to use the product.

CREATE TABLE IF NOT EXISTS users (
    id              BIGSERIAL PRIMARY KEY,
    -- Stored lowercased and trimmed by the application; the unique index is on
    -- the stored value so two casings cannot become two accounts.
    email           TEXT        NOT NULL,
    password_hash   TEXT        NOT NULL,
    display_name    TEXT,
    -- NULL until the address is proven. An unverified account can log in but
    -- cannot receive alerts, because sending mail to an unconfirmed address is
    -- how a product becomes a spam vector.
    verified_at     TIMESTAMPTZ,
    is_admin        BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_login_at   TIMESTAMPTZ,
    -- Cheap brute-force brake that survives a restart, unlike an in-memory one.
    failed_logins   INTEGER     NOT NULL DEFAULT 0,
    locked_until    TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_key ON users (email);

-- One row per issued refresh token, so a session can be revoked individually
-- and a stolen token can be detected on reuse.
CREATE TABLE IF NOT EXISTS sessions (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- The token itself is never stored. A database dump must not hand anyone a
    -- working session, which is the same reason passwords are hashed.
    token_hash  TEXT        NOT NULL,
    user_agent  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    -- Set when a token is presented after it was already rotated. That means
    -- two parties hold it, so every session for the user is killed.
    reused_at   TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS sessions_token_hash_key ON sessions (token_hash);
CREATE INDEX IF NOT EXISTS sessions_user_id_idx ON sessions (user_id);

-- Email verification and password reset share a table because they are the same
-- shape: a single-use secret with a short life, delivered out of band.
CREATE TABLE IF NOT EXISTS tokens (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose     TEXT        NOT NULL CHECK (purpose IN ('verify_email', 'reset_password')),
    token_hash  TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS tokens_token_hash_key ON tokens (token_hash);
CREATE INDEX IF NOT EXISTS tokens_user_purpose_idx ON tokens (user_id, purpose);

-- A saved search is the query a user typed, kept so we can re-run it. Storing
-- the raw text rather than the parsed intent means an improvement to the parser
-- improves every saved search retroactively.
CREATE TABLE IF NOT EXISTS saved_searches (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label       TEXT,
    query       TEXT        NOT NULL,
    mode        TEXT        NOT NULL DEFAULT 'relevant',
    -- Email when a matching spec's median moves by more than this. NULL = never.
    alert_pct   NUMERIC(5,2),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_run_at TIMESTAMPTZ,
    -- Median at the last run, so a change can be measured rather than guessed.
    last_median BIGINT
);

CREATE INDEX IF NOT EXISTS saved_searches_user_idx ON saved_searches (user_id);
CREATE INDEX IF NOT EXISTS saved_searches_alerting_idx ON saved_searches (alert_pct)
    WHERE alert_pct IS NOT NULL;

-- Every alert we send, so a user is never mailed the same movement twice and we
-- can answer "why did I get this".
CREATE TABLE IF NOT EXISTS alerts_sent (
    id              BIGSERIAL PRIMARY KEY,
    saved_search_id BIGINT      NOT NULL REFERENCES saved_searches(id) ON DELETE CASCADE,
    spec_key        TEXT        NOT NULL,
    old_median      BIGINT      NOT NULL,
    new_median      BIGINT      NOT NULL,
    sent_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS alerts_sent_search_idx ON alerts_sent (saved_search_id, sent_at DESC);

-- Contact-form submissions. Kept in the database rather than only emailed so a
-- message is not lost when SMTP is down, which is precisely when someone is
-- most likely to be writing in about a problem.
CREATE TABLE IF NOT EXISTS contact_messages (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT        NOT NULL,
    email       TEXT        NOT NULL,
    subject     TEXT,
    body        TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    handled_at  TIMESTAMPTZ,
    -- Truncated to a /24 and kept only for abuse triage, never displayed.
    ip_prefix   TEXT
);

CREATE INDEX IF NOT EXISTS contact_messages_unhandled_idx ON contact_messages (created_at DESC)
    WHERE handled_at IS NULL;
