// Package auditintent provides a durable local spool for audit intents that
// must reach the tamper-evident audit ledger (ops.audit_logs) even when the
// ops-api is unreachable, the originating request is cancelled, or the BLL
// process restarts between a committed mutation and the audit append.
//
// The pattern follows the transactional-outbox principle: the intent is
// written durably (atomic file rename + fsync) as part of the request path,
// and a background relay re-delivers every unacknowledged intent to the
// audit ledger with capped exponential backoff until it is accepted.
// Delivery is at-least-once; a duplicated ledger entry during an ambiguous
// relay is preferred over a lost entry for an already-committed mutation.
package auditintent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// State is the delivery state of a single audit intent.
type State string

const (
	// StatePending means the intent has been durably spooled and still needs
	// (or is being retried toward) delivery to the audit ledger.
	StatePending State = "pending"
)

// Entry is one durable audit intent. Fields mirror opsv1.AppendAuditLogRequest
// so a relay can convert an entry to the RPC one-to-one.
type Entry struct {
	ID           string    `json:"id"`
	ActorID      string    `json:"actor_id"`
	ActorRole    string    `json:"actor_role"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	MetadataJSON string    `json:"metadata_json"`
	OccurredAt   time.Time `json:"occurred_at"`
	Attempts     int       `json:"attempts,omitempty"`
	NextRetryAt  time.Time `json:"next_retry_at,omitempty"`
}

// DeliverFunc delivers one entry to the tamper-evident audit ledger. It is
// invoked by the relay with a fresh, request-independent context and must
// return nil only when the ledger accepted the entry.
type DeliverFunc func(ctx context.Context, e *Entry) error

// Defaults for the spool/relay.
const (
	// DefaultTick is how often the relay wakes to look for due intents.
	DefaultTick = 500 * time.Millisecond
	// DefaultBatchSize caps how many intents are handled per relay pass.
	DefaultBatchSize = 25
	// DefaultRetention is how long acknowledged entries are kept on disk for
	// post-incident review before pruning.
	DefaultRetention = 7 * 24 * time.Hour
	// maxBackoff caps per-intent retry spacing.
	maxBackoff = 5 * time.Minute
	// baseBackoff is the first retry spacing.
	baseBackoff = 5 * time.Second
	// deliverTimeout bounds a single ledger append RPC.
	deliverTimeout = 15 * time.Second
)

// Spool is a durable store of unacknowledged audit intents.
type Spool interface {
	// Add durably records e and returns the stored entry (with its
	// assigned ID). It must be safe to call concurrently.
	Add(ctx context.Context, e Entry) (Entry, error)
	// Due returns up to limit pending entries whose NextRetryAt has passed
	// (or that have no retry time yet), oldest first.
	Due(ctx context.Context, now time.Time, limit int) ([]Entry, error)
	// Ack removes the pending intent for id after successful delivery.
	Ack(ctx context.Context, id string) error
	// Retry records a failed delivery attempt, spacing the next one out.
	Retry(ctx context.Context, e Entry) error
	// Prune removes acknowledged entries older than retention, returning
	// the number removed.
	Prune(now time.Time, retention time.Duration) int
	// Close releases any resources held by the spool.
	Close() error
}

// FileSpool is a directory-backed Spool: one atomic JSON file per pending
// intent under root/pending, moved to root/acked on success.
type FileSpool struct {
	root    string
	pendDir string
	ackDir  string
	logger  *slog.Logger
}

// NewFileSpool binds the spool to root. Directories are created lazily on
// the first durable write, so healthy operation (synchronous ledger append
// always succeeding) leaves no state behind.
func NewFileSpool(root string, logger *slog.Logger) (*FileSpool, error) {
	if root == "" {
		return nil, errors.New("auditintent: spool root must not be empty")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &FileSpool{
		root:    root,
		pendDir: filepath.Join(root, "pending"),
		ackDir:  filepath.Join(root, "acked"),
		logger:  logger,
	}, nil
}

// ensureDirs creates the spool directories on demand (idempotent).
func (s *FileSpool) ensureDirs() error {
	for _, d := range []string{s.pendDir, s.ackDir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return fmt.Errorf("auditintent: create %s: %w", d, err)
		}
	}
	return nil
}

func (s *FileSpool) entryPath(id string) string {
	return filepath.Join(s.pendDir, id+".json")
}

// Add durably appends the entry: temp file + fsync + atomic rename, so a
// crash can never leave a half-written intent behind.
func (s *FileSpool) Add(_ context.Context, e Entry) (Entry, error) {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now().UTC()
	}
	if err := s.ensureDirs(); err != nil {
		return e, err
	}
	raw, err := json.Marshal(e)
	if err != nil {
		return e, fmt.Errorf("auditintent: marshal intent %s: %w", e.ID, err)
	}
	final := s.entryPath(e.ID)
	tmp, err := os.CreateTemp(s.pendDir, e.ID+".tmp-*")
	if err != nil {
		return e, fmt.Errorf("auditintent: create temp for %s: %w", e.ID, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after successful rename

	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return e, fmt.Errorf("auditintent: write intent %s: %w", e.ID, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return e, fmt.Errorf("auditintent: sync intent %s: %w", e.ID, err)
	}
	if err := tmp.Close(); err != nil {
		return e, fmt.Errorf("auditintent: close intent %s: %w", e.ID, err)
	}
	if err := os.Rename(tmpName, final); err != nil {
		return e, fmt.Errorf("auditintent: persist intent %s: %w", e.ID, err)
	}
	return e, nil
}

// Due lists pending intents that are ready for (re)delivery.
func (s *FileSpool) Due(_ context.Context, now time.Time, limit int) ([]Entry, error) {
	if limit <= 0 {
		limit = DefaultBatchSize
	}
	names, err := os.ReadDir(s.pendDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("auditintent: list pending: %w", err)
	}
	var due []Entry
	for _, n := range names {
		if n.IsDir() || filepath.Ext(n.Name()) != ".json" {
			continue
		}
		id := n.Name()[:len(n.Name())-len(".json")]
		if hasSuffix(id, ".tmp-") {
			continue
		}
		raw, err := os.ReadFile(s.entryPath(id))
		if err != nil {
			// Concurrent rename (Add/ack) — skip this pass.
			continue
		}
		var e Entry
		if err := json.Unmarshal(raw, &e); err != nil {
			s.logger.Error("auditintent: corrupt pending entry; leaving for inspection", "id", id, "error", err)
			continue
		}
		if !e.NextRetryAt.IsZero() && now.Before(e.NextRetryAt) {
			continue
		}
		due = append(due, e)
		if len(due) >= limit {
			break
		}
	}
	sort.Slice(due, func(i, j int) bool {
		if !due[i].OccurredAt.Equal(due[j].OccurredAt) {
			return due[i].OccurredAt.Before(due[j].OccurredAt)
		}
		return due[i].ID < due[j].ID
	})
	return due, nil
}

// Ack moves the intent from pending to acked (atomic rename).
func (s *FileSpool) Ack(_ context.Context, id string) error {
	src := s.entryPath(id)
	dst := filepath.Join(s.ackDir, id+".json")
	if err := os.Rename(src, dst); err != nil {
		if os.IsNotExist(err) {
			return nil // already acknowledged or pruned
		}
		return fmt.Errorf("auditintent: ack %s: %w", id, err)
	}
	return nil
}

// Retry rewrites the intent in place with updated backoff state.
func (s *FileSpool) Retry(_ context.Context, e Entry) error {
	raw, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("auditintent: marshal retry state for %s: %w", e.ID, err)
	}
	final := s.entryPath(e.ID)
	tmp, err := os.CreateTemp(s.pendDir, e.ID+".tmp-*")
	if err != nil {
		return fmt.Errorf("auditintent: create temp for retry %s: %w", e.ID, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return fmt.Errorf("auditintent: write retry state for %s: %w", e.ID, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("auditintent: sync retry state for %s: %w", e.ID, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("auditintent: close retry state for %s: %w", e.ID, err)
	}
	if err := os.Rename(tmpName, final); err != nil {
		return fmt.Errorf("auditintent: persist retry state for %s: %w", e.ID, err)
	}
	return nil
}

// Prune removes acknowledged entries older than retention.
func (s *FileSpool) Prune(now time.Time, retention time.Duration) int {
	if retention <= 0 {
		retention = DefaultRetention
	}
	removed := 0
	names, err := os.ReadDir(s.ackDir)
	if err != nil {
		return 0
	}
	cutoff := now.Add(-retention)
	for _, n := range names {
		if n.IsDir() || filepath.Ext(n.Name()) != ".json" {
			continue
		}
		info, err := n.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			if os.Remove(filepath.Join(s.ackDir, n.Name())) == nil {
				removed++
			}
		}
	}
	return removed
}

// PendingCount reports how many intents await delivery.
func (s *FileSpool) PendingCount() (int, error) {
	names, err := os.ReadDir(s.pendDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	n := 0
	for _, f := range names {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".json" && !hasSuffix(f.Name()[:len(f.Name())-len(".json")], ".tmp-") {
			n++
		}
	}
	return n, nil
}

// Close is a no-op for the file spool but satisfies Spool.
func (s *FileSpool) Close() error { return nil }

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// backoff computes the capped exponential retry spacing for attempt number
// n (1-based), with up to 25% jitter so a fleet of BLL replicas does not
// retry in lockstep.
func backoff(attempts int) time.Duration {
	if attempts <= 0 {
		attempts = 1
	}
	shift := attempts - 1
	if shift > 6 {
		shift = 6
	}
	d := time.Duration(1<<uint(shift)) * baseBackoff
	if d > maxBackoff {
		d = maxBackoff
	}
	jitter := time.Duration(float64(d) * 0.25 * rand.Float64()) // #nosec G404 -- non-cryptographic retry backoff jitter
	return d + jitter
}

// Recorder durably spools audit intents.
type Recorder struct {
	spool  Spool
	logger *slog.Logger
}

// NewRecorder wraps a durable spool.
func NewRecorder(spool Spool, logger *slog.Logger) *Recorder {
	if logger == nil {
		logger = slog.Default()
	}
	return &Recorder{spool: spool, logger: logger}
}

// Record durably persists one audit intent and returns the stored entry.
// It is the only write path: the caller must treat a returned error as
// critical (the intent for an already-committed mutation could not be made
// durable). The write is synchronous and local, so it is independent of
// ops-api health and of the caller's request context.
func (r *Recorder) Record(ctx context.Context, e Entry) (Entry, error) {
	if r.spool == nil {
		return e, errors.New("auditintent: recorder has no spool")
	}
	stored, err := r.spool.Add(ctx, e)
	if err != nil {
		r.logger.Error("CRITICAL: failed to durably record audit intent; entry for committed operation may be lost",
			"action", e.Action, "resource_type", e.ResourceType, "resource_id", e.ResourceID, "error", err)
		return stored, err
	}
	return stored, nil
}

// Spool exposes the underlying spool (for pending-count health checks).
func (r *Recorder) Spool() Spool { return r.spool }

// RelayConfig tunes the delivery loop.
type RelayConfig struct {
	Tick      time.Duration
	BatchSize int
	Retention time.Duration
	Logger    *slog.Logger
}

// Relay delivers spooled intents to the audit ledger with retries.
type Relay struct {
	spool   Spool
	deliver DeliverFunc
	cfg     RelayConfig
	logger  *slog.Logger

	stopOnce sync.Once
	stop     chan struct{}
	done     chan struct{}
	started  bool
	mu       sync.Mutex
}

// NewRelay builds a relay. deliver must be non-nil.
func NewRelay(spool Spool, deliver DeliverFunc, cfg RelayConfig) *Relay {
	if cfg.Tick <= 0 {
		cfg.Tick = DefaultTick
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = DefaultBatchSize
	}
	if cfg.Retention <= 0 {
		cfg.Retention = DefaultRetention
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &Relay{
		spool:   spool,
		deliver: deliver,
		cfg:     cfg,
		logger:  cfg.Logger,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
}

// Start begins the background delivery loop (idempotent).
func (r *Relay) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started {
		return
	}
	r.started = true
	go r.run()
}

// Stop terminates the loop and waits for it to drain.
func (r *Relay) Stop() {
	r.stopOnce.Do(func() { close(r.stop) })
	<-r.done
}

func (r *Relay) run() {
	defer close(r.done)
	ticker := time.NewTicker(r.cfg.Tick)
	defer ticker.Stop()
	pruneTicker := time.NewTicker(time.Hour)
	defer pruneTicker.Stop()
	for {
		select {
		case <-r.stop:
			return
		case <-ticker.C:
			r.pass()
		case <-pruneTicker.C:
			removed := r.spool.Prune(time.Now().UTC(), r.cfg.Retention)
			if removed > 0 {
				r.logger.Debug("auditintent: pruned acknowledged entries", "removed", removed)
			}
		}
	}
}

func (r *Relay) pass() {
	now := time.Now().UTC()
	due, err := r.spool.Due(context.Background(), now, r.cfg.BatchSize)
	if err != nil {
		r.logger.Warn("auditintent: failed to list due intents", "error", err)
		return
	}
	for i := range due {
		e := due[i]
		if r.deliver == nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), deliverTimeout)
		err := r.deliver(ctx, &e)
		cancel()
		if err == nil {
			if ackErr := r.spool.Ack(context.Background(), e.ID); ackErr != nil {
				// Delivered but not acknowledged: the next pass redelivers.
				// At-least-once is intentional (duplicate beats loss).
				r.logger.Warn("auditintent: delivered but ack failed; entry will redeliver", "id", e.ID, "error", ackErr)
			}
			continue
		}
		e.Attempts++
		e.NextRetryAt = now.Add(backoff(e.Attempts))
		if retryErr := r.spool.Retry(context.Background(), e); retryErr != nil {
			r.logger.Error("auditintent: failed to persist retry state", "id", e.ID, "error", retryErr)
			continue
		}
		if e.Attempts == 1 || e.Attempts%20 == 0 {
			r.logger.Warn("auditintent: audit ledger delivery failed; retrying with backoff",
				"id", e.ID, "action", e.Action, "attempts", e.Attempts, "next_retry_in", e.NextRetryAt.Sub(now).Round(time.Second), "error", err)
		}
	}
}
