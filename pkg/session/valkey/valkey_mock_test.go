package valkey

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRedisServer is a lightweight in-memory RESP server for testing Redis clients.
type mockRedisServer struct {
	listener net.Listener
	addr     string
	mu       sync.Mutex
	strings  map[string]string
	sets     map[string]map[string]bool
	closed   bool
}

func startMockRedisServer(t *testing.T) *mockRedisServer {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	s := &mockRedisServer{
		listener: ln,
		addr:     ln.Addr().String(),
		strings:  make(map[string]string),
		sets:     make(map[string]map[string]bool),
	}

	go s.serve()
	return s
}

func (s *mockRedisServer) close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	_ = s.listener.Close()
}

func (s *mockRedisServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *mockRedisServer) handleConn(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		if line[0] == '*' {
			numArgs, _ := strconv.Atoi(line[1:])
			args := make([]string, 0, numArgs)
			for i := 0; i < numArgs; i++ {
				lenLine, err := reader.ReadString('\n')
				if err != nil {
					break
				}
				lenLine = strings.TrimSpace(lenLine)
				if len(lenLine) > 0 && lenLine[0] == '$' {
					argLen, _ := strconv.Atoi(lenLine[1:])
					if argLen >= 0 {
						buf := make([]byte, argLen)
						_, _ = io.ReadFull(reader, buf)
						_, _ = reader.ReadString('\n') // read trailing \r\n
						args = append(args, string(buf))
						continue
					}
				}
				args = append(args, "")
			}

			if len(args) == 0 {
				continue
			}

			cmd := strings.ToUpper(args[0])
			s.mu.Lock()
			switch cmd {
			case "HELLO":
				conn.Write([]byte("%7\r\n$6\r\nserver\r\n$5\r\nredis\r\n$7\r\nversion\r\n$5\r\n7.2.4\r\n$5\r\nproto\r\n:3\r\n$2\r\nid\r\n:1\r\n$4\r\nmode\r\n$10\r\nstandalone\r\n$4\r\nrole\r\n$6\r\nmaster\r\n$7\r\nmodules\r\n*0\r\n"))
			case "PING":
				conn.Write([]byte("+PONG\r\n"))
			case "SET":
				if len(args) >= 3 {
					s.strings[args[1]] = args[2]
				}
				conn.Write([]byte("+OK\r\n"))
			case "GET":
				if len(args) >= 2 {
					if val, ok := s.strings[args[1]]; ok {
						conn.Write([]byte(fmt.Sprintf("$%d\r\n%s\r\n", len(val), val)))
					} else {
						conn.Write([]byte("$-1\r\n"))
					}
				}
			case "DEL":
				count := 0
				for _, k := range args[1:] {
					if _, ok := s.strings[k]; ok {
						delete(s.strings, k)
						count++
					}
					if _, ok := s.sets[k]; ok {
						delete(s.sets, k)
						count++
					}
				}
				conn.Write([]byte(fmt.Sprintf(":%d\r\n", count)))
			case "SADD":
				if len(args) >= 3 {
					key := args[1]
					if s.sets[key] == nil {
						s.sets[key] = make(map[string]bool)
					}
					added := 0
					for _, m := range args[2:] {
						if !s.sets[key][m] {
							s.sets[key][m] = true
							added++
						}
					}
					conn.Write([]byte(fmt.Sprintf(":%d\r\n", added)))
				}
			case "SMEMBERS":
				if len(args) >= 2 {
					set := s.sets[args[1]]
					conn.Write([]byte(fmt.Sprintf("*%d\r\n", len(set))))
					for m := range set {
						conn.Write([]byte(fmt.Sprintf("$%d\r\n%s\r\n", len(m), m)))
					}
				}
			case "SREM":
				if len(args) >= 3 {
					key := args[1]
					removed := 0
					if s.sets[key] != nil {
						for _, m := range args[2:] {
							if s.sets[key][m] {
								delete(s.sets[key], m)
								removed++
							}
						}
					}
					conn.Write([]byte(fmt.Sprintf(":%d\r\n", removed)))
				}
			case "EXPIRE":
				conn.Write([]byte(":1\r\n"))
			case "EVAL":
				if len(args) >= 4 {
					key := args[3]
					prefix := ""
					if len(args) >= 5 {
						prefix = args[4]
					}
					set := s.sets[key]
					count := 0
					for m := range set {
						if _, exists := s.strings[prefix+m]; exists {
							count++
							delete(s.strings, prefix+m)
						}
					}
					delete(s.sets, key)
					conn.Write([]byte(fmt.Sprintf(":%d\r\n", count)))
				} else {
					conn.Write([]byte(":0\r\n"))
				}
			default:
				conn.Write([]byte("+OK\r\n"))
			}
			s.mu.Unlock()
		}
	}
}

