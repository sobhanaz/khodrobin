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
	_, body := Verify(`https://x.test/v?t=a"><script>alert(1)</script>`)
	if strings.Contains(body, "<script>") {
		t.Error("unescaped markup reached the email body")
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

func TestCommaGrouping(t *testing.T) {
	for in, want := range map[int64]string{0: "0", 999: "999", 1000: "1,000", 1_660_000_000: "1,660,000,000"} {
		if got := comma(in); got != want {
			t.Errorf("comma(%d) = %q, want %q", in, got, want)
		}
	}
}

var _ = io.Discard
