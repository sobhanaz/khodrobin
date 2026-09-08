-- The marketing list, and the consent that makes it mailable.
--
-- The register page promises «هیچ ایمیل تبلیغاتی‌ای نمی‌فرستیم». That promise is
-- kept by making consent a separate, explicit, timestamped act rather than a
-- side effect of having an account: a boolean alone cannot answer "when did
-- they agree", which is the only question that matters when someone complains.
--
-- Every statement is idempotent because this file runs on every boot against a
-- database that already holds live rows — there is no migration table, so
-- "already applied" has to be expressed by the statement itself.

ALTER TABLE users ADD COLUMN IF NOT EXISTS marketing_consent    BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS marketing_consent_at TIMESTAMPTZ;

-- Subscribers are deliberately not users. Most people who want price news never
-- want an account, and a list that costs a registration is a list that stays
-- empty.
CREATE TABLE IF NOT EXISTS subscribers (
    id              BIGSERIAL PRIMARY KEY,
    -- Lowercased and trimmed by the application, like users.email, so one
    -- person cannot occupy two rows and receive everything twice.
    email           TEXT        NOT NULL UNIQUE,
    -- One secret per row, doing double duty: it confirms the subscription and
    -- it is the unsubscribe link in every message afterwards. Stored as its
    -- SHA-256 like every other link secret here — a database dump must not let
    -- anyone confirm or cancel somebody else's address.
    token_hash      TEXT        NOT NULL,
    -- NULL until the link in the mail is clicked. Anything before that is an
    -- address someone typed into a form, not consent — possibly not even theirs.
    confirmed_at    TIMESTAMPTZ,
    -- Set, never deleted. A row is kept after unsubscribing so a re-import
    -- cannot quietly resurrect someone who left.
    unsubscribed_at TIMESTAMPTZ,
    source          TEXT        NOT NULL DEFAULT 'landing',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Unique, matching tokens and sessions: two rows sharing a token would mean the
-- unsubscribe link cancels an address at random.
CREATE UNIQUE INDEX IF NOT EXISTS subscribers_token_hash_key ON subscribers (token_hash);

-- The export reads exactly this slice — confirmed and not unsubscribed — so it
-- is the one that gets an index.
CREATE INDEX IF NOT EXISTS subscribers_mailable_idx ON subscribers (confirmed_at)
    WHERE confirmed_at IS NOT NULL AND unsubscribed_at IS NULL;
