package valkey

import (
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func BenchmarkPruneStaleSessions_Customer(b *testing.B) {
	srv := startMockRedisServer(&testing.T{})
	defer srv.close()

	client := redis.NewClient(&redis.Options{Addr: srv.addr})
	defer client.Close()

	store := NewValkeyCustomerSessionStore(client, nil)

	// Setup 50 tokens: 25 active, 25 stale/expired
	customerID := "cust-bench-1"
	ctx := b.Context()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Re-populate data for each iteration
		client.Del(ctx, customerUserIndexKey(customerID))
		for j := 0; j < 25; j++ {
			activeToken := fmt.Sprintf("token-active-%d", j)
			sess := &CustomerSession{
				SessionID:  activeToken,
				CustomerID: customerID,
				ExpiresAt:  time.Now().Add(1 * time.Hour),
			}
			store.SetSession(sess)
		}
		for j := 0; j < 25; j++ {
			staleToken := fmt.Sprintf("token-stale-%d", j)
			client.SAdd(ctx, customerUserIndexKey(customerID), staleToken)
		}
		b.StartTimer()

		pruned := store.PruneStaleSessions(customerID)
		if pruned != 25 {
			b.Fatalf("expected 25 pruned tokens, got %d", pruned)
		}
	}
}

func BenchmarkPruneStaleSessions_Employee(b *testing.B) {
	srv := startMockRedisServer(&testing.T{})
	defer srv.close()

	client := redis.NewClient(&redis.Options{Addr: srv.addr})
	defer client.Close()

	store := NewValkeyEmployeeSessionStore(client, nil)

	employeeID := "emp-bench-1"
	ctx := b.Context()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		client.Del(ctx, employeeUserIndexKey(employeeID))
		for j := 0; j < 25; j++ {
			activeToken := fmt.Sprintf("emp-token-active-%d", j)
			sess := &EmployeeSession{
				SessionID:  activeToken,
				EmployeeID: employeeID,
				ExpiresAt:  time.Now().Add(1 * time.Hour),
			}
			store.SetSession(sess)
		}
		for j := 0; j < 25; j++ {
			staleToken := fmt.Sprintf("emp-token-stale-%d", j)
			client.SAdd(ctx, employeeUserIndexKey(employeeID), staleToken)
		}
		b.StartTimer()

		pruned := store.PruneStaleSessions(employeeID)
		if pruned != 25 {
			b.Fatalf("expected 25 pruned tokens, got %d", pruned)
		}
	}
}
