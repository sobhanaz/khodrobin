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
// Send delivers one message over an always-encrypted connection.
//
// TWO WAYS TO GET THERE, and the port decides which. Port 465 is implicit TLS
// (SMTPS): the socket is wrapped before a single SMTP verb is spoken. Everything
// else negotiates STARTTLS after connecting. Cloudflare's submission endpoint
// offers only the former and explicitly refuses STARTTLS on 587; Gmail offers
// only the latter. Supporting both is what lets the provider change by editing
// two variables instead of this file.
//
// STARTTLS IS NOW REQUIRED RATHER THAN PREFERRED. It used to run only when the
// server advertised the extension, so a server that stayed quiet about it got
// the password and the verification links in clear text. The comment here
// claimed that could not happen while the code allowed it, which is the same
// shape as every other bug this project has had: a promise nothing enforced.
func (m *Mailer) Send(to, subject, body string) error {
	if !m.Configured() {
		return fmt.Errorf("smtp is not configured")
	}

	addr := net.JoinHostPort(m.host, m.port)
	tlsConf := &tls.Config{ServerName: m.host, MinVersion: tls.VersionTLS12}
	dialer := &net.Dialer{Timeout: 15 * time.Second}

	var conn net.Conn
	var err error
	implicit := m.port == "465"
	if implicit {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsConf)
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Quit()

	if !implicit {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("%s does not offer STARTTLS; refusing to send credentials in clear text", addr)
		}
		if err := client.StartTLS(tlsConf); err != nil {
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
