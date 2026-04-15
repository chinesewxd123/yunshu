package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"net"
	"net/smtp"
	"strings"
	"time"

	"go-permission-system/internal/config"
)

type Sender interface {
	Enabled() bool
	Send(ctx context.Context, toEmail, subject, textBody string) error
	// SendMultipart 发送 multipart/alternative邮件：textPlain 为纯文本，htmlBody 非空时同时附带 HTML（客户端优先展示 HTML）。
	SendMultipart(ctx context.Context, toEmail, subject, textPlain, htmlBody string) error
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
	return s.SendMultipart(ctx, toEmail, subject, textBody, "")
}

func (s *SMTPSender) SendMultipart(ctx context.Context, toEmail, subject, textPlain, htmlBody string) error {
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

	message := buildMessage(s.cfg, toEmail, subject, textPlain, htmlBody)
	if _, err = writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}

	return client.Quit()
}

func buildMessage(cfg config.MailConfig, toEmail, subject, textPlain, htmlBody string) string {
	from := strings.TrimSpace(cfg.FromEmail)
	if strings.TrimSpace(cfg.FromName) != "" {
		from = fmt.Sprintf("%s <%s>", strings.TrimSpace(cfg.FromName), strings.TrimSpace(cfg.FromEmail))
	}
	to := strings.TrimSpace(toEmail)
	subjEnc := mime.QEncoding.Encode("UTF-8", subject)

	if strings.TrimSpace(htmlBody) == "" {
		return strings.Join([]string{
			fmt.Sprintf("From: %s", from),
			fmt.Sprintf("To: %s", to),
			fmt.Sprintf("Subject: %s", subjEnc),
			"MIME-Version: 1.0",
			"Content-Type: text/plain; charset=UTF-8",
			"Content-Transfer-Encoding: 8bit",
			"",
			textPlain,
		}, "\r\n")
	}

	var altBody strings.Builder
	mw := multipart.NewWriter(&altBody)
	boundary := mw.Boundary()
	p1, _ := mw.CreatePart(map[string][]string{
		"Content-Type":              {"text/plain; charset=UTF-8"},
		"Content-Transfer-Encoding": {"8bit"},
	})
	_, _ = p1.Write([]byte(textPlain))
	p2, _ := mw.CreatePart(map[string][]string{
		"Content-Type":              {"text/html; charset=UTF-8"},
		"Content-Transfer-Encoding": {"8bit"},
	})
	_, _ = p2.Write([]byte(htmlBody))
	_ = mw.Close()

	return strings.Join([]string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subjEnc),
		"MIME-Version: 1.0",
		fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s", boundary),
		"",
		strings.TrimSuffix(altBody.String(), "\r\n"),
	}, "\r\n")
}
