package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/orange-cat-investments/oci/pkg/envelope"
)

// Schema identifies one of the four isolated OCI PostgreSQL domain schemas.
type Schema string

const (
	SchemaWorkforce  Schema = "workforce"
	SchemaFacilities Schema = "facilities"
	SchemaCoreInvest Schema = "core_invest"
	SchemaOps        Schema = "ops"
)

// DefaultLeaseDuration is the claim lease window for FOR UPDATE SKIP LOCKED relay batches.
const DefaultLeaseDuration = 30 * time.Second

// Record represents a row in <schema>.outbox_events.
type Record struct {
	ID          int64      `json:"id"`
	Schema      Schema     `json:"schema"`
	EventID     string     `json:"event_id"`
	Topic       string     `json:"topic"`
	AggregateID string     `json:"aggregate_id"`
	Payload     []byte     `json:"payload"`
	Status      string     `json:"status"`
	Attempts    int        `json:"attempts"`
	LastError   string     `json:"last_error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

// Failure describes a publish failure for a claimed outbox row.
type Failure struct {
	ID     int64  `json:"id"`
	Schema Schema `json:"schema"`
	Error  string `json:"error"`
}

// ExecQuerier is the minimal pgx surface implemented by *pgxpool.Pool and pgx.Tx.
type ExecQuerier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// schemaQueries maps each allowlisted domain schema to its static, parameterized SQL statements.
// Using a static map rather than fmt.Sprintf prevents any dynamic SQL construction (gosec G201/G202).
type schemaQueries struct {
	insertSQL        string
	claimSQL         string
	markPublishedSQL string
	markFailedSQL    string
}

var queriesBySchema = map[Schema]schemaQueries{
	SchemaWorkforce: {
		insertSQL: `INSERT INTO workforce.outbox_events (event_id, topic, aggregate_id, payload, status)
VALUES ($1, $2, $3, $4::jsonb, 'pending')
ON CONFLICT (event_id) DO NOTHING`,
		claimSQL: `WITH claimed AS (
	SELECT id FROM workforce.outbox_events
	WHERE published_at IS NULL
	  AND (locked_until IS NULL OR locked_until < NOW())
	ORDER BY id ASC
	LIMIT $1
	FOR UPDATE SKIP LOCKED
)
UPDATE workforce.outbox_events o
SET status = 'processing',
    attempts = o.attempts + 1,
    locked_until = NOW() + ($2 * INTERVAL '1 millisecond')
FROM claimed
WHERE o.id = claimed.id
RETURNING o.id, o.event_id::text, o.topic, o.aggregate_id, o.payload::text, o.status, o.attempts, o.created_at`,
		markPublishedSQL: `UPDATE workforce.outbox_events
SET status = 'published', published_at = NOW(), locked_until = NULL, last_error = NULL
WHERE id = ANY($1::bigint[])`,
		markFailedSQL: `UPDATE workforce.outbox_events
SET status = 'pending',
    last_error = $2,
    locked_until = NOW() + ($3 * INTERVAL '1 millisecond')
WHERE id = $1 AND published_at IS NULL`,
	},
	SchemaFacilities: {
		insertSQL: `INSERT INTO facilities.outbox_events (event_id, topic, aggregate_id, payload, status)
VALUES ($1, $2, $3, $4::jsonb, 'pending')
ON CONFLICT (event_id) DO NOTHING`,
		claimSQL: `WITH claimed AS (
	SELECT id FROM facilities.outbox_events
	WHERE published_at IS NULL
	  AND (locked_until IS NULL OR locked_until < NOW())
	ORDER BY id ASC
	LIMIT $1
	FOR UPDATE SKIP LOCKED
)
UPDATE facilities.outbox_events o
SET status = 'processing',
    attempts = o.attempts + 1,
    locked_until = NOW() + ($2 * INTERVAL '1 millisecond')
FROM claimed
WHERE o.id = claimed.id
RETURNING o.id, o.event_id::text, o.topic, o.aggregate_id, o.payload::text, o.status, o.attempts, o.created_at`,
		markPublishedSQL: `UPDATE facilities.outbox_events
SET status = 'published', published_at = NOW(), locked_until = NULL, last_error = NULL
WHERE id = ANY($1::bigint[])`,
		markFailedSQL: `UPDATE facilities.outbox_events
SET status = 'pending',
    last_error = $2,
    locked_until = NOW() + ($3 * INTERVAL '1 millisecond')
WHERE id = $1 AND published_at IS NULL`,
	},
	SchemaCoreInvest: {
		insertSQL: `INSERT INTO core_invest.outbox_events (event_id, topic, aggregate_id, payload, status)
VALUES ($1, $2, $3, $4::jsonb, 'pending')
ON CONFLICT (event_id) DO NOTHING`,
		claimSQL: `WITH claimed AS (
	SELECT id FROM core_invest.outbox_events
	WHERE published_at IS NULL
	  AND (locked_until IS NULL OR locked_until < NOW())
	ORDER BY id ASC
	LIMIT $1
	FOR UPDATE SKIP LOCKED
)
UPDATE core_invest.outbox_events o
SET status = 'processing',
    attempts = o.attempts + 1,
    locked_until = NOW() + ($2 * INTERVAL '1 millisecond')
FROM claimed
WHERE o.id = claimed.id
RETURNING o.id, o.event_id::text, o.topic, o.aggregate_id, o.payload::text, o.status, o.attempts, o.created_at`,
		markPublishedSQL: `UPDATE core_invest.outbox_events
SET status = 'published', published_at = NOW(), locked_until = NULL, last_error = NULL
WHERE id = ANY($1::bigint[])`,
		markFailedSQL: `UPDATE core_invest.outbox_events
SET status = 'pending',
    last_error = $2,
    locked_until = NOW() + ($3 * INTERVAL '1 millisecond')
WHERE id = $1 AND published_at IS NULL`,
	},
	SchemaOps: {
		insertSQL: `INSERT INTO ops.outbox_events (event_id, topic, aggregate_id, payload, status)
VALUES ($1, $2, $3, $4::jsonb, 'pending')
ON CONFLICT (event_id) DO NOTHING`,
		claimSQL: `WITH claimed AS (
	SELECT id FROM ops.outbox_events
	WHERE published_at IS NULL
	  AND (locked_until IS NULL OR locked_until < NOW())
	ORDER BY id ASC
	LIMIT $1
	FOR UPDATE SKIP LOCKED
)
UPDATE ops.outbox_events o
SET status = 'processing',
    attempts = o.attempts + 1,
    locked_until = NOW() + ($2 * INTERVAL '1 millisecond')
FROM claimed
WHERE o.id = claimed.id
RETURNING o.id, o.event_id::text, o.topic, o.aggregate_id, o.payload::text, o.status, o.attempts, o.created_at`,
		markPublishedSQL: `UPDATE ops.outbox_events
SET status = 'published', published_at = NOW(), locked_until = NULL, last_error = NULL
WHERE id = ANY($1::bigint[])`,
		markFailedSQL: `UPDATE ops.outbox_events
SET status = 'pending',
    last_error = $2,
    locked_until = NOW() + ($3 * INTERVAL '1 millisecond')
WHERE id = $1 AND published_at IS NULL`,
	},
}

// ValidateSchema verifies that s is one of the four OCI domain schemas.
func ValidateSchema(s Schema) error {
	if _, ok := queriesBySchema[s]; !ok {
		return fmt.Errorf("invalid outbox schema: %q", s)
	}
	return nil
}

// NewEventID returns a valid RFC 4122 UUIDv7 (falling back to UUIDv4 if unavailable).
func NewEventID() string {
	if id, err := uuid.NewV7(); err == nil {
		return id.String()
	}
	return uuid.NewString()
}

// BuildEnvelopeBytes marshals a standard EventEnvelope[T] for outbox storage.
func BuildEnvelopeBytes[T any](ctx context.Context, eventID, topic, aggregateID string, payload T) (string, []byte, error) {
	if eventID == "" {
		eventID = NewEventID()
	} else if _, err := uuid.Parse(eventID); err != nil {
		eventID = NewEventID()
	}
	env := envelope.NewEventEnvelope(ctx, eventID, topic, aggregateID, payload)
	raw, err := json.Marshal(env)
	if err != nil {
		return "", nil, fmt.Errorf("marshal outbox envelope: %w", err)
	}
	return eventID, raw, nil
}

// InsertTx persists an outbox event row within an active pgx.Tx (or ExecQuerier).
func InsertTx(ctx context.Context, exec ExecQuerier, schema Schema, eventID, topic, aggregateID string, envelopeBytes []byte) error {
	q, ok := queriesBySchema[schema]
	if !ok {
		return fmt.Errorf("invalid outbox schema: %q", schema)
	}
	if exec == nil {
		return errors.New("outbox executor is nil")
	}
	if topic == "" {
		return errors.New("outbox topic is required")
	}
	if eventID == "" {
		eventID = NewEventID()
	} else if _, err := uuid.Parse(eventID); err != nil {
		eventID = NewEventID()
	}
	if len(envelopeBytes) == 0 {
		envelopeBytes = []byte(`{}`)
	}
	_, err := exec.Exec(ctx, q.insertSQL, eventID, topic, aggregateID, string(envelopeBytes))
	return err
}

// ClaimPending atomically leases up to limit unpublished rows using FOR UPDATE SKIP LOCKED.
func ClaimPending(ctx context.Context, db ExecQuerier, schema Schema, limit int32, lease time.Duration) ([]*Record, error) {
	q, ok := queriesBySchema[schema]
	if !ok {
		return nil, fmt.Errorf("invalid outbox schema: %q", schema)
	}
	if db == nil {
		return nil, errors.New("outbox database connection is nil")
	}
	if limit <= 0 {
		limit = 50
	}
	if lease <= 0 {
		lease = DefaultLeaseDuration
	}

	rows, err := db.Query(ctx, q.claimSQL, limit, lease.Milliseconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Record
	for rows.Next() {
		var rec Record
		var payloadStr string
		if err := rows.Scan(&rec.ID, &rec.EventID, &rec.Topic, &rec.AggregateID, &payloadStr, &rec.Status, &rec.Attempts, &rec.CreatedAt); err != nil {
			return nil, err
		}
		rec.Schema = schema
		rec.Payload = []byte(payloadStr)
		out = append(out, &rec)
	}
	return out, rows.Err()
}

// MarkPublished marks the specified outbox row IDs as published.
func MarkPublished(ctx context.Context, db ExecQuerier, schema Schema, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	q, ok := queriesBySchema[schema]
	if !ok {
		return fmt.Errorf("invalid outbox schema: %q", schema)
	}
	if db == nil {
		return errors.New("outbox database connection is nil")
	}
	_, err := db.Exec(ctx, q.markPublishedSQL, ids)
	return err
}

// MarkFailed records a publish error and releases/delays the lock so the row will be retried.
func MarkFailed(ctx context.Context, db ExecQuerier, schema Schema, id int64, errMsg string, retryAfter time.Duration) error {
	q, ok := queriesBySchema[schema]
	if !ok {
		return fmt.Errorf("invalid outbox schema: %q", schema)
	}
	if db == nil {
		return errors.New("outbox database connection is nil")
	}
	if retryAfter < 0 {
		retryAfter = 0
	}
	_, err := db.Exec(ctx, q.markFailedSQL, id, errMsg, retryAfter.Milliseconds())
	return err
}

// --- In-Memory Outbox Store (for offline repository mocks and race tests) ---

type memoryEntry struct {
	rec         Record
	lockedUntil time.Time
}

// MemoryStore provides thread-safe in-memory outbox semantics mirroring PostgreSQL FOR UPDATE SKIP LOCKED.
type MemoryStore struct {
	mu      sync.Mutex
	schema  Schema
	nextID  int64
	entries []*memoryEntry
	byEvent map[string]struct{}
}

// NewMemoryStore creates an in-memory outbox store for the given domain schema.
func NewMemoryStore(schema Schema) *MemoryStore {
	return &MemoryStore{
		schema:  schema,
		byEvent: make(map[string]struct{}),
	}
}

// Enqueue appends an unpublished event row (idempotent by eventID).
func (m *MemoryStore) Enqueue(eventID, topic, aggregateID string, payload []byte) *Record {
	m.mu.Lock()
	defer m.mu.Unlock()
	if eventID == "" {
		eventID = NewEventID()
	}
	if _, exists := m.byEvent[eventID]; exists {
		for _, e := range m.entries {
			if e.rec.EventID == eventID {
				cp := e.rec
				cp.Payload = append([]byte(nil), e.rec.Payload...)
				return &cp
			}
		}
	}
	m.nextID++
	rec := Record{
		ID:          m.nextID,
		Schema:      m.schema,
		EventID:     eventID,
		Topic:       topic,
		AggregateID: aggregateID,
		Payload:     append([]byte(nil), payload...),
		Status:      "pending",
		CreatedAt:   time.Now().UTC(),
	}
	m.entries = append(m.entries, &memoryEntry{rec: rec})
	m.byEvent[eventID] = struct{}{}
	cp := rec
	cp.Payload = append([]byte(nil), rec.Payload...)
	return &cp
}

// ClaimPending leases up to limit unpublished, unlocked rows.
func (m *MemoryStore) ClaimPending(limit int32, lease time.Duration) []*Record {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 {
		limit = 50
	}
	if lease <= 0 {
		lease = DefaultLeaseDuration
	}
	now := time.Now().UTC()
	var claimed []*Record
	for _, e := range m.entries {
		if int32(len(claimed)) >= limit {
			break
		}
		if e.rec.PublishedAt != nil {
			continue
		}
		if !e.lockedUntil.IsZero() && e.lockedUntil.After(now) {
			continue
		}
		e.rec.Status = "processing"
		e.rec.Attempts++
		e.lockedUntil = now.Add(lease)
		cp := e.rec
		cp.Payload = append([]byte(nil), e.rec.Payload...)
		claimed = append(claimed, &cp)
	}
	return claimed
}

// MarkPublished marks matching row IDs as published.
func (m *MemoryStore) MarkPublished(ids []int64) {
	if len(ids) == 0 {
		return
	}
	idSet := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	now := time.Now().UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.entries {
		if _, ok := idSet[e.rec.ID]; ok {
			e.rec.Status = "published"
			e.rec.LastError = ""
			e.lockedUntil = time.Time{}
			ts := now
			e.rec.PublishedAt = &ts
		}
	}
}

// MarkFailed releases the claim lease (after retryAfter) and records last_error.
func (m *MemoryStore) MarkFailed(id int64, errMsg string, retryAfter time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	for _, e := range m.entries {
		if e.rec.ID == id && e.rec.PublishedAt == nil {
			e.rec.Status = "pending"
			e.rec.LastError = errMsg
			if retryAfter > 0 {
				e.lockedUntil = now.Add(retryAfter)
			} else {
				e.lockedUntil = time.Time{}
			}
			return
		}
	}
}

// Snapshot returns a deep copy of all stored outbox records (useful in tests).
func (m *MemoryStore) Snapshot() []Record {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Record, 0, len(m.entries))
	for _, e := range m.entries {
		cp := e.rec
		cp.Payload = append([]byte(nil), e.rec.Payload...)
		out = append(out, cp)
	}
	return out
}
