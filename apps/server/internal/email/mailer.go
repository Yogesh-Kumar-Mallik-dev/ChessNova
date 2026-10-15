package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

type Mailer interface {
	SendVerificationOTP(ctx context.Context, toEmail, username, otp string) error
	IsConfigured() bool
}

type Config struct {
	ResendAPIKey string
	Host         string
	Port         string
	User         string
	Password     string
	From         string
}

type UniversalMailer struct {
	cfg        Config
	httpClient *http.Client
}

func NewMailer(cfg Config) Mailer {
	return &UniversalMailer{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (m *UniversalMailer) IsConfigured() bool {
	return m.cfg.ResendAPIKey != "" || (m.cfg.Host != "" && m.cfg.User != "" && m.cfg.Password != "") || (m.cfg.Host == "127.0.0.1" || m.cfg.Host == "localhost")
}

func (m *UniversalMailer) SendVerificationOTP(ctx context.Context, toEmail, username, otp string) error {
	toEmail = strings.TrimSpace(toEmail)
	if !m.IsConfigured() {
		log.Printf("[MAILER NOTICE] No email provider configured. Verification code for %s (%s): %s", username, toEmail, otp)
		return nil
	}

	from := m.cfg.From
	if from == "" {
		if m.cfg.ResendAPIKey != "" {
			from = "ChessNova <onboarding@resend.dev>"
		} else {
			from = "ChessNova <noreply@chess.yogeshmallik.in>"
		}
	}

	subject := fmt.Sprintf("%s is your ChessNova verification code", otp)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>ChessNova Email Verification</title>
</head>
<body style="margin: 0; padding: 0; background-color: #050811; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; color: #f1f5f9;">
  <table role="presentation" width="100%%" border="0" cellspacing="0" cellpadding="0" style="background-color: #050811; padding: 40px 10px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%%" style="max-width: 520px; background-color: #0b111e; border: 1px solid #162032; border-radius: 16px; overflow: hidden; box-shadow: 0 20px 40px rgba(0,0,0,0.6);" cellpadding="0" cellspacing="0">
          <tr>
            <td style="padding: 32px 32px 24px; text-align: center; border-bottom: 1px solid #162032;">
              <h1 style="margin: 0; font-size: 24px; font-weight: 800; color: #ffffff; letter-spacing: -0.5px;">
                Chess<span style="color: #38bdf8;">Nova</span>
              </h1>
              <p style="margin: 6px 0 0; font-size: 13px; color: #94a3b8;">
                Real-Time Competitive Chess Platform
              </p>
            </td>
          </tr>
          <tr>
            <td style="padding: 32px 32px 28px;">
              <h2 style="margin: 0 0 12px; font-size: 18px; font-weight: 700; color: #f8fafc;">
                Welcome, %s!
              </h2>
              <p style="margin: 0 0 24px; font-size: 14px; line-height: 1.5; color: #cbd5e1;">
                Thank you for creating an account on ChessNova. To verify your email address and activate your account, please enter the one-time code below:
              </p>
              
              <div style="background-color: #050811; border: 1px solid #38bdf840; border-radius: 12px; padding: 20px; text-align: center; margin-bottom: 24px;">
                <span style="font-size: 36px; font-weight: 900; letter-spacing: 10px; color: #38bdf8; font-family: monospace;">
                  %s
                </span>
                <p style="margin: 8px 0 0; font-size: 12px; color: #94a3b8;">
                  This code expires in <strong>10 minutes</strong>.
                </p>
              </div>

              <p style="margin: 0 0 8px; font-size: 13px; color: #64748b; line-height: 1.4;">
                If you did not attempt to register on ChessNova, you can safely ignore or delete this email.
              </p>
            </td>
          </tr>
          <tr>
            <td style="background-color: #080d17; padding: 20px 32px; text-align: center; border-top: 1px solid #162032;">
              <p style="margin: 0; font-size: 11px; color: #475569;">
                &copy; %d ChessNova. High-Performance Chess Engine & Platform.
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, username, otp, time.Now().Year())

	plainText := fmt.Sprintf("Welcome, %s!\n\nYour ChessNova verification code is: %s\n\nThis code expires in 10 minutes. If you did not request this, please ignore this email.\n\n- The ChessNova Team", username, otp)

	// 1. Resend HTTP API (Industry-standard proven solution, 100% deliverability over HTTPS)
	if m.cfg.ResendAPIKey != "" {
		return m.sendViaResend(ctx, from, toEmail, subject, htmlBody, plainText)
	}

	// 2. SMTP Transport (Gmail App Password, Brevo, SendGrid, or local relay)
	return m.sendViaSMTP(ctx, from, toEmail, subject, htmlBody, plainText)
}

func (m *UniversalMailer) sendViaResend(ctx context.Context, from, toEmail, subject, html, text string) error {
	payload := map[string]interface{}{
		"from":    from,
		"to":      []string{toEmail},
		"subject": subject,
		"html":    html,
		"text":    text,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal resend payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create resend request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+m.cfg.ResendAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		log.Printf("[RESEND ERROR] HTTP request failed: %v", err)
		return fmt.Errorf("failed to send via Resend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("[RESEND ERROR] Resend returned %d: %s", resp.StatusCode, string(respBody))
		return fmt.Errorf("resend email delivery failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("[RESEND SUCCESS] Dispatched verification code to %s", toEmail)
	return nil
}

func (m *UniversalMailer) sendViaSMTP(ctx context.Context, from, toEmail, subject, htmlBody, plainText string) error {
	boundary := "CHESSNOVA_MULTIPART_BOUNDARY"
	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: multipart/alternative; boundary=%s\r\n\r\n"+
		"--%s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n"+
		"%s\r\n\r\n"+
		"--%s\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n\r\n"+
		"%s\r\n\r\n"+
		"--%s--\r\n",
		from, toEmail, subject, boundary, boundary, plainText, boundary, htmlBody, boundary)

	port := m.cfg.Port
	if port == "" {
		port = "587"
	}
	addr := net.JoinHostPort(m.cfg.Host, port)

	// Direct SSL on port 465
	if port == "465" {
		tlsConfig := &tls.Config{
			ServerName: m.cfg.Host,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to dial SMTP over TLS: %w", err)
		}
		defer conn.Close()

		c, err := smtp.NewClient(conn, m.cfg.Host)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer c.Close()

		if m.cfg.User != "" && m.cfg.Password != "" {
			auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)
			if err := c.Auth(auth); err != nil {
				return fmt.Errorf("SMTP auth failed: %w", err)
			}
		}

		if err := c.Mail(cleanAddress(from)); err != nil {
			return err
		}
		if err := c.Rcpt(toEmail); err != nil {
			return err
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(msg)); err != nil {
			return err
		}
		return w.Close()
	}

	// Port 587, 25, or standard SMTP with STARTTLS
	c, err := smtp.Dial(addr)
	if err != nil {
		log.Printf("[MAILER ERROR] Failed to connect to SMTP at %s: %v", addr, err)
		return err
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: m.cfg.Host,
		}
		if m.cfg.Host == "127.0.0.1" || m.cfg.Host == "localhost" {
			tlsConfig.InsecureSkipVerify = true
		}
		if err := c.StartTLS(tlsConfig); err != nil {
			log.Printf("[MAILER ERROR] STARTTLS failed on %s: %v", addr, err)
			return err
		}
	}

	if m.cfg.User != "" && m.cfg.Password != "" {
		auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)
		if err := c.Auth(auth); err != nil {
			log.Printf("[MAILER ERROR] SMTP auth failed on %s: %v", addr, err)
			return err
		}
	}

	if err := c.Mail(cleanAddress(from)); err != nil {
		return err
	}
	if err := c.Rcpt(toEmail); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return err
	}
	return w.Close()
}

func cleanAddress(addr string) string {
	if start := strings.Index(addr, "<"); start != -1 {
		if end := strings.Index(addr, ">"); end != -1 && end > start {
			return addr[start+1 : end]
		}
	}
	return addr
}
