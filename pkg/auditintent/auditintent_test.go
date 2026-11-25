package auditintent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(os.Stderr, nil)) }

func mustSpool(t *testing.T) *FileSpool {
	t.Helper()
	sp, err := NewFileSpool(t.TempDir(), testLogger())
	if err != nil {
		t.Fatalf("NewFileSpool: %v", err)
	}
	return sp
}

func TestAddDueAckRoundtrip(t *testing.T) {
	sp := mustSpool(t)
	ctx := context.Background()

	if _, err := sp.Add(ctx, Entry{Action: "a1", OccurredAt: time.Now().UTC().Add(-time.Second)}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := sp.Add(ctx, Entry{Action: "a2"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	due, err := sp.Due(ctx, time.Now().UTC(), 10)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due) != 2 {
		t.Fatalf("want 2 due, got %d", len(due))
	}
	if due[0].Action != "a1" || due[1].Action != "a2" {
		t.Fatalf("unexpected ordering: %q %q", due[0].Action, due[1].Action)
	}

	if err := sp.Ack(ctx, due[0].ID); err != nil {
		t.Fatalf("Ack: %v", err)
	}
	// Double ack is a no-op.
	if err := sp.Ack(ctx, due[0].ID); err != nil {
		t.Fatalf("double Ack: %v", err)
	}

	due, _ = sp.Due(ctx, time.Now().UTC(), 10)
	if len(due) != 1 || due[0].Action != "a2" {
		t.Fatalf("want only a2 pending, got %+v", due)
	}

	if n, _ := sp.PendingCount(); n != 1 {
		t.Fatalf("want 1 pending, got %d", n)
	}
}

func TestRetryBackoffSpacing(t *testing.T) {
	sp := mustSpool(t)
	ctx := context.Background()
	now := time.Now().UTC()

	stored, err := sp.Add(ctx, Entry{Action: "flaky"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	stored.Attempts = 1
	stored.NextRetryAt = now.Add(time.Minute)
	if err := sp.Retry(ctx, stored); err != nil {
		t.Fatalf("Retry: %v", err)
	}

	// Not due before NextRetryAt.
	due, _ := sp.Due(ctx, now, 10)
	if len(due) != 0 {
		t.Fatalf("entry should be backoff-spaced, got %d due", len(due))
	}
	// Due after the spacing.
	due, _ = sp.Due(ctx, now.Add(time.Minute+time.Second), 10)
	if len(due) != 1 || due[0].Attempts != 1 {
		t.Fatalf("want 1 due with attempts=1, got %+v", due)
	}
}

func TestCrashLeavesNoPartialFiles(t *testing.T) {
	sp := mustSpool(t)
	ctx := context.Background()
	for i := 0; i < 25; i++ {
		if _, err := sp.Add(ctx, Entry{Action: fmt.Sprintf("a%d", i)}); err != nil {
			t.Fatalf("Add %d: %v", i, err)
		}
	}
	entries, err := os.ReadDir(sp.pendDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			t.Fatalf("unexpected file in pending: %s", e.Name())
		}
	}
	if n, _ := sp.PendingCount(); n != 25 {
		t.Fatalf("want 25 pending, got %d", n)
	}
}

func TestRelayDeliversAndRetries(t *testing.T) {
	sp := mustSpool(t)
	ctx := context.Background()
	rec := NewRecorder(sp, testLogger())

	var (
		callMu       sync.Mutex
		deliverCalls []string
	)
	deliver := func(ctx context.Context, e *Entry) error {
		callMu.Lock()
		deliverCalls = append(deliverCalls, e.Action)
		callMu.Unlock()
		if e.Action == "flaky" {
			return errors.New("ops-api unavailable")
		}
		return nil
	}
	snapshotCalls := func() []string {
		callMu.Lock()
		defer callMu.Unlock()
		return append([]string(nil), deliverCalls...)
	}

	if _, err := rec.Record(ctx, Entry{Action: "flaky"}); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if _, err := rec.Record(ctx, Entry{Action: "ok"}); err != nil {
		t.Fatalf("Record: %v", err)
	}

	relay := NewRelay(sp, deliver, RelayConfig{Tick: 10 * time.Millisecond, Logger: testLogger()})
	relay.Start()
	defer relay.Stop()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		due, _ := sp.Due(context.Background(), time.Now().UTC(), 10)
		okPending := false
		for _, e := range due {
			if e.Action == "ok" {
				okPending = true
			}
		}
		if !okPending {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	if !contains(snapshotCalls(), "ok") {
		t.Fatalf("relay never delivered ok intent: %v", snapshotCalls())
	}
	// The flaky intent must still be pending (retried, not dropped).
	due, _ := sp.Due(context.Background(), time.Now().UTC().Add(time.Hour), 10)
	found := false
	for _, e := range due {
		if e.Action == "flaky" && e.Attempts >= 1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("flaky intent was not retried; due=%+v", due)
	}
}

func TestBackoffCapped(t *testing.T) {
	if d := backoff(1); d < baseBackoff || d > baseBackoff+baseBackoff/4 {
		t.Fatalf("backoff(1)=%v out of range", d)
	}
	if d := backoff(99); d > maxBackoff+maxBackoff/4 {
		t.Fatalf("backoff(99)=%v exceeds cap", d)
	}
}

func TestPruneRemovesOldAcked(t *testing.T) {
	sp := mustSpool(t)
	ctx := context.Background()
	e, err := sp.Add(ctx, Entry{Action: "done"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := sp.Ack(ctx, e.ID); err != nil {
		t.Fatalf("Ack: %v", err)
	}
	// Backdate the acked file.
	acked, _ := os.ReadDir(sp.ackDir)
	old := time.Now().Add(-48 * time.Hour)
	for _, a := range acked {
		_ = os.Chtimes(filepath.Join(sp.ackDir, a.Name()), old, old)
	}
	if n := sp.Prune(time.Now().UTC(), 24*time.Hour); n != 1 {
		t.Fatalf("want 1 pruned, got %d", n)
	}
	if n, _ := sp.PendingCount(); n != 0 {
		t.Fatalf("pending should be 0, got %d", n)
	}
}

func contains(vals []string, want string) bool {
	for _, v := range vals {
		if v == want {
			return true
		}
	}
	return false
}
