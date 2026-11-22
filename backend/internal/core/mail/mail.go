// Package mail sends transactional email (password recovery). Without SMTP
// config it falls back to a log-only mailer so development never blocks.
package mail

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"

	"mauzo/internal/config"
)

// Mailer sends plain-text transactional email.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// New returns an SMTP mailer when a host is configured, otherwise a mailer
// that logs messages instead of sending them.
func New(cfg config.MailConfig, log *slog.Logger) Mailer {
	if cfg.Host == "" {
		return logMailer{log: log, from: cfg.From}
	}
	return smtpMailer{cfg: cfg}
}

type logMailer struct {
	log  *slog.Logger
	from string
}

func (m logMailer) Send(_ context.Context, to, subject, body string) error {
	m.log.Info("mail (log-only, configure SMTP_HOST)", "to", to, "subject", subject, "body", body)
	return nil
}

type smtpMailer struct {
	cfg config.MailConfig
}

func (m smtpMailer) Send(_ context.Context, to, subject, body string) error {
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", m.cfg.From, to, subject, body)
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	var auth smtp.Auth
	if m.cfg.User != "" {
		auth = smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)
	}
	if err := smtp.SendMail(addr, auth, m.cfg.From, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("mail: smtp send: %w", err)
	}
	return nil
}
