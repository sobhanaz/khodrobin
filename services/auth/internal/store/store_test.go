package store_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sobhanaz/khodrobin/auth/internal/store"
	"github.com/sobhanaz/khodrobin/auth/internal/store/storetest"
	"github.com/sobhanaz/khodrobin/auth/internal/tokens"
)

func TestMain(m *testing.M) { os.Exit(storetest.Run(m)) }

// fingerprint is what the row actually stores. The tests hold the raw secret,
// exactly as an email does, and never the hash.
func fingerprint(secret string) string { return tokens.Fingerprint(secret) }

func TestEveryMigrationIsEmbeddedInOrder(t *testing.T) {
	// The embed used to name 001_init.sql directly. The day a second file was
	// added it would have sat in the repository looking applied and never run,
	// so this pins the glob rather than the filename.
	init001 := strings.Index(store.Schema, "CREATE TABLE IF NOT EXISTS users")
	marketing002 := strings.Index(store.Schema, "CREATE TABLE IF NOT EXISTS subscribers")
	repair003 := strings.Index(store.Schema, "ADD COLUMN IF NOT EXISTS last_sent_at")
	if init001 < 0 {
		t.Fatal("001_init.sql is missing from the embedded schema")
	}
	if marketing002 < 0 {
		t.Fatal("002_marketing.sql is missing from the embedded schema")
	}
	if repair003 < 0 {
		t.Fatal("003_subscribe_repair.sql is missing from the embedded schema")
	}
	// Each file alters what the one before it created, so a swapped order is a
	// boot that fails on a fresh database and nowhere else.
	if marketing002 < init001 {
		t.Error("002 is concatenated before 001")
	}
	if repair003 < marketing002 {
		t.Error("003 is concatenated before 002")
	}
}

func TestSchemaSurvivesASecondApply(t *testing.T) {
	// Production has live rows and no migration table, so every boot re-runs the
	// whole file. A single statement missing IF NOT EXISTS is a container that
	// crash-loops on deploy with the database intact.
	st := storetest.New(t)
	ctx := context.Background()

	user, err := st.CreateUser(ctx, "live@khodrobin.test", "hash", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Migrate(ctx, store.Schema); err != nil {
		t.Fatalf("second apply: %v", err)
	}
	again, err := st.UserByID(ctx, user.ID)
	if err != nil || !again.MarketingConsent {
		t.Fatalf("the row did not survive re-applying the schema: %v", err)
	}
}

func TestConsentIsStampedOnlyWhenItIsGiven(t *testing.T) {
	st := storetest.New(t)
	ctx := context.Background()

	for _, tc := range []struct {
		name      string
		email     string
		consented bool
	}{
		{"opted in", "yes@khodrobin.test", true},
		{"declined", "no@khodrobin.test", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u, err := st.CreateUser(ctx, tc.email, "hash", nil, tc.consented)
			if err != nil {
				t.Fatal(err)
			}
			// Read raw: a consent flag with no date behind it is exactly the
			// state that cannot be defended when somebody complains, and Store
			// has no reason to expose the column otherwise.
			conn, err := pgx.Connect(ctx, storetest.DSN())
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close(ctx)
			var at *time.Time
			if err := conn.QueryRow(ctx,
				`SELECT marketing_consent_at FROM users WHERE id = $1`, u.ID).Scan(&at); err != nil {
				t.Fatal(err)
			}
			if tc.consented && at == nil {
				t.Error("consent recorded with no timestamp")
			}
			if !tc.consented && at != nil {
				t.Error("a declined opt-in was stamped as consent")
			}
		})
	}
}

// subscribe is the two-line dance every subscriber test starts with: a secret
// the test knows, and the row it creates.
func subscribe(t *testing.T, st *store.Store, email, secret string) bool {
	t.Helper()
	owed, err := st.SubscribePending(context.Background(), email, fingerprint(secret), "landing")
	if err != nil {
		t.Fatal(err)
	}
	return owed
}

func mailable(t *testing.T, st *store.Store) []string {
	t.Helper()
	list, err := st.MailableSubscribers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(list))
	for _, s := range list {
		out = append(out, s.Email)
	}
	return out
}

