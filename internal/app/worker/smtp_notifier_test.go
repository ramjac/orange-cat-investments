package worker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/orange-cat-investments/oci/pkg/envelope"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturedEmail struct {
	mailFrom string
	rcptTo   string
	data     string
}

func startTestSMTPServer(t *testing.T) (string, func() []capturedEmail) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	var mu sync.Mutex
	var captured []capturedEmail

	go func() {
		for {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				rw := bufio.NewReadWriter(bufio.NewReader(c), bufio.NewWriter(c))
				_, _ = rw.WriteString("220 localhost ESMTP test\r\n")
				_ = rw.Flush()

				var cur capturedEmail
				for {
					line, rErr := rw.ReadString('\n')
					if rErr != nil {
						return
					}
					trimmed := strings.TrimRight(line, "\r\n")
					upper := strings.ToUpper(trimmed)
					switch {
					case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
						_, _ = rw.WriteString("250-localhost\r\n250 OK\r\n")
					case strings.HasPrefix(upper, "MAIL FROM:"):
						cur.mailFrom = strings.TrimSpace(trimmed[len("MAIL FROM:"):])
						_, _ = rw.WriteString("250 OK\r\n")
					case strings.HasPrefix(upper, "RCPT TO:"):
						cur.rcptTo = strings.TrimSpace(trimmed[len("RCPT TO:"):])
						_, _ = rw.WriteString("250 OK\r\n")
					case upper == "DATA":
						_, _ = rw.WriteString("354 End data with <CR><LF>.<CR><LF>\r\n")
						_ = rw.Flush()
						var bodyBuf strings.Builder
						for {
							dline, dErr := rw.ReadString('\n')
							if dErr != nil {
								return
							}
							if strings.TrimRight(dline, "\r\n") == "." {
								break
							}
							bodyBuf.WriteString(dline)
						}
						cur.data = bodyBuf.String()
						mu.Lock()
						captured = append(captured, cur)
						mu.Unlock()
						_, _ = rw.WriteString("250 OK\r\n")
					case upper == "QUIT":
						_, _ = rw.WriteString("221 Bye\r\n")
						_ = rw.Flush()
						return
					default:
						_, _ = rw.WriteString("250 OK\r\n")
					}
					_ = rw.Flush()
				}
			}(conn)
		}
	}()

	t.Cleanup(func() { _ = ln.Close() })
	return ln.Addr().String(), func() []capturedEmail {
		mu.Lock()
		defer mu.Unlock()
		out := make([]capturedEmail, len(captured))
		copy(out, captured)
		return out
	}
}

func TestSMTPNotifier_OnboardingAndTradeEmails_CWE532Redaction(t *testing.T) {
	addr, getEmails := startTestSMTPServer(t)

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))
	notifier := NewSMTPNotifier(addr, "noreply@oci.local", logger)

	// 1. Onboarding welcome email
	onboardEnv := envelope.NewEventEnvelope(context.Background(), "evt-onb-1", envelope.TopicEmployeeOnboarded, "emp-1", envelope.EmployeeOnboardedPayload{
		EmployeeID:   "emp-1",
		EmployeeType: "human",
		FirstName:    "Rick",
		LastName:     "Newhire",
		Email:        "rick.newhire@oci.local",
		RoleTitle:    "Junior Operations Associate",
		Department:   "Workforce Operations",
	})
	onboardBytes, err := json.Marshal(onboardEnv)
	require.NoError(t, err)
	require.NoError(t, HandleEmployeeOnboarded(context.Background(), logger, notifier, message.NewMessage("evt-onb-1", onboardBytes)))

	// 2. Trade execution confirmation email
	tradeEnv := envelope.NewEventEnvelope(context.Background(), "evt-trd-1", envelope.TopicTradeExecuted, "ord-1", envelope.TradeExecutedPayload{
		OrderID:       "ord-1",
		CustomerID:    "cust-1",
		CustomerEmail: "chloe@oci.local",
		PortfolioID:   "port-1",
		Symbol:        "NVDA",
		Side:          "buy",
		Quantity:      50,
		Price:         125.75,
		Status:        "filled",
		ExecutedAt:    time.Now().UTC().Format(time.RFC3339),
	})
	tradeBytes, err := json.Marshal(tradeEnv)
	require.NoError(t, err)
	require.NoError(t, HandleTradeExecutedEmail(context.Background(), logger, notifier, message.NewMessage("evt-trd-1", tradeBytes)))

	// 3. ACH settled confirmation email
	achEnv := envelope.NewEventEnvelope(context.Background(), "evt-ach-1", envelope.TopicACHSettled, "tr-1", ACHSettledPayload{
		TransferID:  "tr-1",
		CustomerID:  "arthur",
		PortfolioID: "port-2",
		AmountUSD:   1500.00,
		Status:      "settled",
		SettledAt:   time.Now().UTC(),
	})
	achBytes, err := json.Marshal(achEnv)
	require.NoError(t, err)
	require.NoError(t, HandleACHSettledEmail(context.Background(), logger, notifier, message.NewMessage("evt-ach-1", achBytes)))

	emails := getEmails()
	require.Len(t, emails, 3)

	assert.Contains(t, emails[0].rcptTo, "rick.newhire@oci.local")
	assert.Contains(t, emails[0].data, "Subject: Welcome to Orange Cat Investments")
	assert.Contains(t, emails[0].data, "Welcome to Orange Cat Investments, Rick!")

	assert.Contains(t, emails[1].rcptTo, "chloe@oci.local")
	assert.Contains(t, emails[1].data, "Subject: OCI Trade Confirmation: BUY NVDA")
	assert.Contains(t, emails[1].data, "50.0000 shares of NVDA")

	assert.Contains(t, emails[2].rcptTo, "arthur@oci.local")
	assert.Contains(t, emails[2].data, "Subject: OCI ACH Transfer Settled")

	// CWE-532: structured logs must NOT contain unredacted email addresses or body text
	logs := logBuf.String()
	assert.NotContains(t, logs, "rick.newhire@oci.local")
	assert.NotContains(t, logs, "chloe@oci.local")
	assert.NotContains(t, logs, "arthur@oci.local")
	assert.NotContains(t, logs, "125.75")
	assert.Contains(t, logs, "r***@oci.local")
	assert.Contains(t, logs, "c***@oci.local")
}