func TestValkeyCustomerAndEmployeeSessionStore_WithMockValkey(t *testing.T) {
	srv := startMockRedisServer(t)
	defer srv.close()

	client := redis.NewClient(&redis.Options{Addr: srv.addr})
	defer client.Close()

	// 1. Test Customer Session Store with Valkey
	custStore := NewValkeyCustomerSessionStore(client, nil)

	custSess, err := custStore.Create("cust-202", "Diana Prince", "diana@amazon.com")
	require.NoError(t, err)
	assert.NotEmpty(t, custSess.SessionID)

	gotCust, ok := custStore.Get(custSess.SessionID)
	require.True(t, ok)
	assert.Equal(t, "cust-202", gotCust.CustomerID)
	assert.Equal(t, "Diana Prince", gotCust.Name)

	// Create 2nd session for same customer
	custSess2, err := custStore.Create("cust-202", "Diana Prince", "diana@amazon.com")
	require.NoError(t, err)

	purgedCust, err := custStore.PurgeCustomerSessions("cust-202")
	require.NoError(t, err)
	assert.Equal(t, 2, purgedCust)

	_, ok = custStore.Get(custSess.SessionID)
	assert.False(t, ok)
	_, ok = custStore.Get(custSess2.SessionID)
	assert.False(t, ok)

	// 2. Test Employee Session Store with Valkey
	empStore := NewValkeyEmployeeSessionStore(client, nil)

	empSess, err := empStore.Create("emp-505", "marcus@oci.internal", "Marcus Aurelius", "ENGINEERING", "Infrastructure")
	require.NoError(t, err)
	assert.NotEmpty(t, empSess.SessionID)

	gotEmp, ok := empStore.Get(empSess.SessionID)
	require.True(t, ok)
	assert.Equal(t, "emp-505", gotEmp.EmployeeID)
	assert.Equal(t, "engineering", gotEmp.Role)

	// Delete specific employee session
	empStore.Delete(empSess.SessionID)
	_, ok = empStore.Get(empSess.SessionID)
	assert.False(t, ok)

	// 3. Test Stale Token Pruning and Return Count Consistency (Customer)
	ctx := t.Context()
	activeSess, err := custStore.Create("cust-drift", "Alice Drift", "alice@example.com")
	require.NoError(t, err)

	// Simulate drift: inject an already-evicted token into the Redis index set
	client.SAdd(ctx, customerUserIndexKey("cust-drift"), "stale-evicted-token-1")

	// Verify PruneStaleSessions prunes the stale token while leaving the active session intact
	pruned := custStore.PruneStaleSessions("cust-drift")
	assert.Equal(t, 1, pruned)

	members, err := client.SMembers(ctx, customerUserIndexKey("cust-drift")).Result()
	require.NoError(t, err)
	assert.Contains(t, members, activeSess.SessionID)
	assert.NotContains(t, members, "stale-evicted-token-1")

	// Inject another stale token to test PurgeCustomerSessions return count consistency
	client.SAdd(ctx, customerUserIndexKey("cust-drift"), "stale-evicted-token-2")
	// Purge should return 1 (only the active session), even though SMembers had 2 tokens
	purgedCount, err := custStore.PurgeCustomerSessions("cust-drift")
	require.NoError(t, err)
	assert.Equal(t, 1, purgedCount)

	// 4. Test Expired Session Cleanup during Get()
	expiredToken := "sess-expired-manual"
	expiredData := `{"session_id":"sess-expired-manual","customer_id":"cust-exp","name":"Expired User","expires_at":"2020-01-01T00:00:00Z"}`
	client.Set(ctx, customerSessionKey(expiredToken), expiredData, 0)
	client.SAdd(ctx, customerUserIndexKey("cust-exp"), expiredToken)

	gotExp, ok := custStore.Get(expiredToken)
	assert.False(t, ok)
	assert.Nil(t, gotExp)

	// Key should be deleted and token pruned from set
	val, err := client.Get(ctx, customerSessionKey(expiredToken)).Result()
	assert.Empty(t, val)
	expMembers, _ := client.SMembers(ctx, customerUserIndexKey("cust-exp")).Result()
	assert.NotContains(t, expMembers, expiredToken)

	// 5. Test Stale Token Pruning and Purge Count Consistency (Employee)
	activeEmp, err := empStore.Create("emp-drift", "emp@example.com", "Bob Drift", "ENGINEERING", "Ops")
	require.NoError(t, err)
	assert.NotEmpty(t, activeEmp.SessionID)

	client.SAdd(ctx, employeeUserIndexKey("emp-drift"), "stale-emp-token-1")
	empPruned := empStore.PruneStaleSessions("emp-drift")
	assert.Equal(t, 1, empPruned)

	client.SAdd(ctx, employeeUserIndexKey("emp-drift"), "stale-emp-token-2")
	empPurgedCount, err := empStore.PurgeEmployeeSessions("emp-drift")
	require.NoError(t, err)
	assert.Equal(t, 1, empPurgedCount)
}