func TestAnUnconfirmedAddressIsNotOnTheList(t *testing.T) {
	// The entire promise of double opt-in: typing a stranger's address into the
	// form gets them one email and no list entry.
	st := storetest.New(t)
	subscribe(t, st, "typed@khodrobin.test", "secret-a")
	if got := mailable(t, st); len(got) != 0 {
		t.Errorf("mailable = %v, want none before the link is clicked", got)
	}
	if err := st.ConfirmSubscriber(context.Background(), fingerprint("some-other-secret")); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("a wrong token confirmed something: %v", err)
	}
	if got := mailable(t, st); len(got) != 0 {
		t.Errorf("mailable = %v after a wrong token", got)
	}
}

func TestConfirmingTwiceKeepsTheFirstConsent(t *testing.T) {
	// Mail clients prefetch links and people double-click. The second click has
	// to be a success, and it must not rewrite the date consent was given.
	st := storetest.New(t)
	ctx := context.Background()
	subscribe(t, st, "keen@khodrobin.test", "secret-b")
	if err := st.ConfirmSubscriber(ctx, fingerprint("secret-b")); err != nil {
		t.Fatal(err)
	}
	first := mailableAt(t, st, "keen@khodrobin.test")
	if err := st.ConfirmSubscriber(ctx, fingerprint("secret-b")); err != nil {
		t.Fatalf("second click errored: %v", err)
	}
	if second := mailableAt(t, st, "keen@khodrobin.test"); !second.Equal(first) {
		t.Errorf("confirmed_at moved from %v to %v", first, second)
	}
}

func mailableAt(t *testing.T, st *store.Store, email string) time.Time {
	t.Helper()
	list, err := st.MailableSubscribers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range list {
		if s.Email == email && s.ConfirmedAt != nil {
			return *s.ConfirmedAt
		}
	}
	t.Fatalf("%s is not on the mailable list", email)
	return time.Time{}
}

func TestUnsubscribeIsIdempotent(t *testing.T) {
	st := storetest.New(t)
	ctx := context.Background()
	subscribe(t, st, "leaving@khodrobin.test", "secret-c")
	if err := st.ConfirmSubscriber(ctx, fingerprint("secret-c")); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		if err := st.Unsubscribe(ctx, fingerprint("secret-c")); err != nil {
			t.Fatalf("click %d errored: %v", i, err)
		}
	}
	if got := mailable(t, st); len(got) != 0 {
		t.Errorf("mailable = %v, want none after unsubscribing", got)
	}
	if err := st.Unsubscribe(ctx, fingerprint("never-issued")); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("an unknown token reported success: %v", err)
	}
}

func TestComingBackIsFreshConsent(t *testing.T) {
	// Someone who left and returned is consenting again, not resuming the
	// consent they withdrew — so the row must go back to unconfirmed and the
	// old link must stop working.
	st := storetest.New(t)
	ctx := context.Background()
	const email = "boomerang@khodrobin.test"

	subscribe(t, st, email, "old-secret")
	if err := st.ConfirmSubscriber(ctx, fingerprint("old-secret")); err != nil {
		t.Fatal(err)
	}
	if err := st.Unsubscribe(ctx, fingerprint("old-secret")); err != nil {
		t.Fatal(err)
	}

	if owed := subscribe(t, st, email, "new-secret"); !owed {
		t.Fatal("a returning address was not owed a fresh confirmation mail")
	}
	if got := mailable(t, st); len(got) != 0 {
		t.Errorf("mailable = %v, want none until the new link is clicked", got)
	}
	if err := st.ConfirmSubscriber(ctx, fingerprint("old-secret")); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("the retired link still confirmed: %v", err)
	}
	if err := st.ConfirmSubscriber(ctx, fingerprint("new-secret")); err != nil {
		t.Fatal(err)
	}
	if got := mailable(t, st); len(got) != 1 || got[0] != email {
		t.Errorf("mailable = %v, want just the returning address", got)
	}
}

func TestASecondSignupCannotKnockSomebodyOff(t *testing.T) {
	// Without the guard on the conflict clause, anyone could type a confirmed
	// address into the box and clear its consent until the owner re-clicked.
	st := storetest.New(t)
	ctx := context.Background()
	const email = "settled@khodrobin.test"

	subscribe(t, st, email, "secret-d")
	if err := st.ConfirmSubscriber(ctx, fingerprint("secret-d")); err != nil {
		t.Fatal(err)
	}
	if owed := subscribe(t, st, email, "attacker-secret"); owed {
		t.Error("a confirmed address was owed another confirmation mail")
	}
	if got := mailable(t, st); len(got) != 1 {
		t.Fatalf("mailable = %v, want the address to have stayed on the list", got)
	}
	// The old link still owns the row, which is what keeps the unsubscribe link
	// in every past email working.
	if err := st.Unsubscribe(ctx, fingerprint("secret-d")); err != nil {
		t.Errorf("the original unsubscribe link stopped working: %v", err)
	}
}

