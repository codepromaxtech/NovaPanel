package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	novacrypto "github.com/novapanel/novapanel/internal/crypto"
	"github.com/novapanel/novapanel/internal/config"
)

type SMTPService struct {
	mu       sync.RWMutex
	host     string
	port     string
	user     string
	password string
	from     string
}

func NewSMTPService(cfg *config.Config) *SMTPService {
	return &SMTPService{
		host:     cfg.SMTPHost,
		port:     cfg.SMTPPort,
		user:     cfg.SMTPUser,
		password: cfg.SMTPPassword,
		from:     cfg.SMTPFrom,
	}
}

func (s *SMTPService) Enabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.host != "" && s.user != ""
}

// ReloadFromDB reads SMTP settings from system_settings and updates the live service.
func (s *SMTPService) ReloadFromDB(ctx context.Context, pool *pgxpool.Pool, cryptoKey string) {
	rows, err := pool.Query(ctx, `SELECT key, value, encrypted FROM system_settings WHERE key LIKE 'smtp_%'`)
	if err != nil {
		return
	}
	defer rows.Close()

	settings := map[string]string{}
	for rows.Next() {
		var key, value string
		var encrypted bool
		if err := rows.Scan(&key, &value, &encrypted); err != nil {
			continue
		}
		if encrypted && value != "" {
			if plain, err := novacrypto.Decrypt(value, []byte(cryptoKey)); err == nil {
				settings[key] = plain
			}
		} else {
			settings[key] = value
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := settings["smtp_host"]; ok && v != "" { s.host = v }
	if v, ok := settings["smtp_port"]; ok && v != "" { s.port = v }
	if v, ok := settings["smtp_user"]; ok && v != "" { s.user = v }
	if v, ok := settings["smtp_password"]; ok { s.password = v }
	if v, ok := settings["smtp_from"]; ok && v != "" { s.from = v }
}

// SendTestEmail sends a test message to confirm SMTP is working.
func (s *SMTPService) SendTestEmail(to string) error {
	body := `<p>This is a test email from <strong>NovaPanel</strong>.</p><p>Your SMTP configuration is working correctly.</p>`
	return s.send(to, "NovaPanel — SMTP Test", body)
}

func (s *SMTPService) send(to, subject, body string) error {
	if !s.Enabled() {
		return nil
	}

	s.mu.RLock()
	host, port, user, password, from := s.host, s.port, s.user, s.password, s.from
	s.mu.RUnlock()

	addr := net.JoinHostPort(host, port)
	msg := strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	auth := smtp.PlainAuth("", user, password, host)

	if port == "465" {
		tlsCfg := &tls.Config{ServerName: host}
		conn, err := tls.Dial("tcp", addr, tlsCfg)
		if err != nil {
			return fmt.Errorf("smtp tls dial: %w", err)
		}
		client, err := smtp.NewClient(conn, host)
		if err != nil {
			return fmt.Errorf("smtp new client: %w", err)
		}
		defer client.Close()
		if err = client.Auth(auth); err != nil {
			return err
		}
		if err = client.Mail(from); err != nil {
			return err
		}
		if err = client.Rcpt(to); err != nil {
			return err
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(w, msg)
		if err != nil {
			return err
		}
		return w.Close()
	}

	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
}

func (s *SMTPService) SendPasswordReset(to, resetURL string) error {
	body := fmt.Sprintf(`
<p>You requested a password reset for your NovaPanel account.</p>
<p><a href="%s">Click here to reset your password</a></p>
<p>This link expires in 15 minutes. If you did not request this, ignore this email.</p>
`, resetURL)
	return s.send(to, "NovaPanel — Reset your password", body)
}

func (s *SMTPService) SendAlertNotification(to, subject, body string) error {
	htmlBody := fmt.Sprintf(`<pre>%s</pre>`, body)
	return s.send(to, subject, htmlBody)
}

func (s *SMTPService) SendWelcome(to, firstName string) error {
	body := fmt.Sprintf(`<p>Welcome to NovaPanel, %s! Your account has been created.</p>`, firstName)
	return s.send(to, "Welcome to NovaPanel", body)
}
