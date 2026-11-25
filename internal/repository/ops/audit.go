package ops

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const AuditGenesisHash = "GENESIS"

// AuditLog represents an immutable cryptographic hash-chained audit entry in ops.audit_logs.
type AuditLog struct {
	ID           int64     `json:"id"`
	ActorID      string    `json:"actor_id"`
	ActorRole    string    `json:"actor_role"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	MetadataJSON string    `json:"metadata_json"`
	CreatedAt    time.Time `json:"created_at"`
	CanonicalTS  string    `json:"-"`
	PreviousHash string    `json:"previous_hash"`
	RecordHash   string    `json:"record_hash"`
}

// AuditChainVerification reports the result of walking and validating the full ops.audit_logs hash chain.
type AuditChainVerification struct {
	Valid           bool   `json:"valid"`
	VerifiedRecords int64  `json:"verified_records"`
	BrokenAtID      int64  `json:"broken_at_id,omitempty"`
	ExpectedHash    string `json:"expected_hash,omitempty"`
	ActualHash      string `json:"actual_hash,omitempty"`
	LatestHash      string `json:"latest_hash,omitempty"`
	Reason          string `json:"reason,omitempty"`
}

// FormatCanonicalAuditTimestamp produces the exact UTC microsecond format used by
// ops.compute_audit_log_hash() in PostgreSQL: YYYY-MM-DD"T"HH24:MI:SS.US"Z".
func FormatCanonicalAuditTimestamp(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000000Z")
}

// ComputeAuditHash computes SHA-256 hex digest matching the PostgreSQL BEFORE INSERT trigger.
func ComputeAuditHash(previousHash, actorID, actorRole, action, resourceType, resourceID, metadataText, canonicalTS string) string {
	payload := strings.Join([]string{
		previousHash,
		actorID,
		actorRole,
		action,
		resourceType,
		resourceID,
		metadataText,
		canonicalTS,
	}, "|")
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func normalizeAuditMetadata(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "{}"
	}
	var js any
	if err := json.Unmarshal([]byte(trimmed), &js); err != nil {
		b, _ := json.Marshal(map[string]string{"raw": trimmed})
		return string(b)
	}
	return trimmed
}

const appendAuditLogSQL = `
	INSERT INTO ops.audit_logs (
		actor_id, actor_role, action, resource_type, resource_id, metadata_json
	) VALUES (
		$1, $2, $3, $4, $5, $6::jsonb
	)
	RETURNING id, actor_id, actor_role, action, resource_type, resource_id,
	          metadata_json::text,
	          to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
	          created_at, previous_hash, record_hash`

const verifyAuditChainSQL = `
	SELECT id, actor_id, actor_role, action, resource_type, resource_id,
	       metadata_json::text,
	       to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
	       created_at, previous_hash, record_hash
	FROM ops.audit_logs
	ORDER BY id ASC`

const listAuditLogsSQL = `
	SELECT id, actor_id, actor_role, action, resource_type, resource_id,
	       metadata_json::text,
	       to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
	       created_at, previous_hash, record_hash
	FROM ops.audit_logs
	WHERE ($1 = '' OR actor_id = $1)
	  AND ($2 = '' OR action = $2)
	ORDER BY id DESC
	LIMIT $3 OFFSET $4`

func scanAuditLog(row interface{ Scan(dest ...any) error }) (*AuditLog, error) {
	var rec AuditLog
	if err := row.Scan(
		&rec.ID,
		&rec.ActorID,
		&rec.ActorRole,
		&rec.Action,
		&rec.ResourceType,
		&rec.ResourceID,
		&rec.MetadataJSON,
		&rec.CanonicalTS,
		&rec.CreatedAt,
		&rec.PreviousHash,
		&rec.RecordHash,
	); err != nil {
		return nil, err
	}
	return &rec, nil
}

func verifyAuditRecords(records []*AuditLog) *AuditChainVerification {
	expectedPrev := AuditGenesisHash
	var verified int64
	var latestHash string
	for _, rec := range records {
		if rec.PreviousHash != expectedPrev {
			return &AuditChainVerification{
				Valid:           false,
				VerifiedRecords: verified,
				BrokenAtID:      rec.ID,
				ExpectedHash:    expectedPrev,
				ActualHash:      rec.PreviousHash,
				LatestHash:      latestHash,
				Reason:          fmt.Sprintf("broken previous_hash link at audit record id=%d", rec.ID),
			}
		}
		ts := rec.CanonicalTS
		if ts == "" {
			ts = FormatCanonicalAuditTimestamp(rec.CreatedAt)
		}
		expectedRecordHash := ComputeAuditHash(
			rec.PreviousHash,
			rec.ActorID,
			rec.ActorRole,
			rec.Action,
			rec.ResourceType,
			rec.ResourceID,
			rec.MetadataJSON,
			ts,
		)
		if rec.RecordHash != expectedRecordHash {
			return &AuditChainVerification{
				Valid:           false,
				VerifiedRecords: verified,
				BrokenAtID:      rec.ID,
				ExpectedHash:    expectedRecordHash,
				ActualHash:      rec.RecordHash,
				LatestHash:      latestHash,
				Reason:          fmt.Sprintf("record_hash mismatch (tampering detected) at audit record id=%d", rec.ID),
			}
		}
		expectedPrev = rec.RecordHash
		latestHash = rec.RecordHash
		verified++
	}
	return &AuditChainVerification{
		Valid:           true,
		VerifiedRecords: verified,
		LatestHash:      latestHash,
	}
}

func (r *pgxRepository) AppendAuditLog(ctx context.Context, entry *AuditLog) (*AuditLog, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	if entry == nil || strings.TrimSpace(entry.ActorID) == "" || strings.TrimSpace(entry.Action) == "" {
		return nil, fmt.Errorf("actor_id and action are required for audit logs")
	}
	meta := normalizeAuditMetadata(entry.MetadataJSON)
	row := r.db.QueryRow(ctx, appendAuditLogSQL,
		entry.ActorID,
		entry.ActorRole,
		entry.Action,
		entry.ResourceType,
		entry.ResourceID,
		meta,
	)
	return scanAuditLog(row)
}

func (r *pgxRepository) VerifyAuditChain(ctx context.Context) (*AuditChainVerification, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	rows, err := r.db.Query(ctx, verifyAuditChainSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*AuditLog
	for rows.Next() {
		rec, err := scanAuditLog(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return verifyAuditRecords(records), nil
}

func (r *pgxRepository) ListAuditLogs(ctx context.Context, actorID, action string, limit, offset int32) ([]*AuditLog, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.db.Query(ctx, listAuditLogsSQL, actorID, action, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := []*AuditLog{}
	for rows.Next() {
		rec, err := scanAuditLog(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}
