package services

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"github.com/novapanel/novapanel/internal/config"
)

type SMTPService struct {
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
	return s.host != "" && s.user != ""
}

func (s *SMTPService) send(to, subject, body string) error {
	if !s.Enabled() {
		return nil // silently skip when SMTP is not configured
	}

	addr := net.JoinHostPort(s.host, s.port)
	msg := strings.Join([]string{
		"From: " + s.from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	auth := smtp.PlainAuth("", s.user, s.password, s.host)

	// Try STARTTLS on port 587, plain TLS on port 465, plain on others
	if s.port == "465" {
		tlsCfg := &tls.Config{ServerName: s.host}
		conn, err := tls.Dial("tcp", addr, tlsCfg)
		if err != nil {
			return fmt.Errorf("smtp tls dial: %w", err)
		}
		client, err := smtp.NewClient(conn, s.host)
		if err != nil {
			return fmt.Errorf("smtp new client: %w", err)
		}
		defer client.Close()
		if err = client.Auth(auth); err != nil {
			return err
		}
		if err = client.Mail(s.from); err != nil {
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

	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
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
