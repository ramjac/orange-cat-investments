package ops

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// liveDSN returns the PostgreSQL DSN for live chain tests, or "" when unset.
func liveDSN() string {
	if dsn := os.Getenv("OCI_TEST_PG_DSN"); dsn != "" {
		return dsn
	}
	return os.Getenv("DATABASE_URL")
}

// TestAuditChainConcurrentAppends exercises the tamper-evident hash chain
// under concurrent appends against a real PostgreSQL. It is skipped unless
// OCI_TEST_PG_DSN (or DATABASE_URL) points at a live database where the test
// user may create schemas.
//
// The fixture is the real Phase 10 (7.41) migration, schema-substituted onto
// a scratch schema, so the test always validates the shipped SQL: with
// serialized id allocation (the sequence is drawn inside the hash trigger
// after the advisory chain lock is held), N concurrent appends must produce
// a chain whose ascending-id order matches the hash-chain order. A regression
// to a plain BIGSERIAL column default would let concurrent inserts draw ids
// before the lock, acquire the lock in reverse id order, and break the
// ascending-id verification this test performs.
func TestAuditChainConcurrentAppends(t *testing.T) {
	dsn := liveDSN()
	if dsn == "" {
		t.Skip("set OCI_TEST_PG_DSN to run the live PostgreSQL concurrent append test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()
	require.NoError(t, pool.Ping(ctx))

	// Load the shipped migration and re-target it at a private schema.
	rel := filepath.Join("..", "..", "..", "scripts", "migrations", "004-tamper-evident-audit-log.sql")
	raw, err := os.ReadFile(rel)
	require.NoError(t, err, "read %s", rel)
	schema := fmt.Sprintf("auditchaintest_%d", time.Now().UnixNano())
	require.True(t, strings.Contains(string(raw), "ops.audit_logs"), "migration must define ops.audit_logs")
	fixture := strings.ReplaceAll(string(raw), "ops.", schema+".")
	t.Cleanup(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer ccancel()
		_, _ = pool.Exec(cctx, fmt.Sprintf(`DROP SCHEMA IF EXISTS %s CASCADE`, schema))
	})

	_, err = pool.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA %s`, schema))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, fixture)
	require.NoError(t, err, "apply audit chain migration fixture")

	const workers = 8
	const perWorker = 25
	total := workers * perWorker

	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	for w := 0; w < workers; w++ {
		w := w
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				tx, err := pool.Begin(ctx)
				if err != nil {
					errCh <- err
					return
				}
				_, err = tx.Exec(ctx, fmt.Sprintf(`
					INSERT INTO %s.audit_logs (actor_id, actor_role, action, resource_type, resource_id, metadata_json)
					VALUES ('concurrency-test', 'qa', 'audit.chain.probe', 'audit', $1, '{}'::jsonb)`, schema),
					fmt.Sprintf("probe-%d-%d", w, i))
				if err != nil {
					_ = tx.Rollback(ctx)
					errCh <- err
					return
				}
				if err := tx.Commit(ctx); err != nil {
					errCh <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent append failed: %v", err)
	}

	// Walk the chain exactly like the production verifier does.
	rows, err := pool.Query(ctx, fmt.Sprintf(`
		SELECT id, actor_id, actor_role, action, resource_type, resource_id,
		       metadata_json::text,
		       to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
		       created_at, previous_hash, record_hash
		FROM %s.audit_logs
		ORDER BY id ASC`, schema))
	require.NoError(t, err)
	var records []*AuditLog
	for rows.Next() {
		var rec AuditLog
		require.NoError(t, rows.Scan(
			&rec.ID, &rec.ActorID, &rec.ActorRole, &rec.Action, &rec.ResourceType,
			&rec.ResourceID, &rec.MetadataJSON, &rec.CanonicalTS, &rec.CreatedAt,
			&rec.PreviousHash, &rec.RecordHash,
		))
		records = append(records, &rec)
	}
	require.NoError(t, rows.Err())

	require.Len(t, records, total)
	res := verifyAuditRecords(records)
	require.True(t, res.Valid, "hash chain must verify after concurrent appends: %+v", res)
	require.Equal(t, int64(total), res.VerifiedRecords)
}
