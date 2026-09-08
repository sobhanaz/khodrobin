package mail

import (
	"io"
	"log/slog"
	"strings"
	"testing"
)

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestUnconfiguredMailerFailsCleanly(t *testing.T) {
	// The service must start and serve search without SMTP credentials, so this
	// returns an error rather than panicking or blocking.
	m := &Mailer{log: quiet()}
	if m.Configured() {
		t.Fatal("empty mailer reported itself configured")
	}
	if err := m.Send("a@b.co", "s", "b"); err == nil {
		t.Error("Send succeeded with no configuration")
	}
}

func TestAddressesAreRedactedInLogs(t *testing.T) {
	for in, want := range map[string]string{
		"sobhan@gmail.com": "s***@gmail.com",
		"a@b.co":           "***",
		"":                 "***",
	} {
		if got := redact(in); got != want {
			t.Errorf("redact(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLinksAreEscapedIntoTemplates(t *testing.T) {
	// A link is attacker-influenced if a redirect parameter ever reaches it.
	_, body := Verify(`https://x.test/v?t=a"><script>alert(1)</script>`, "123456")
	if strings.Contains(body, "<script>") {
		t.Error("unescaped markup reached the email body")
	}
}

func TestVerifyCarriesTheCode(t *testing.T) {
	// The code is the fallback path when a link tap fails; a mail that loses it
	// has silently cut the recovery route in half.
	_, body := Verify("https://x.test/verify?t=abc", "483920")
	for _, want := range []string{"483920", "۱۵ دقیقه", "https://x.test/verify", "صفحه‌ی تأیید"} {
		if !strings.Contains(body, want) {
			t.Errorf("verify body is missing %q", want)
		}
	}
	if strings.Contains(body, "/verify?t=abc\">") {
		t.Error("token link reached the code panel href unescaped")
	}
}

func TestWelcomeNamesTheNewCapabilities(t *testing.T) {
	subject, body := Welcome("https://khodrobin.noxioai.com/")
	if !strings.Contains(subject, "خوش آمدی") {
		t.Errorf("subject does not welcome: %s", subject)
	}
	for _, want := range []string{"جست‌وجوهایت را ذخیره کنی", "هشدار بگیری"} {
		if !strings.Contains(body, want) {
			t.Errorf("welcome body is missing %q", want)
		}
	}
}

func TestPasswordChangedPointsAtTheRecoveryMove(t *testing.T) {
	subject, body := PasswordChanged("https://x.test/login", "https://x.test/forgot")
	if !strings.Contains(subject, "رمز عبور") {
		t.Errorf("subject does not name the password: %s", subject)
	}
	for _, want := range []string{"/login", "/forgot", "اگر این کار را تو نکرده‌ای"} {
		if !strings.Contains(body, want) {
			t.Errorf("password-changed body is missing %q", want)
		}
	}
}

func TestSavedSearchCreatedStatesThresholdAndQuery(t *testing.T) {
	subject, body := SavedSearchCreated("ارزون‌ترین پژو ۲۰۶", 7, "https://x.test/?q=%D9%BE%D8%B2%D9%88")
	if !strings.Contains(subject, "هشدار قیمت") {
		t.Errorf("subject does not name the alert: %s", subject)
	}
	for _, want := range []string{"ارزون‌ترین پژو ۲۰۶", "7%", "/?q=", "لغو هشدار"} {
		if !strings.Contains(body, want) {
			t.Errorf("saved-search body is missing %q", want)
		}
	}
}

func TestPriceAlertStatesDirectionAndAmount(t *testing.T) {
	// An alert that does not say why it fired trains people to ignore alerts.
	subject, body := PriceAlert("پژو ۲۰۶", "ارزون‌ترین پژو ۲۰۶", 1_000_000_000, 900_000_000,
		"https://khodrobin.noxioai.com/")
	if !strings.Contains(subject, "کاهش") {
		t.Errorf("subject does not state the direction: %s", subject)
	}
	for _, want := range []string{"1,000,000,000", "900,000,000", "-10.0%"} {
		if !strings.Contains(body, want) {
			t.Errorf("body is missing %q", want)
		}
	}
}

func TestConfirmationMailCarriesBothLinks(t *testing.T) {
	// One is the consent, the other is the way out for somebody whose address
	// was typed in by a stranger. A template that dropped either would turn the
	// only mail an unconfirmed address ever gets into a dead end.
	subject, body := ConfirmSubscription(
		"https://khodrobin.test/subscribed?token=abc",
		`https://khodrobin.test/unsubscribe?token=x"><script>alert(1)</script>`)
	if !strings.Contains(subject, "خبرنامه") {
		t.Errorf("subject does not name the newsletter: %s", subject)
	}
	for _, want := range []string{"/subscribed?token=abc", "/unsubscribe?token=x"} {
		if !strings.Contains(body, want) {
			t.Errorf("body is missing %q", want)
		}
	}
	if strings.Contains(body, "<script>") {
		t.Error("unescaped markup reached the email body")
	}
}

func TestEveryMailCarriesPreheaderFallbackAndColorScheme(t *testing.T) {
	// Three email-craft rules, applied to every message:
	//  - a preheader (the sentence shown in the inbox list; without one the
	//    client pulls a random body line instead),
	//  - a plain-text fallback under the CTA (some clients render styled links
	//    as plain text, and a verification link the reader cannot reach is a
	//    lost account),
	//  - an explicit dark color-scheme, so Gmail/Apple don't invert the dark
	//    card into a light one.
	cases := []struct {
		name string
		body string
	}{
		{"verify", mustBody(Verify("https://x.test/v?t=abc", "123456"))},
		{"reset", mustBody(Reset("https://x.test/r?t=abc"))},
		{"subscribe", mustBody(ConfirmSubscription("https://x.test/s?t=abc", "https://x.test/u?t=abc"))},
		{"price", mustBody(PriceAlert("پژو ۲۰۶", "ارزون‌ترین پژو ۲۰۶", 1_000_000_000, 900_000_000, "https://x.test/"))},
		{"welcome", mustBody(Welcome("https://x.test/"))},
		{"password", mustBody(PasswordChanged("https://x.test/login", "https://x.test/forgot"))},
		{"saved", mustBody(SavedSearchCreated("پژو ۲۰۶", 10, "https://x.test/?q=x"))},
	}
	for _, tc := range cases {
		for _, want := range []string{"display:none;max-height:0", "یا این لینک را در مرورگر باز کن", "color-scheme:dark"} {
			if !strings.Contains(tc.body, want) {
				t.Errorf("%s: body is missing %q", tc.name, want)
			}
		}
	}
}

func mustBody(_ string, body string) string { return body }

func TestCommaGrouping(t *testing.T) {
	for in, want := range map[int64]string{0: "0", 999: "999", 1000: "1,000", 1_660_000_000: "1,660,000,000"} {
		if got := comma(in); got != want {
			t.Errorf("comma(%d) = %q, want %q", in, got, want)
		}
	}
}

var _ = io.Discard