func TestTheExportIsOnlyTheMailableSlice(t *testing.T) {
	st := storetest.New(t)
	ctx := context.Background()

	for _, tc := range []struct {
		email   string
		secret  string
		confirm bool
		unsub   bool
	}{
		{"confirmed@khodrobin.test", "s1", true, false},
		{"pending@khodrobin.test", "s2", false, false},
		{"left@khodrobin.test", "s3", true, true},
		{"never-clicked-then-left@khodrobin.test", "s4", false, true},
	} {
		subscribe(t, st, tc.email, tc.secret)
		if tc.confirm {
			if err := st.ConfirmSubscriber(ctx, fingerprint(tc.secret)); err != nil {
				t.Fatal(err)
			}
		}
		if tc.unsub {
			if err := st.Unsubscribe(ctx, fingerprint(tc.secret)); err != nil {
				t.Fatal(err)
			}
		}
	}

	got := mailable(t, st)
	if len(got) != 1 || got[0] != "confirmed@khodrobin.test" {
		t.Errorf("export = %v, want only the confirmed and still-subscribed address", got)
	}

	// The admin table shows everyone, including the people who left; the export
	// is the narrower list, and the dashboard needs both numbers to see the gap.
	all, total, err := st.ListSubscribers(ctx, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 4 || len(all) != 4 {
		t.Errorf("admin list = %d rows (total %d), want 4", len(all), total)
	}

	counts, err := st.Counts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if counts.Subscribers != 4 || counts.SubscribersConfirmed != 1 {
		t.Errorf("counts subscribers=%d confirmed=%d, want 4 and 1",
			counts.Subscribers, counts.SubscribersConfirmed)
	}

	if _, err := st.CreateUser(ctx, "optin@khodrobin.test", "h", nil, true); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateUser(ctx, "optout@khodrobin.test", "h", nil, false); err != nil {
		t.Fatal(err)
	}
	counts, err = st.Counts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if counts.MarketingOptin != 1 {
		t.Errorf("marketing_optin = %d, want 1", counts.MarketingOptin)
	}
}

// --- the four defects the subscribe flow was carrying ------------------------

func TestACompleteRemovalIsNotUndoneByTheNextSignup(t *testing.T) {
	// The confirmation mail offers «حذف کامل این نشانی» to whoever the address
	// actually belongs to, and calls it complete removal. Before this, the
	// predicate treated "never confirmed, then removed" as the same case as
	// "confirmed, then left": the next person to type the address into the box
	// reset the row and mailed them all over again.
	st := storetest.New(t)
	ctx := context.Background()
	const email = "never-asked-for-this@khodrobin.test"

	subscribe(t, st, email, "unwanted")
	if err := st.Unsubscribe(ctx, fingerprint("unwanted")); err != nil {
		t.Fatal(err)
	}
	// Past the resend window, so the only thing standing between this address
	// and another mail is the rule under test rather than the clock.
	backdateLastSent(t, email, time.Hour)

	if owed := subscribe(t, st, email, "second-try"); owed {
		t.Error("an address that asked to be removed was owed another confirmation mail")
	}
	if err := st.ConfirmSubscriber(ctx, fingerprint("second-try")); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("the row was reset onto a new token: %v", err)
	}
	if got := mailable(t, st); len(got) != 0 {
		t.Errorf("mailable = %v, want nobody", got)
	}
}

func TestAPendingAddressIsMailedOncePerWindow(t *testing.T) {
	// Every POST used to rotate the token and report a mail owed, so one form,
	// one stranger's address and a loop was a mail bomb with this domain's name
	// on the envelope. The rate limit does not help: it is per path and IP, and
	// there are more IPs.
	st := storetest.New(t)
	ctx := context.Background()
	const email = "victim@khodrobin.test"

	if owed := subscribe(t, st, email, "the-one-mail"); !owed {
		t.Fatal("the first signup was owed no confirmation mail")
	}
	for i := 2; i <= 6; i++ {
		if owed := subscribe(t, st, email, "attempt-"+strconv.Itoa(i)); owed {
			t.Fatalf("signup %d mailed the address again", i)
		}
	}
	// The token from the message that did go out has to survive the attempts,
	// or the amplifier is replaced by a link that stops working.
	if err := st.ConfirmSubscriber(ctx, fingerprint("the-one-mail")); err != nil {
		t.Errorf("the mailed link was rotated away under it: %v", err)
	}
}

