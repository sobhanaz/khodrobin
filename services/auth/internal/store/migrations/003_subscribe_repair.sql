-- Three holes in the subscribe flow, closed where the rules actually live.
--
-- Idempotent like everything before it: there is no migration table, so this
-- file runs again on every boot against a database that already holds live
-- rows, and "already applied" has to be expressed by the statement itself.

-- The brake on confirmation mail. Without a record of when the last one went
-- out, every POST for an unconfirmed address rotated the token and sent
-- another — and since the rate limit is per path and IP, one form plus one
-- stranger's address was an amplifier with this domain's name on the envelope.
-- Existing rows start NULL, which reads as "never mailed": the first signup
-- after this lands sends one message and stamps the clock.
ALTER TABLE subscribers ADD COLUMN IF NOT EXISTS last_sent_at TIMESTAMPTZ;

-- The clock alone was not a brake, only a throttle. The reset arm of the upsert
-- carried no time gate at all, so an address that had confirmed and later
-- unsubscribed could be pushed back to pending once per window, forever: about
-- ninety-six confirmation mails a day to somebody who had already left.
-- This counts confirmation mails nobody answered, and three is where it stops.
-- Confirming resets it, so a person who genuinely comes back gets a fresh three
-- rather than inheriting the silence of whoever typed their address in before.
ALTER TABLE subscribers ADD COLUMN IF NOT EXISTS sends INT NOT NULL DEFAULT 0;

-- The label is written straight into the CSV. csv.Writer quotes an embedded
-- newline correctly, across two physical lines — and the admin page counts
-- exported addresses by splitting on newline, so one crafted label tells the
-- operator that more people are on the list than are. Rejected at the door
-- now; this is the handful that got in before the door existed.
-- [[:cntrl:]] is category Cc only, while the handler rejects everything that is
-- not printable — which includes the Cf bidi overrides its own comment names.
-- The repair was narrower than the validator it was repairing, so U+202E went
-- in and stayed in. Matching on "not printable" makes the two agree.
UPDATE subscribers SET source = 'landing' WHERE source !~ '^[[:print:]]*$';

-- Consent given by ticking the box at registration was recorded on users and
-- nowhere else: it reached no export, and with no token behind it «لغوش هم یک
-- کلیک است» was a promise with no mechanism. One row here gives that consent
-- the same list, the same export and the same one-click exit as an address
-- typed into the newsletter box.
--
-- ON CONFLICT DO NOTHING is what keeps this safe to re-run and safe for people
-- who already have a subscriber row — including anyone who has since left,
-- whose row must not be resurrected by a registration.
--
-- The token is deliberately unusable: md5 is 32 hex characters and a real
-- fingerprint is sha256's 64, so no secret anyone could present will ever hash
-- to it. These rows predate the token being issued at registration; the column
-- is NOT NULL and unique, and a placeholder nobody can guess is the honest
-- filling for it.
--
-- Which would leave them on the list with no way off, so the exit for these is
-- not a token at all: every one of them belongs to a registered account, and
-- the account page turns marketing off through an authenticated route that
-- matches on the address rather than a secret. New opt-ins additionally get a
-- real token, carried out in the welcome mail.
INSERT INTO subscribers (email, token_hash, source, confirmed_at, created_at)
SELECT u.email,
       md5(random()::text || clock_timestamp()::text || u.email),
       'register',
       COALESCE(u.marketing_consent_at, u.created_at),
       u.created_at
  FROM users u
 WHERE u.marketing_consent
ON CONFLICT (email) DO NOTHING;
