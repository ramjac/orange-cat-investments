package valkey

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRedisServer struct {
	listener net.Listener
	addr     string
	mu       sync.Mutex
	strings  map[string]string
	zsets    map[string]map[string]float64
	closed   bool
}

func startMockRedisServer(t *testing.T) *mockRedisServer {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	s := &mockRedisServer{
		listener: ln,
		addr:     ln.Addr().String(),
		strings:  make(map[string]string),
		zsets:    make(map[string]map[string]float64),
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
					if _, ok := s.zsets[k]; ok {
						delete(s.zsets, k)
						count++
					}
				}
				conn.Write([]byte(fmt.Sprintf(":%d\r\n", count)))
			case "ZREMRANGEBYSCORE":
				if len(args) >= 4 {
					key := args[1]
					maxScore, _ := strconv.ParseFloat(args[3], 64)
					removed := 0
					if s.zsets[key] != nil {
						for m, score := range s.zsets[key] {
							if score <= maxScore {
								delete(s.zsets[key], m)
								removed++
							}
						}
					}
					conn.Write([]byte(fmt.Sprintf(":%d\r\n", removed)))
				}
			case "ZCARD":
				if len(args) >= 2 {
					key := args[1]
					count := len(s.zsets[key])
					conn.Write([]byte(fmt.Sprintf(":%d\r\n", count)))
				}
			case "ZADD":
				if len(args) >= 4 {
					key := args[1]
					score, _ := strconv.ParseFloat(args[2], 64)
					member := args[3]
					if s.zsets[key] == nil {
						s.zsets[key] = make(map[string]float64)
					}
					s.zsets[key][member] = score
					conn.Write([]byte(":1\r\n"))
				}
			case "ZRANGE":
				conn.Write([]byte("*0\r\n"))
			case "EVALSHA":
				conn.Write([]byte("-NOSCRIPT No matching script\r\n"))
			case "EVAL":
				if len(args) >= 8 {
					key := args[3]
					windowStart, _ := strconv.ParseFloat(args[4], 64)
					nowNano, _ := strconv.ParseFloat(args[5], 64)
					member := args[6]
					limit, _ := strconv.Atoi(args[7])

					if s.zsets[key] == nil {
						s.zsets[key] = make(map[string]float64)
					}
					for m, score := range s.zsets[key] {
						if score <= windowStart {
							delete(s.zsets[key], m)
						}
					}
					count := len(s.zsets[key])
					if count >= limit {
						var oldestScore float64
						first := true
						for _, score := range s.zsets[key] {
							if first || score < oldestScore {
								oldestScore = score
								first = false
							}
						}
						scoreStr := fmt.Sprintf("%.0f", oldestScore)
						conn.Write([]byte(fmt.Sprintf("*3\r\n:0\r\n:%d\r\n$%d\r\n%s\r\n", count, len(scoreStr), scoreStr)))
					} else {
						s.zsets[key][member] = nowNano
						conn.Write([]byte(fmt.Sprintf("*3\r\n:1\r\n:%d\r\n$1\r\n0\r\n", count+1)))
					}
				} else {
					conn.Write([]byte("+OK\r\n"))
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

func TestValkeyCacheAndRateLimiter_WithMock(t *testing.T) {
	srv := startMockRedisServer(t)
	defer srv.close()

	client := redis.NewClient(&redis.Options{Addr: srv.addr})
	defer client.Close()

	ctx := context.Background()

	// 1. Test ValkeyCache
	cache := NewValkeyCache(client, nil)

	type Data struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	err := cache.Set(ctx, "test:key", Data{Name: "OCI", Value: 42}, time.Minute)
	require.NoError(t, err)

	var retrieved Data
	ok, err := cache.Get(ctx, "test:key", &retrieved)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "OCI", retrieved.Name)
	assert.Equal(t, 42, retrieved.Value)

	err = cache.Delete(ctx, "test:key")
	require.NoError(t, err)

	ok, err = cache.Get(ctx, "test:key", &retrieved)
	require.NoError(t, err)
	assert.False(t, ok)

	// 2. Test Valkey RateLimiter
	limiter := NewRateLimiter(client, nil)
	key := "valkey-user:cust-1"
	limit := 2
	window := time.Second

	allowed, rem, _, err := limiter.Allow(ctx, key, limit, window)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 1, rem)

	allowed, rem, _, err = limiter.Allow(ctx, key, limit, window)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 0, rem)

	allowed, rem, _, err = limiter.Allow(ctx, key, limit, window)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, rem)
}