func TestTheResendGateOpensAgainAfterTheWindow(t *testing.T) {
	// Gated, not sealed. Somebody who really did sign up and lost the message
	// has to be able to ask for it a second time.
	st := storetest.New(t)
	const email = "lost-the-mail@khodrobin.test"

	subscribe(t, st, email, "first")
	if owed := subscribe(t, st, email, "too-soon"); owed {
		t.Fatal("a second signup inside the window mailed again")
	}
	backdateLastSent(t, email, time.Hour)
	if owed := subscribe(t, st, email, "an-hour-later"); !owed {
		t.Error("no mail was owed an hour after the last one")
	}
}

// backdateLastSent moves the resend clock into the past. Waiting out a fifteen
// minute window in a test is not an option, and the column has no reader on
// Store worth adding for this alone.
func backdateLastSent(t *testing.T, email string, by time.Duration) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, storetest.DSN())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	tag, err := conn.Exec(ctx,
		`UPDATE subscribers SET last_sent_at = last_sent_at - $2::interval WHERE email = $1`,
		email, fmt.Sprintf("%d seconds", int(by.Seconds())))
	if err != nil {
		t.Fatal(err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatalf("backdated %d rows for %s, want 1", tag.RowsAffected(), email)
	}
}

func TestARegisterOptInIsOnTheListAndCanLeaveIt(t *testing.T) {
	// The tick box wrote one boolean on users that no export read and no
	// unsubscribe path could clear. A confirmed row is what turns it into
	// consent with a token behind it.
	st := storetest.New(t)
	ctx := context.Background()
	const email = "ticked-the-box@khodrobin.test"

	if err := st.AddConfirmedSubscriber(ctx, email, fingerprint("register-token"), "register"); err != nil {
		t.Fatal(err)
	}
	if got := mailable(t, st); len(got) != 1 || got[0] != email {
		t.Fatalf("mailable = %v, want the account that ticked the box", got)
	}
	// Nothing is owed, so registration cannot turn into a second message on top
	// of the verification mail already going to this address.
	if owed := subscribe(t, st, email, "would-be-a-second-mail"); owed {
		t.Error("the register opt-in was owed a confirmation mail as well")
	}
	if err := st.Unsubscribe(ctx, fingerprint("register-token")); err != nil {
		t.Errorf("the one-click exit did not work for a register opt-in: %v", err)
	}
	if got := mailable(t, st); len(got) != 0 {
		t.Errorf("mailable = %v after unsubscribing", got)
	}
}

func TestRegisteringDoesNotRevivePeopleAndDoesNotRotateTheirToken(t *testing.T) {
	st := storetest.New(t)
	ctx := context.Background()

	// Someone who left, whose address is then used to register an account: the
	// registration is not their consent to come back.
	const left = "left@khodrobin.test"
	subscribe(t, st, left, "gone")
	if err := st.ConfirmSubscriber(ctx, fingerprint("gone")); err != nil {
		t.Fatal(err)
	}
	if err := st.Unsubscribe(ctx, fingerprint("gone")); err != nil {
		t.Fatal(err)
	}
	if err := st.AddConfirmedSubscriber(ctx, left, fingerprint("fresh-token"), "register"); err != nil {
		t.Fatal(err)
	}

	// Someone mid double opt-in, who then registers with the box ticked: the
	// row is upgraded in place, and the link already sitting in their inbox —
	// which is also their unsubscribe link — has to keep working.
	const pending = "halfway@khodrobin.test"
	subscribe(t, st, pending, "mailed-link")
	if err := st.AddConfirmedSubscriber(ctx, pending, fingerprint("a-second-token"), "register"); err != nil {
		t.Fatal(err)
	}

	if got := mailable(t, st); len(got) != 1 || got[0] != pending {
		t.Fatalf("mailable = %v, want only the address that was still pending", got)
	}
	if err := st.Unsubscribe(ctx, fingerprint("mailed-link")); err != nil {
		t.Errorf("the token from the confirmation mail stopped working: %v", err)
	}
	if err := st.Unsubscribe(ctx, fingerprint("fresh-token")); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("registration issued a token against somebody else's row: %v", err)
	}
}

