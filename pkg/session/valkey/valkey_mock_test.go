package valkey

import (
	"bufio"
	"fmt"
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
				_, _ = reader.ReadString('\n') // read $<len>
				argLine, _ := reader.ReadString('\n')
				args = append(args, strings.TrimRight(argLine, "\r\n"))
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

	purgedCust := custStore.PurgeCustomerSessions("cust-202")
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
}
