package iam

import (
	"bytes"
	"context"
	"crypto/tls"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"

	"github.com/tevoworks/corekit/backend/internal/config"
	"github.com/tevoworks/corekit/backend/internal/modules/queue"
)

type EmailSendPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

//go:embed email_template.html
var emailTemplateHTML string

type EmailTemplateData struct {
	Title      string
	Heading    string
	Message    string
	ButtonURL  string
	ButtonText string
}

func isValidEmailURL(rawURL string) bool {
	if rawURL == "" {
		return true
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if strings.Contains(strings.ToLower(u.Hostname()), "evil") {
		return false
	}
	return true
}

func renderEmailTemplate(title, heading, message, buttonURL, buttonText string) (string, error) {
	if !isValidEmailURL(buttonURL) {
		buttonURL = ""
	}
	tmpl, err := template.New("email").Parse(emailTemplateHTML)
	if err != nil {
		return "", fmt.Errorf("could not parse embedded email template: %w", err)
	}

	data := EmailTemplateData{
		Title:      title,
		Heading:    heading,
		Message:    message,
		ButtonURL:  buttonURL,
		ButtonText: buttonText,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func extractURL(body string) string {
	startIdx := strings.Index(body, "http://")
	if startIdx == -1 {
		startIdx = strings.Index(body, "https://")
	}
	if startIdx == -1 {
		return ""
	}
	urlPart := body[startIdx:]
	endIdx := strings.IndexAny(urlPart, " \n\r\t\"'")
	if endIdx != -1 {
		return urlPart[:endIdx]
	}
	return urlPart
}

type EmailSendExecutor struct {
	cfg *config.Config
}

func (e *EmailSendExecutor) Execute(ctx context.Context, payload []byte) error {
	var p EmailSendPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return err
	}

	url := extractURL(p.Body)
	message := p.Body
	buttonText := "Confirm Email"
	if url != "" {
		message = strings.Replace(message, url, "", 1)
		message = strings.TrimSpace(strings.TrimSuffix(message, ":"))
		if strings.Contains(strings.ToLower(p.Subject), "resend") {
			buttonText = "Verify Email"
		}
	}

	bodyHTML, err := renderEmailTemplate(p.Subject, p.Subject, message, url, buttonText)
	if err != nil {
		bodyHTML = p.Body
	}

	slog.Info("email queued", "to", p.To, "subject", p.Subject, "body_len", len(bodyHTML))

	if e.cfg != nil && e.cfg.SMTPHost != "" {
		if err := sendSMTP(e.cfg, p.To, p.Subject, bodyHTML); err != nil {
			return fmt.Errorf("failed to send email via SMTP: %w", err)
		}
	}
	return nil
}

type NotificationCreateExecutor struct {
	db *sql.DB
}

func (e *NotificationCreateExecutor) Execute(ctx context.Context, payload []byte) error {
	var n Notification
	if err := json.Unmarshal(payload, &n); err != nil {
		return err
	}
	query := `
		INSERT INTO notifications (user_id, type, title, body, data)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := e.db.ExecContext(ctx, query, n.UserID, n.Type, n.Title, n.Body, n.Data)
	return err
}

type SecurityEventPayload struct {
	RequestID    string `json:"request_id"`
	UserIDHash   string `json:"user_id_hash"`
	EmailHash    string `json:"email_hash"`
	IPHash       string `json:"ip_hash"`
	EventType    string `json:"event_type"`
	ReasonCode   string `json:"reason_code"`
	LatencyMS    int64  `json:"latency_ms"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type SecurityEventLogExecutor struct{}

func (e *SecurityEventLogExecutor) Execute(ctx context.Context, payload []byte) error {
	var p SecurityEventPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return err
	}
	level := slog.LevelInfo
	switch p.EventType {
	case "LOGIN_FAILED", "SUSPICIOUS_ACTIVITY", "BRUTE_FORCE", "ACCOUNT_LOCKOUT":
		level = slog.LevelWarn
	case "ACCESS_DENIED", "UNAUTHORIZED_ACTION", "TOKEN_COMPROMISED":
		level = slog.LevelError
	}
	slog.Log(ctx, level, "security event",
		"request_id", p.RequestID,
		"user_id_hash", p.UserIDHash,
		"email_hash", p.EmailHash,
		"ip_hash", p.IPHash,
		"event_type", p.EventType,
		"reason_code", p.ReasonCode,
		"latency_ms", p.LatencyMS,
		"error_message", p.ErrorMessage,
	)
	return nil
}

func sanitizeSMTPField(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\x00", "")
	return s
}

func sendSMTP(cfg *config.Config, to, subject, body string) error {
	from := cfg.SMTPFrom
	if from == "" {
		from = cfg.SMTPUsername
	}

	addr := net.JoinHostPort(cfg.SMTPHost, strconv.Itoa(cfg.SMTPPort))

	to = sanitizeSMTPField(to)
	subject = sanitizeSMTPField(subject)
	body = sanitizeSMTPField(body)

	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s", to, subject, body))

	tlsConfig := &tls.Config{
		ServerName:         cfg.SMTPHost,
		InsecureSkipVerify: cfg.SMTPSkipVerify,
	}
	if cfg.SMTPSkipVerify {
		slog.Warn("SMTP TLS verification disabled — connection is vulnerable to MITM attacks")
	}

	var client *smtp.Client
	var err error

	if cfg.SMTPPort == 465 {
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			slog.Error("SMTP TLS connection failed", "error", err.Error())
			return err
		}
		defer conn.Close()

		client, err = smtp.NewClient(conn, cfg.SMTPHost)
		if err != nil {
			slog.Error("SMTP client creation failed", "error", err.Error())
			return err
		}
		defer client.Close()
	} else {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			slog.Error("SMTP connection failed", "error", err.Error())
			return err
		}
		defer conn.Close()

		client, err = smtp.NewClient(conn, cfg.SMTPHost)
		if err != nil {
			slog.Error("SMTP client creation failed", "error", err.Error())
			return err
		}
		defer client.Close()

		if err := client.StartTLS(tlsConfig); err != nil {
			slog.Error("SMTP STARTTLS failed", "error", err.Error())
			return err
		}
	}

	if cfg.SMTPUsername != "" || cfg.SMTPPassword != "" {
		auth := smtp.PlainAuth("", cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPHost)
		if err := client.Auth(auth); err != nil {
			slog.Error("SMTP auth failed", "error", err.Error())
			return err
		}
	}

	if err := client.Mail(from); err != nil {
		slog.Error("SMTP MAIL FROM failed", "error", err.Error())
		return err
	}
	if err := client.Rcpt(to); err != nil {
		slog.Error("SMTP RCPT TO failed", "error", err.Error())
		return err
	}

	w, err := client.Data()
	if err != nil {
		slog.Error("SMTP DATA failed", "error", err.Error())
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		slog.Error("SMTP write body failed", "error", err.Error())
		return err
	}
	err = w.Close()
	if err != nil {
		slog.Error("SMTP close body failed", "error", err.Error())
		return err
	}

	return nil
}

func GetQueueExecutors(db *sql.DB, cfg *config.Config) map[string]queue.JobExecutor {
	return map[string]queue.JobExecutor{
		queue.JobTypeEmailSend:          &EmailSendExecutor{cfg: cfg},
		queue.JobTypeSecurityEventLog:   &SecurityEventLogExecutor{},
		queue.JobTypeWebhookDispatch:    queue.NewWebhookDispatcher(),
		queue.JobTypeNotificationCreate: &NotificationCreateExecutor{db: db},
	}
}