func TestTheDashboardCountIsTheExportCounted(t *testing.T) {
	// Two numbers both described as "the mailable list" is one number too many.
	// They are the same predicate now, so a fixture that pulls every lever has
	// to leave them agreeing.
	st := storetest.New(t)
	ctx := context.Background()

	subscribe(t, st, "confirmed@khodrobin.test", "s1")
	if err := st.ConfirmSubscriber(ctx, fingerprint("s1")); err != nil {
		t.Fatal(err)
	}
	subscribe(t, st, "pending@khodrobin.test", "s2")
	subscribe(t, st, "left@khodrobin.test", "s3")
	if err := st.ConfirmSubscriber(ctx, fingerprint("s3")); err != nil {
		t.Fatal(err)
	}
	if err := st.Unsubscribe(ctx, fingerprint("s3")); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateUser(ctx, "registered@khodrobin.test", "h", nil, true); err != nil {
		t.Fatal(err)
	}
	if err := st.AddConfirmedSubscriber(ctx, "registered@khodrobin.test", fingerprint("s4"), "register"); err != nil {
		t.Fatal(err)
	}

	counts, err := st.Counts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := mailable(t, st)
	if int(counts.SubscribersConfirmed) != len(got) {
		t.Errorf("dashboard says %d mailable, the export has %d: %v",
			counts.SubscribersConfirmed, len(got), got)
	}
	if len(got) != 2 {
		t.Errorf("export = %v, want the confirmed subscriber and the register opt-in", got)
	}
	// The opt-in count is a breakdown of that number, not something to add to
	// it: every consenting account is already a row on the list.
	if counts.MarketingOptin != 1 {
		t.Errorf("marketing_optin = %d, want 1", counts.MarketingOptin)
	}
}

func TestTheRepairMigrationFixesLiveRowsAndCanRunTwice(t *testing.T) {
	// 003 is not only a column: it carries the two repairs that the code fixes
	// going forward but cannot fix backwards — consent that never reached the
	// list, and labels that got in before the door existed. It runs again on
	// every boot, so running it twice has to be the same as running it once.
	st := storetest.New(t)
	ctx := context.Background()

	if _, err := st.CreateUser(ctx, "old-optin@khodrobin.test", "h", nil, true); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateUser(ctx, "old-optout@khodrobin.test", "h", nil, false); err != nil {
		t.Fatal(err)
	}
	// Someone who consented at registration and has since left. Nothing is
	// allowed to put them back on the list, a boot least of all.
	if _, err := st.CreateUser(ctx, "gone@khodrobin.test", "h", nil, true); err != nil {
		t.Fatal(err)
	}
	subscribe(t, st, "gone@khodrobin.test", "their-token")
	if err := st.ConfirmSubscriber(ctx, fingerprint("their-token")); err != nil {
		t.Fatal(err)
	}
	if err := st.Unsubscribe(ctx, fingerprint("their-token")); err != nil {
		t.Fatal(err)
	}
	writeRawSource(t, "gone@khodrobin.test", "footer\nphantom@khodrobin.test")

	for i := 1; i <= 2; i++ {
		if err := st.Migrate(ctx, store.Schema); err != nil {
			t.Fatalf("apply %d: %v", i, err)
		}
		got := mailable(t, st)
		if len(got) != 1 || got[0] != "old-optin@khodrobin.test" {
			t.Fatalf("after apply %d mailable = %v, want the backfilled opt-in alone", i, got)
		}
		all, total, err := st.ListSubscribers(ctx, 50, 0)
		if err != nil {
			t.Fatal(err)
		}
		if total != 2 {
			t.Errorf("after apply %d there are %d subscriber rows, want 2", i, total)
		}
		for _, sub := range all {
			if strings.ContainsAny(sub.Source, "\r\n") {
				t.Errorf("after apply %d, source %q still carries a line break", i, sub.Source)
			}
		}
	}
}

// writeRawSource plants the kind of label that got into the table before the
// handler started rejecting it. Store has no method for writing a broken value,
// which is the point of it.
func writeRawSource(t *testing.T, email, source string) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, storetest.DSN())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx,
		`UPDATE subscribers SET source = $2 WHERE email = $1`, email, source); err != nil {
		t.Fatal(err)
	}
}
