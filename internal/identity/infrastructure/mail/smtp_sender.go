package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strconv"
	"time"
)

const (
	smtpTimeout = 10 * time.Second
	smtpsPort   = 465
)

// Config holds the SMTP configuration for sending emails.
type Config struct {
	Server        string // e.g. "smtp.yandex.ru:465"
	Username      string
	Password      string
	From          string // e.g. "noreply@example.com" — used as Reply-To
	VerifyBaseURL string // public URL of the verify endpoint, e.g. "https://identity.example.com"
}

// SmtpMailSender implements ports.MailSender via SMTP.
type SmtpMailSender struct {
	cfg Config
}

// NewSmtpMailSender creates a new SmtpMailSender.
func NewSmtpMailSender(cfg Config) *SmtpMailSender {
	return &SmtpMailSender{cfg: cfg}
}

// SendVerificationEmail sends a verification email with a link containing the token.
// If SMTP is not configured, it logs a warning and returns nil (no error).
func (s *SmtpMailSender) SendVerificationEmail(ctx context.Context, email, token string) error {
	// Graceful degradation: if SMTP is not configured, skip sending.
	if s.cfg.Server == "" || s.cfg.From == "" {
		log.Printf("mail: SMTP not configured, skipping verification email to %s", email)
		return nil
	}

	// Build the verification link pointing to the identity service GET /verify endpoint.
	base := s.cfg.VerifyBaseURL
	if base == "" {
		base = "http://localhost:8080"
	}
	link := fmt.Sprintf("%s/verify?token=%s&email=%s", base, token, email)

	// Determine the header From address.
	// Yandex (and many other providers) require the header From to match
	// the authenticated user. So we use the SMTP username when available.
	headerFrom := s.cfg.Username
	if headerFrom == "" {
		headerFrom = s.cfg.From
	}

	subject := "Subject: Подтверждение регистрации\r\n"
	mime := "MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"
	body := fmt.Sprintf(`<h2>Подтверждение регистрации</h2>
<p>Для завершения регистрации перейдите по ссылке:</p>
<p><a href="%s">%s</a></p>
<p>Ссылка действительна 24 часа.</p>`, link, link)

	msg := []byte("To: " + email + "\r\n" +
		"From: " + headerFrom + "\r\n" +
		"Reply-To: " + s.cfg.From + "\r\n" +
		subject + mime +
		"\r\n" + body)

	// Use the request context or a default timeout to avoid hanging.
	dialCtx := ctx
	if _, hasDeadline := dialCtx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		dialCtx, cancel = context.WithTimeout(ctx, smtpTimeout)
		defer cancel()
	}

	return sendMailWithContext(dialCtx, s.cfg.Server, s.cfg.Username, s.cfg.Password, s.cfg.From, []string{email}, msg)
}

// sendMailWithContext sends an email via SMTP, supporting both SMTPS (port 465)
// and standard SMTP with STARTTLS (ports 587, 25).
//
// The `from` parameter is used for the Message-ID and logging, but the SMTP
// envelope MAIL FROM is set to `username` when credentials are provided —
// this is required by servers like Yandex and Gmail that check the envelope
// sender matches the authenticated user.
func sendMailWithContext(ctx context.Context, addr, username, password, from string, to []string, msg []byte) error {
	var conn net.Conn

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("mail: invalid addr %q: %w", addr, err)
	}

	if isSMTPS(addr) {
		// Port 465 — SMTPS (implicit TLS). Dial and wrap with TLS immediately.
		var dialer net.Dialer
		conn, err = dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("mail: dial %s: %w", addr, err)
		}
		tlsConn := tls.Client(conn, &tls.Config{
			ServerName: host,
		})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			conn.Close()
			return fmt.Errorf("mail: tls handshake: %w", err)
		}
		conn = tlsConn
	} else {
		// Other ports (587, 25) — plain TCP, then STARTTLS.
		var dialer net.Dialer
		conn, err = dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("mail: dial %s: %w", addr, err)
		}
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("mail: smtp client: %w", err)
	}
	defer client.Close()

	// Try STARTTLS for non-SMTPS connections.
	if !isSMTPS(addr) {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
				return fmt.Errorf("mail: starttls: %w", err)
			}
		}
	}

	// Authenticate if credentials are provided.
	if username != "" || password != "" {
		auth := smtp.PlainAuth("", username, password, host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("mail: auth: %w", err)
		}
	}

	// Use the authenticated username as the SMTP envelope sender (MAIL FROM).
	// Many servers (Yandex, Gmail) require it to match the auth user.
	// The header From in the message body can still be a different noreply address.
	envelopeFrom := from
	if username != "" {
		envelopeFrom = username
	}
	if err := client.Mail(envelopeFrom); err != nil {
		return fmt.Errorf("mail: mail from: %w", err)
	}

	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("mail: rcpt %s: %w", rcpt, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mail: data: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("mail: write body: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("mail: close body: %w", err)
	}

	return client.Quit()
}

// isSMTPS returns true if the port is 465 (SMTPS, implicit TLS).
func isSMTPS(addr string) bool {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		return false
	}
	return p == smtpsPort
}
