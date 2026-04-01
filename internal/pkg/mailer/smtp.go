package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"go-permission-system/internal/config"
)

type Sender interface {
	Enabled() bool
	Send(ctx context.Context, toEmail, subject, textBody string) error
}

type SMTPSender struct {
	cfg config.MailConfig
}

func NewSMTPSender(cfg config.MailConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) Enabled() bool {
	return strings.TrimSpace(s.cfg.Host) != "" &&
		s.cfg.Port > 0 &&
		strings.TrimSpace(s.cfg.FromEmail) != ""
}

func (s *SMTPSender) Send(ctx context.Context, toEmail, subject, textBody string) error {
	if !s.Enabled() {
		return errors.New("mail channel is not configured")
	}

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	conn, err := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}

	if err = conn.SetDeadline(time.Now().Add(20 * time.Second)); err != nil {
		_ = conn.Close()
		return err
	}

	if s.cfg.UseTLS || s.cfg.Port == 465 {
		tlsConn := tls.Client(conn, &tls.Config{ServerName: s.cfg.Host})
		if err = tlsConn.Handshake(); err != nil {
			_ = tlsConn.Close()
			return err
		}
		conn = tlsConn
	}

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer client.Close()

	if !s.cfg.UseTLS && s.cfg.Port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err = client.StartTLS(&tls.Config{ServerName: s.cfg.Host}); err != nil {
				return err
			}
		}
	}

	if strings.TrimSpace(s.cfg.Username) != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
		if err = client.Auth(auth); err != nil {
			return err
		}
	}

	if err = client.Mail(strings.TrimSpace(s.cfg.FromEmail)); err != nil {
		return err
	}
	if err = client.Rcpt(strings.TrimSpace(toEmail)); err != nil {
		return err
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}

	message := buildMessage(s.cfg, toEmail, subject, textBody)
	if _, err = writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}

	return client.Quit()
}

func buildMessage(cfg config.MailConfig, toEmail, subject, textBody string) string {
	from := strings.TrimSpace(cfg.FromEmail)
	if strings.TrimSpace(cfg.FromName) != "" {
		from = fmt.Sprintf("%s <%s>", strings.TrimSpace(cfg.FromName), strings.TrimSpace(cfg.FromEmail))
	}

	return strings.Join([]string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", strings.TrimSpace(toEmail)),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		textBody,
	}, "\r\n")
}
