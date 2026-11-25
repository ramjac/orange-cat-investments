package facilities

import (
	"context"
	"fmt"
	"time"

	"github.com/orange-cat-investments/oci/pkg/outbox"
)

func (r *pgxRepository) ClaimOutboxEvents(ctx context.Context, limit int32, lease time.Duration) ([]*outbox.Record, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	return outbox.ClaimPending(ctx, r.db, outbox.SchemaFacilities, limit, lease)
}

func (r *pgxRepository) AckOutboxEvents(ctx context.Context, publishedIDs []int64, failures []outbox.Failure) (int, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}
	acked := 0
	if len(publishedIDs) > 0 {
		if err := outbox.MarkPublished(ctx, r.db, outbox.SchemaFacilities, publishedIDs); err != nil {
			return acked, err
		}
		acked += len(publishedIDs)
	}
	for _, f := range failures {
		if err := outbox.MarkFailed(ctx, r.db, outbox.SchemaFacilities, f.ID, f.Error, 2*time.Second); err != nil {
			return acked, err
		}
		acked++
	}
	return acked, nil
}
