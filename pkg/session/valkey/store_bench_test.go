package valkey

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func BenchmarkPruneStaleSessions_Customer(b *testing.B) {
	srv := startMockRedisServer(nil)
	defer srv.close()

	client := redis.NewClient(&redis.Options{Addr: srv.addr})
	defer client.Close()

	store := NewValkeyCustomerSessionStore(client, nil)
	ctx := context.Background()

	const totalTokens = 50
	const validTokens = 25

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		customerID := fmt.Sprintf("cust-bench-%d", i)
		indexKey := customerUserIndexKey(customerID)

		// Populate 25 active sessions and 25 stale sessions in the index set
		for j := 0; j < totalTokens; j++ {
			token := fmt.Sprintf("token-%d-%d", i, j)
			client.SAdd(ctx, indexKey, token)
			if j < validTokens {
				data := fmt.Sprintf(`{"session_id":"%s","customer_id":"%s","expires_at":"%s"}`,
					token, customerID, time.Now().Add(1*time.Hour).Format(time.RFC3339))
				client.Set(ctx, customerSessionKey(token), data, 0)
			}
		}
		b.StartTimer()

		pruned := store.PruneStaleSessions(customerID)
		if pruned != totalTokens-validTokens {
			b.Fatalf("expected %d pruned tokens, got %d", totalTokens-validTokens, pruned)
		}
	}
}

func BenchmarkPruneStaleSessions_Employee(b *testing.B) {
	srv := startMockRedisServer(nil)
	defer srv.close()

	client := redis.NewClient(&redis.Options{Addr: srv.addr})
	defer client.Close()

	store := NewValkeyEmployeeSessionStore(client, nil)
	ctx := context.Background()

	const totalTokens = 50
	const validTokens = 25

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		employeeID := fmt.Sprintf("emp-bench-%d", i)
		indexKey := employeeUserIndexKey(employeeID)

		// Populate 25 active sessions and 25 stale sessions in the index set
		for j := 0; j < totalTokens; j++ {
			token := fmt.Sprintf("emp-token-%d-%d", i, j)
			client.SAdd(ctx, indexKey, token)
			if j < validTokens {
				data := fmt.Sprintf(`{"session_id":"%s","employee_id":"%s","expires_at":"%s"}`,
					token, employeeID, time.Now().Add(1*time.Hour).Format(time.RFC3339))
				client.Set(ctx, employeeSessionKey(token), data, 0)
			}
		}
		b.StartTimer()

		pruned := store.PruneStaleSessions(employeeID)
		if pruned != totalTokens-validTokens {
			b.Fatalf("expected %d pruned tokens, got %d", totalTokens-validTokens, pruned)
		}
	}
}
