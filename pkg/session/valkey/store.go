package valkey

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// CustomerSession represents an authenticated customer session.
type CustomerSession struct {
	SessionID  string    `json:"session_id"`
	CustomerID string    `json:"customer_id"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// EmployeeSession represents an authenticated internal employee session.
type EmployeeSession struct {
	SessionID   string    `json:"session_id"`
	EmployeeID  string    `json:"employee_id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	Role        string    `json:"role"`
	Department  string    `json:"department"`
	Permissions []string  `json:"permissions,omitempty"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// CustomerSessionStore defines the interface for managing customer sessions.
type CustomerSessionStore interface {
	Get(token string) (*CustomerSession, bool)
	Create(customerID, name, email string) (*CustomerSession, error)
	SetSession(session *CustomerSession)
	Delete(token string)
	PurgeCustomerSessions(customerID string) int
	Close() error
}

// EmployeeSessionStore defines the interface for managing employee sessions.
type EmployeeSessionStore interface {
	Get(token string) (*EmployeeSession, bool)
	Create(employeeID, email, name, role, department string) (*EmployeeSession, error)
	SetSession(session *EmployeeSession)
	Delete(token string)
	PurgeEmployeeSessions(employeeID string) int
	Close() error
}

// --- Customer Session Store: In-Memory Implementation ---

type memoryCustomerSessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*CustomerSession
}

func NewMemoryCustomerSessionStore() CustomerSessionStore {
	return &memoryCustomerSessionStore{
		sessions: make(map[string]*CustomerSession),
	}
}

func (s *memoryCustomerSessionStore) Get(token string) (*CustomerSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[token]
	if !ok || time.Now().After(sess.ExpiresAt) {
		return nil, false
	}
	cpy := *sess
	return &cpy, true
}

func (s *memoryCustomerSessionStore) Create(customerID, name, email string) (*CustomerSession, error) {
	b := make([]byte, 16)
	if _, err := crand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate random session token: %w", err)
	}
	token := "sess-" + hex.EncodeToString(b)
	sess := &CustomerSession{
		SessionID:  token,
		CustomerID: customerID,
		Email:      email,
		Name:       name,
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	s.SetSession(sess)
	return sess, nil
}

func (s *memoryCustomerSessionStore) SetSession(session *CustomerSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cpy := *session
	s.sessions[session.SessionID] = &cpy
}

func (s *memoryCustomerSessionStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func (s *memoryCustomerSessionStore) PurgeCustomerSessions(customerID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	purged := 0
	for token, sess := range s.sessions {
		if sess.CustomerID == customerID {
			delete(s.sessions, token)
			purged++
		}
	}
	return purged
}

func (s *memoryCustomerSessionStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = make(map[string]*CustomerSession)
	return nil
}

// --- Customer Session Store: Valkey Implementation ---

type valkeyCustomerSessionStore struct {
	client *redis.Client
	logger *slog.Logger
}

// NewValkeyCustomerSessionStore creates a Valkey-backed customer session store.
func NewValkeyCustomerSessionStore(client *redis.Client, logger *slog.Logger) CustomerSessionStore {
	if logger == nil {
		logger = slog.Default()
	}
	return &valkeyCustomerSessionStore{
		client: client,
		logger: logger,
	}
}

func customerSessionKey(token string) string {
	return "session:customer:" + token
}

func customerUserIndexKey(customerID string) string {
	return "user:sessions:customer:" + customerID
}

func (s *valkeyCustomerSessionStore) Get(token string) (*CustomerSession, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	val, err := s.client.Get(ctx, customerSessionKey(token)).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			s.logger.Warn("valkey error fetching customer session", "token", token, "error", err)
		}
		return nil, false
	}

	var sess CustomerSession
	if err := json.Unmarshal([]byte(val), &sess); err != nil {
		s.logger.Error("failed to deserialize customer session", "error", err)
		return nil, false
	}

	if time.Now().After(sess.ExpiresAt) {
		s.Delete(token)
		return nil, false
	}

	return &sess, true
}

func (s *valkeyCustomerSessionStore) Create(customerID, name, email string) (*CustomerSession, error) {
	b := make([]byte, 16)
	if _, err := crand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate random session token: %w", err)
	}
	token := "sess-" + hex.EncodeToString(b)
	sess := &CustomerSession{
		SessionID:  token,
		CustomerID: customerID,
		Email:      email,
		Name:       name,
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}

	s.SetSession(sess)
	return sess, nil
}

func (s *valkeyCustomerSessionStore) SetSession(session *CustomerSession) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	data, err := json.Marshal(session)
	if err != nil {
		s.logger.Error("failed to serialize customer session", "error", err)
		return
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return
	}

	pipe := s.client.Pipeline()
	pipe.Set(ctx, customerSessionKey(session.SessionID), data, ttl)
	if session.CustomerID != "" {
		pipe.SAdd(ctx, customerUserIndexKey(session.CustomerID), session.SessionID)
		pipe.Expire(ctx, customerUserIndexKey(session.CustomerID), 48*time.Hour)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		s.logger.Error("failed to persist customer session in valkey", "error", err)
	}
}

func (s *valkeyCustomerSessionStore) Delete(token string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sess, ok := s.Get(token)
	pipe := s.client.Pipeline()
	pipe.Del(ctx, customerSessionKey(token))
	if ok && sess.CustomerID != "" {
		pipe.SRem(ctx, customerUserIndexKey(sess.CustomerID), token)
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		s.logger.Warn("failed to delete customer session from valkey", "token", token, "error", err)
	}
}

func (s *valkeyCustomerSessionStore) PurgeCustomerSessions(customerID string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	indexKey := customerUserIndexKey(customerID)
	tokens, err := s.client.SMembers(ctx, indexKey).Result()
	if err != nil || len(tokens) == 0 {
		_ = s.client.Del(ctx, indexKey)
		return 0
	}

	pipe := s.client.Pipeline()
	for _, token := range tokens {
		pipe.Del(ctx, customerSessionKey(token))
	}
	pipe.Del(ctx, indexKey)
	_, _ = pipe.Exec(ctx)

	return len(tokens)
}

func (s *valkeyCustomerSessionStore) Close() error {
	return s.client.Close()
}

// --- Employee Session Store: In-Memory Implementation ---

type memoryEmployeeSessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*EmployeeSession
}

func NewMemoryEmployeeSessionStore() EmployeeSessionStore {
	return &memoryEmployeeSessionStore{
		sessions: make(map[string]*EmployeeSession),
	}
}

func (s *memoryEmployeeSessionStore) Get(token string) (*EmployeeSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[token]
	if !ok || time.Now().After(sess.ExpiresAt) {
		return nil, false
	}
	cpy := *sess
	return &cpy, true
}

func (s *memoryEmployeeSessionStore) Create(employeeID, email, name, role, department string) (*EmployeeSession, error) {
	b := make([]byte, 24)
	if _, err := crand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate random session token: %w", err)
	}
	token := "emp-sess-" + hex.EncodeToString(b)
	sess := &EmployeeSession{
		SessionID:  token,
		EmployeeID: employeeID,
		Email:      email,
		Name:       name,
		Role:       strings.ToLower(strings.TrimSpace(role)),
		Department: department,
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	s.SetSession(sess)
	return sess, nil
}

func (s *memoryEmployeeSessionStore) SetSession(session *EmployeeSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cpy := *session
	s.sessions[session.SessionID] = &cpy
}

func (s *memoryEmployeeSessionStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func (s *memoryEmployeeSessionStore) PurgeEmployeeSessions(employeeID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	purged := 0
	for token, sess := range s.sessions {
		if sess.EmployeeID == employeeID {
			delete(s.sessions, token)
			purged++
		}
	}
	return purged
}

func (s *memoryEmployeeSessionStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = make(map[string]*EmployeeSession)
	return nil
}

// --- Employee Session Store: Valkey Implementation ---

type valkeyEmployeeSessionStore struct {
	client *redis.Client
	logger *slog.Logger
}

// NewValkeyEmployeeSessionStore creates a Valkey-backed employee session store.
func NewValkeyEmployeeSessionStore(client *redis.Client, logger *slog.Logger) EmployeeSessionStore {
	if logger == nil {
		logger = slog.Default()
	}
	return &valkeyEmployeeSessionStore{
		client: client,
		logger: logger,
	}
}

func employeeSessionKey(token string) string {
	return "session:employee:" + token
}

func employeeUserIndexKey(employeeID string) string {
	return "user:sessions:employee:" + employeeID
}

func (s *valkeyEmployeeSessionStore) Get(token string) (*EmployeeSession, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	val, err := s.client.Get(ctx, employeeSessionKey(token)).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			s.logger.Warn("valkey error fetching employee session", "token", token, "error", err)
		}
		return nil, false
	}

	var sess EmployeeSession
	if err := json.Unmarshal([]byte(val), &sess); err != nil {
		s.logger.Error("failed to deserialize employee session", "error", err)
		return nil, false
	}

	if time.Now().After(sess.ExpiresAt) {
		s.Delete(token)
		return nil, false
	}

	return &sess, true
}

func (s *valkeyEmployeeSessionStore) Create(employeeID, email, name, role, department string) (*EmployeeSession, error) {
	b := make([]byte, 24)
	if _, err := crand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate random session token: %w", err)
	}
	token := "emp-sess-" + hex.EncodeToString(b)
	sess := &EmployeeSession{
		SessionID:  token,
		EmployeeID: employeeID,
		Email:      email,
		Name:       name,
		Role:       strings.ToLower(strings.TrimSpace(role)),
		Department: department,
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}

	s.SetSession(sess)
	return sess, nil
}

func (s *valkeyEmployeeSessionStore) SetSession(session *EmployeeSession) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	data, err := json.Marshal(session)
	if err != nil {
		s.logger.Error("failed to serialize employee session", "error", err)
		return
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return
	}

	pipe := s.client.Pipeline()
	pipe.Set(ctx, employeeSessionKey(session.SessionID), data, ttl)
	if session.EmployeeID != "" {
		pipe.SAdd(ctx, employeeUserIndexKey(session.EmployeeID), session.SessionID)
		pipe.Expire(ctx, employeeUserIndexKey(session.EmployeeID), 48*time.Hour)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		s.logger.Error("failed to persist employee session in valkey", "error", err)
	}
}

func (s *valkeyEmployeeSessionStore) Delete(token string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sess, ok := s.Get(token)
	pipe := s.client.Pipeline()
	pipe.Del(ctx, employeeSessionKey(token))
	if ok && sess.EmployeeID != "" {
		pipe.SRem(ctx, employeeUserIndexKey(sess.EmployeeID), token)
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		s.logger.Warn("failed to delete employee session from valkey", "token", token, "error", err)
	}
}

func (s *valkeyEmployeeSessionStore) PurgeEmployeeSessions(employeeID string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	indexKey := employeeUserIndexKey(employeeID)
	tokens, err := s.client.SMembers(ctx, indexKey).Result()
	if err != nil || len(tokens) == 0 {
		_ = s.client.Del(ctx, indexKey)
		return 0
	}

	pipe := s.client.Pipeline()
	for _, token := range tokens {
		pipe.Del(ctx, employeeSessionKey(token))
	}
	pipe.Del(ctx, indexKey)
	_, _ = pipe.Exec(ctx)

	return len(tokens)
}

func (s *valkeyEmployeeSessionStore) Close() error {
	return s.client.Close()
}

// --- Factory Helpers with Fallback ---

// NewCustomerSessionStore initializes a CustomerSessionStore. If addr is empty or connection fails,
// it gracefully returns an in-memory session store.
func NewCustomerSessionStore(addr, password string, db int, logger *slog.Logger) CustomerSessionStore {
	if logger == nil {
		logger = slog.Default()
	}

	if addr == "" {
		logger.Info("VALKEY_ADDR not specified, using in-memory customer session store")
		return NewMemoryCustomerSessionStore()
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Warn("failed to ping Valkey at "+addr+", falling back to in-memory customer session store", "error", err)
		_ = client.Close()
		return NewMemoryCustomerSessionStore()
	}

	logger.Info("connected Customer BFF to distributed Valkey session store", "addr", addr)
	return NewValkeyCustomerSessionStore(client, logger)
}

// NewEmployeeSessionStore initializes an EmployeeSessionStore. If addr is empty or connection fails,
// it gracefully returns an in-memory session store.
func NewEmployeeSessionStore(addr, password string, db int, logger *slog.Logger) EmployeeSessionStore {
	if logger == nil {
		logger = slog.Default()
	}

	if addr == "" {
		logger.Info("VALKEY_ADDR not specified, using in-memory employee session store")
		return NewMemoryEmployeeSessionStore()
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Warn("failed to ping Valkey at "+addr+", falling back to in-memory employee session store", "error", err)
		_ = client.Close()
		return NewMemoryEmployeeSessionStore()
	}

	logger.Info("connected Employee BFF to distributed Valkey session store", "addr", addr)
	return NewEmployeeSessionStoreWithClient(client, logger)
}

// NewEmployeeSessionStoreWithClient creates an EmployeeSessionStore with an existing redis client.
func NewEmployeeSessionStoreWithClient(client *redis.Client, logger *slog.Logger) EmployeeSessionStore {
	return NewValkeyEmployeeSessionStore(client, logger)
}
