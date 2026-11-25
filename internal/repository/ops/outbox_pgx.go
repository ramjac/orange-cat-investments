package ops

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/orange-cat-investments/oci/pkg/outbox"
)

type execDB interface {
	DBExecutor
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

func (r *pgxRepository) ClaimOutboxEvents(ctx context.Context, limit int32, lease time.Duration) ([]*outbox.Record, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	exec, ok := r.db.(execDB)
	if !ok {
		return []*outbox.Record{}, nil
	}
	return outbox.ClaimPending(ctx, exec, outbox.SchemaOps, limit, lease)
}

func (r *pgxRepository) AckOutboxEvents(ctx context.Context, publishedIDs []int64, failures []outbox.Failure) (int, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}
	exec, ok := r.db.(execDB)
	if !ok {
		return 0, nil
	}
	acked := 0
	if len(publishedIDs) > 0 {
		if err := outbox.MarkPublished(ctx, exec, outbox.SchemaOps, publishedIDs); err != nil {
			return acked, err
		}
		acked += len(publishedIDs)
	}
	for _, f := range failures {
		if err := outbox.MarkFailed(ctx, exec, outbox.SchemaOps, f.ID, f.Error, 2*time.Second); err != nil {
			return acked, err
		}
		acked++
	}
	return acked, nil
}
