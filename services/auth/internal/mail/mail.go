// Package mail sends the few transactional messages this product needs.
//
// net/smtp from the standard library rather than a provider SDK: three message
// types do not justify a dependency, and SMTP means the sending account can be
// changed without touching code.
package mail

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"
)

type Mailer struct {
	host, port, user, pass, from, fromName string
	log                                    *slog.Logger
}

func FromEnv(log *slog.Logger) *Mailer {
	return &Mailer{
		host:     env("SMTP_HOST", "smtp.gmail.com"),
		port:     env("SMTP_PORT", "587"),
		user:     env("SMTP_USER", ""),
		pass:     os.Getenv("SMTP_PASSWORD"),
		from:     env("SMTP_FROM", env("SMTP_USER", "")),
		fromName: env("SMTP_FROM_NAME", "خودروبین"),
		log:      log,
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Configured reports whether mail can actually be sent.
//
// Checked rather than assumed: the service must start and serve search without
// SMTP credentials, so registration degrades to "we could not send the link"
// instead of the whole container failing to boot.
func (m *Mailer) Configured() bool {
	return m.host != "" && m.user != "" && m.pass != "" && m.from != ""
}

// Send delivers one message over STARTTLS.
//
// Explicit STARTTLS with a real ServerName rather than smtp.SendMail's default:
// this carries verification links, and an unauthenticated downgrade would put
// them on the wire in clear text.
func (m *Mailer) Send(to, subject, body string) error {
	if !m.Configured() {
		return fmt.Errorf("smtp is not configured")
	}

	addr := net.JoinHostPort(m.host, m.port)
	conn, err := net.DialTimeout("tcp", addr, 15*time.Second)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Quit()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: m.host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}
	if err := client.Auth(smtp.PlainAuth("", m.user, m.pass, m.host)); err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	if err := client.Mail(m.from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	// Subject is RFC 2047 encoded because it is Persian; unencoded UTF-8 in a
	// header arrives as mojibake in most clients.
	msg := strings.Join([]string{
		fmt.Sprintf("From: %s <%s>", mime.QEncoding.Encode("utf-8", m.fromName), m.from),
		"To: " + to,
		"Subject: " + mime.QEncoding.Encode("utf-8", subject),
		"MIME-Version: 1.0",
		`Content-Type: text/html; charset="utf-8"`,
		"", body,
	}, "\r\n")
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close body: %w", err)
	}
	m.log.Info("mail.sent", "to", redact(to), "subject", subject)
	return nil
}

// redact keeps enough of an address to correlate logs without printing it.
func redact(email string) string {
	at := strings.LastIndex(email, "@")
	if at <= 1 {
		return "***"
	}
	return email[:1] + "***" + email[at:]
}
