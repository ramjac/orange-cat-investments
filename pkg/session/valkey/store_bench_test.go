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
	custStore := NewValkeyCustomerSessionStore(client, nil)

	ctx := context.Background()

	setupEmployeeBench := func(numTokens int) string {
		empID := fmt.Sprintf("emp-bench-%d", numTokens)
		indexKey := employeeUserIndexKey(empID)
		client.Del(ctx, indexKey)

		for i := 0; i < numTokens; i++ {
			token := fmt.Sprintf("token-%d", i)
			client.SAdd(ctx, indexKey, token)
			if i%2 == 0 {
				sess := EmployeeSession{
					SessionID:  token,
					EmployeeID: empID,
					ExpiresAt:  time.Now().Add(1 * time.Hour),
				}
				data := fmt.Sprintf(`{"session_id":"%s","employee_id":"%s","expires_at":"%s"}`, token, empID, sess.ExpiresAt.Format(time.RFC3339))
				client.Set(ctx, employeeSessionKey(token), data, 0)
			}
		}
		return empID
	}

	setupCustomerBench := func(numTokens int) string {
		custID := fmt.Sprintf("cust-bench-%d", numTokens)
		indexKey := customerUserIndexKey(custID)
		client.Del(ctx, indexKey)

		for i := 0; i < numTokens; i++ {
			token := fmt.Sprintf("token-%d", i)
			client.SAdd(ctx, indexKey, token)
			if i%2 == 0 {
				sess := CustomerSession{
					SessionID:  token,
					CustomerID: custID,
					ExpiresAt:  time.Now().Add(1 * time.Hour),
				}
				data := fmt.Sprintf(`{"session_id":"%s","customer_id":"%s","expires_at":"%s"}`, token, custID, sess.ExpiresAt.Format(time.RFC3339))
				client.Set(ctx, customerSessionKey(token), data, 0)
			}
		}
		return custID
	}

	for _, size := range []int{10, 50, 100} {
		empID := setupEmployeeBench(size)
		b.Run(fmt.Sprintf("Employee_PruneStaleSessions_Tokens_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = empStore.PruneStaleSessions(empID)
			}
		})

		custID := setupCustomerBench(size)
		b.Run(fmt.Sprintf("Customer_PruneStaleSessions_Tokens_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = custStore.PruneStaleSessions(custID)
			}
		})
	}
}
