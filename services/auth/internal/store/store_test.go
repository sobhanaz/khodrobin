package store_test

import (
	"context"
	"errors"
	"os"
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

func TestBothMigrationsAreEmbeddedInOrder(t *testing.T) {
	// The embed used to name 001_init.sql directly. The day a second file was
	// added it would have sat in the repository looking applied and never run,
	// so this pins the glob rather than the filename.
	init001 := strings.Index(store.Schema, "CREATE TABLE IF NOT EXISTS users")
	marketing002 := strings.Index(store.Schema, "CREATE TABLE IF NOT EXISTS subscribers")
	if init001 < 0 {
		t.Fatal("001_init.sql is missing from the embedded schema")
	}
	if marketing002 < 0 {
		t.Fatal("002_marketing.sql is missing from the embedded schema")
	}
	if marketing002 < init001 {
		// 002 alters users, so a swapped order is a boot that fails on a fresh
		// database and nowhere else.
		t.Error("002 is concatenated before 001")
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
