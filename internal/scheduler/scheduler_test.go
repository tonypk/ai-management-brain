package scheduler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/tonypk/ai-management-brain/internal/scheduler"
)

// mockRedisClient for scheduler tests
type mockRedisClient struct {
	mu   sync.Mutex
	data map[string]string
}

func mockRedis() *mockRedisClient {
	return &mockRedisClient{data: make(map[string]string)}
}

var errNil = fmt.Errorf("redis: nil")

func (m *mockRedisClient) Get(ctx context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[key]
	if !ok {
		return "", errNil
	}
	return v, nil
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch v := value.(type) {
	case string:
		m.data[key] = v
	default:
		b, _ := json.Marshal(value)
		m.data[key] = string(b)
	}
	return nil
}

func (m *mockRedisClient) Del(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}

type mockCallbacks struct {
	remindCalled  bool
	chaseCalled   bool
	summaryCalled bool
}

func (m *mockCallbacks) Remind(ctx context.Context) error { m.remindCalled = true; return nil }
func (m *mockCallbacks) Chase(ctx context.Context) error  { m.chaseCalled = true; return nil }
func (m *mockCallbacks) Summary(ctx context.Context) error { m.summaryCalled = true; return nil }

func TestScheduler_RegistersThreeJobs(t *testing.T) {
	cb := &mockCallbacks{}
	s, err := scheduler.New("Asia/Singapore", mockRedis(), cb)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if s.JobCount() != 3 {
		t.Errorf("expected 3 jobs, got %d", s.JobCount())
	}
}

func TestScheduler_MissedJobCatchUp(t *testing.T) {
	cb := &mockCallbacks{}
	redis := mockRedis()
	// Set last_run to 3 hours ago (beyond 2h threshold)
	redis.Set(context.Background(), "scheduler:last_run:remind",
		time.Now().Add(-3*time.Hour).Format(time.RFC3339), 0)

	s, _ := scheduler.New("Asia/Singapore", redis, cb)
	s.CheckMissedJobs(context.Background())

	if !cb.remindCalled {
		t.Error("remind should have been called as catch-up")
	}
}
