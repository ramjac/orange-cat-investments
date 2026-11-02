package valkey

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func BenchmarkPruneStaleSessions(b *testing.B) {
	srv := startMockRedisServer(&testing.T{})
	defer srv.close()

	client := redis.NewClient(&redis.Options{Addr: srv.addr})
	defer client.Close()

	empStore := NewValkeyEmployeeSessionStore(client, nil)
	ctx := context.Background()
	employeeID := "emp-bench-1"
	indexKey := employeeUserIndexKey(employeeID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Setup N sessions: half valid, half stale
		const totalSessions = 50
		for j := 0; j < totalSessions; j++ {
			token := fmt.Sprintf("bench-token-%d-%d", i, j)
			client.SAdd(ctx, indexKey, token)
			if j%2 == 0 {
				// Valid session
				sessData := fmt.Sprintf(`{"session_id":"%s","employee_id":"%s","email":"bench@oci.internal","name":"Bench User","role":"engineering","department":"Core","expires_at":"%s"}`,
					token, employeeID, time.Now().Add(1*time.Hour).Format(time.RFC3339Nano))
				client.Set(ctx, employeeSessionKey(token), sessData, 1*time.Hour)
			}
			// Odd sessions have no key in Redis, so they are stale
		}
		b.StartTimer()

		pruned := empStore.PruneStaleSessions(employeeID)
		if pruned != totalSessions/2 {
			b.Fatalf("expected %d pruned sessions, got %d", totalSessions/2, pruned)
		}

		b.StopTimer()
		// Clean up index key for next iteration
		client.Del(ctx, indexKey)
		for j := 0; j < totalSessions; j += 2 {
			token := fmt.Sprintf("bench-token-%d-%d", i, j)
			client.Del(ctx, employeeSessionKey(token))
		}
		b.StartTimer()
	}
}
