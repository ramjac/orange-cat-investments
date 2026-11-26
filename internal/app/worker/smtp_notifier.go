package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/orange-cat-investments/oci/pkg/envelope"
)

const defaultSMTPFrom = "noreply@oci.local"

// SMTPNotifier delivers transactional notifications to an in-cluster SMTP relay (Mailpit)
// while strictly redacting recipient PII and message bodies from structured logs (CWE-532).
type SMTPNotifier struct {
	addr    string
	from    string
	logger  *slog.Logger
	timeout time.Duration
}

// NewSMTPNotifierFromEnv constructs an SMTPNotifier when SMTP_HOST is configured, returning nil otherwise.
func NewSMTPNotifierFromEnv(logger *slog.Logger) *SMTPNotifier {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	if host == "" {
		return nil
	}
	if !strings.Contains(host, ":") {
		host += ":1025"
	}
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if from == "" {
		from = defaultSMTPFrom
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &SMTPNotifier{
		addr:    host,
		from:    from,
		logger:  logger,
		timeout: 5 * time.Second,
	}
}

// NewSMTPNotifier constructs an SMTPNotifier targeting an explicit host:port address.
func NewSMTPNotifier(addr, from string, logger *slog.Logger) *SMTPNotifier {
	if from == "" {
		from = defaultSMTPFrom
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &SMTPNotifier{
		addr:    strings.TrimSpace(addr),
		from:    from,
		logger:  logger,
		timeout: 5 * time.Second,
	}
}

// SendEmail sends an RFC 5322 plain-text email without logging PII or message body content (CWE-532).
func (n *SMTPNotifier) SendEmail(ctx context.Context, to, subject, body string) error {
	if n == nil || n.addr == "" {
		return nil
	}
	to = strings.TrimSpace(to)
	if to == "" || !strings.Contains(to, "@") {
		return errors.New("valid recipient email is required")
	}

	// Sanitize header values against CRLF injection.
	cleanTo := sanitizeSMTPHeader(to)
	cleanFrom := sanitizeSMTPHeader(n.from)
	cleanSubject := sanitizeSMTPHeader(subject)

	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", cleanFrom),
		fmt.Sprintf("To: %s", cleanTo),
		fmt.Sprintf("Subject: %s", cleanSubject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	dialer := &net.Dialer{Timeout: n.timeout}
	if deadline, ok := ctx.Deadline(); ok {
		dialer.Deadline = deadline
	}
	conn, err := dialer.DialContext(ctx, "tcp", n.addr)
	if err != nil {
		n.logger.Warn("SMTP dial failed",
			"smtp_host", n.addr,
			"recipient_hash", hashRecipient(cleanTo),
			"error", err,
		)
		return fmt.Errorf("smtp dial %s: %w", n.addr, err)
	}

	hostOnly, _, splitErr := net.SplitHostPort(n.addr)
	if splitErr != nil {
		hostOnly = n.addr
	}
	client, err := smtp.NewClient(conn, hostOnly)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp client init: %w", err)
	}
	defer func() { _ = client.Close() }()

	if err := client.Mail(cleanFrom); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	if err := client.Rcpt(cleanTo); err != nil {
		return fmt.Errorf("smtp RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		_ = w.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	_ = client.Quit()

	// CWE-532: never log raw recipient email, subject financial figures, or body.
	n.logger.Info("transactional email delivered via SMTP",
		"smtp_host", n.addr,
		"recipient_redacted", redactRecipient(cleanTo),
		"recipient_hash", hashRecipient(cleanTo),
	)
	return nil
}

// HandleEmployeeOnboarded sends a welcome onboarding email when an employee onboarding event arrives.
func HandleEmployeeOnboarded(ctx context.Context, logger *slog.Logger, notifier *SMTPNotifier, msg *message.Message) error {
	if notifier == nil {
		return nil
	}
	var env EventEnvelope[envelope.EmployeeOnboardedPayload]
	if err := json.Unmarshal(msg.Payload, &env); err != nil {
		logger.Error("failed to unmarshal employee onboarded event", "uuid", msg.UUID, "error", err)
		return err
	}
	p := env.Payload
	if strings.EqualFold(p.EmployeeType, "feline") {
		return nil
	}
	recipient := strings.TrimSpace(p.Email)
	if recipient == "" && p.FirstName != "" {
		recipient = strings.ToLower(strings.TrimSpace(p.FirstName)) + "@oci.local"
	}
	if recipient == "" {
		return nil
	}

	subject := "Welcome to Orange Cat Investments — Onboarding Checklist Ready"
	body := fmt.Sprintf(
		"Welcome to Orange Cat Investments, %s!\n\nYour employee account (%s) in %s has been provisioned.\nPlease review your onboarding checklist and workstation assignment in the Employee Portal.\n",
		p.FirstName, p.RoleTitle, p.Department,
	)
	return notifier.SendEmail(ctx, recipient, subject, body)
}

// HandleTradeExecutedEmail sends a trade execution confirmation email to the customer.
func HandleTradeExecutedEmail(ctx context.Context, logger *slog.Logger, notifier *SMTPNotifier, msg *message.Message) error {
	if notifier == nil {
		return nil
	}
	var env EventEnvelope[envelope.TradeExecutedPayload]
	if err := json.Unmarshal(msg.Payload, &env); err != nil {
		logger.Error("failed to unmarshal trade executed event", "uuid", msg.UUID, "error", err)
		return err
	}
	p := env.Payload
	recipient := strings.TrimSpace(p.CustomerEmail)
	if recipient == "" {
		return nil
	}

	subject := fmt.Sprintf("OCI Trade Confirmation: %s %s", strings.ToUpper(p.Side), strings.ToUpper(p.Symbol))
	body := fmt.Sprintf(
		"Your %s order (%s) for %.4f shares of %s was processed at $%.2f (status: %s).\nPortfolio: %s\nExecuted At: %s\n",
		strings.ToUpper(p.Side), p.OrderID, p.Quantity, strings.ToUpper(p.Symbol), p.Price, p.Status, p.PortfolioID, p.ExecutedAt,
	)
	return notifier.SendEmail(ctx, recipient, subject, body)
}

// HandleACHSettledEmail sends an ACH settlement confirmation email when an ACH settlement event arrives.
func HandleACHSettledEmail(ctx context.Context, logger *slog.Logger, notifier *SMTPNotifier, msg *message.Message) error {
	if notifier == nil {
		return nil
	}
	var env EventEnvelope[ACHSettledPayload]
	if err := json.Unmarshal(msg.Payload, &env); err != nil {
		logger.Error("failed to unmarshal ach settled event", "uuid", msg.UUID, "error", err)
		return err
	}
	p := env.Payload
	if p.CustomerID == "" {
		return nil
	}
	recipient := strings.ToLower(strings.TrimSpace(p.CustomerID)) + "@oci.local"
	subject := "OCI ACH Transfer Settled"
	body := fmt.Sprintf(
		"Your ACH transfer (%s) for portfolio %s has settled ($%.2f) at %s.\n",
		p.TransferID, p.PortfolioID, p.AmountUSD, p.SettledAt.UTC().Format(time.RFC3339),
	)
	return notifier.SendEmail(ctx, recipient, subject, body)
}

func sanitizeSMTPHeader(v string) string {
	r := strings.NewReplacer("\r", "", "\n", "")
	return strings.TrimSpace(r.Replace(v))
}

func redactRecipient(addr string) string {
	parts := strings.SplitN(addr, "@", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "***"
	}
	return parts[0][:1] + "***@" + parts[1]
}

func hashRecipient(addr string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(addr))))
	return hex.EncodeToString(sum[:8])
}
