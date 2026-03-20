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

func TestScheduler_MissedJob_RecentIsNotMissed(t *testing.T) {
	cb := &mockCallbacks{}
	redis := mockRedis()
	// Set last_run to 30 minutes ago (within 2h threshold)
	redis.Set(context.Background(), "scheduler:last_run:remind",
		time.Now().Add(-30*time.Minute).Format(time.RFC3339), 0)

	s, _ := scheduler.New("Asia/Singapore", redis, cb)
	s.CheckMissedJobs(context.Background())

	if cb.remindCalled {
		t.Error("remind should NOT have been called — last run was recent")
	}
}

func TestScheduler_MissedJob_NoLastRun_Skips(t *testing.T) {
	cb := &mockCallbacks{}
	redis := mockRedis()
	// No last_run keys → first run, should skip catch-up

	s, _ := scheduler.New("Asia/Singapore", redis, cb)
	s.CheckMissedJobs(context.Background())

	if cb.remindCalled || cb.chaseCalled || cb.summaryCalled {
		t.Error("should not call any callback when no last_run exists")
	}
}

func TestScheduler_MissedJob_InvalidTimestamp_Skips(t *testing.T) {
	cb := &mockCallbacks{}
	redis := mockRedis()
	// Set invalid timestamp
	redis.Set(context.Background(), "scheduler:last_run:remind", "not-a-timestamp", 0)

	s, _ := scheduler.New("Asia/Singapore", redis, cb)
	s.CheckMissedJobs(context.Background())

	if cb.remindCalled {
		t.Error("remind should NOT have been called — invalid timestamp")
	}
}

func TestScheduler_MissedJob_MultipleMissed(t *testing.T) {
	cb := &mockCallbacks{}
	redis := mockRedis()
	// All three jobs missed
	for _, name := range []string{"remind", "chase", "summary"} {
		redis.Set(context.Background(), "scheduler:last_run:"+name,
			time.Now().Add(-5*time.Hour).Format(time.RFC3339), 0)
	}

	s, _ := scheduler.New("Asia/Singapore", redis, cb)
	s.CheckMissedJobs(context.Background())

	if !cb.remindCalled {
		t.Error("remind should have been called")
	}
	if !cb.chaseCalled {
		t.Error("chase should have been called")
	}
	if !cb.summaryCalled {
		t.Error("summary should have been called")
	}
}

func TestScheduler_InvalidTimezone(t *testing.T) {
	cb := &mockCallbacks{}
	_, err := scheduler.New("Invalid/Timezone", mockRedis(), cb)
	if err == nil {
		t.Error("expected error for invalid timezone")
	}
}

func TestScheduler_AddJob(t *testing.T) {
	cb := &mockCallbacks{}
	s, err := scheduler.New("Asia/Singapore", mockRedis(), cb)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if s.JobCount() != 3 {
		t.Fatalf("expected 3 initial jobs, got %d", s.JobCount())
	}

	called := false
	err = s.AddJob("custom", "0 12 * * *", func(ctx context.Context) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("AddJob: %v", err)
	}
	if s.JobCount() != 4 {
		t.Errorf("expected 4 jobs after AddJob, got %d", s.JobCount())
	}
	_ = called
}

func TestScheduler_AddJob_InvalidCron(t *testing.T) {
	cb := &mockCallbacks{}
	s, _ := scheduler.New("Asia/Singapore", mockRedis(), cb)

	err := s.AddJob("bad", "not a cron", func(ctx context.Context) error { return nil })
	if err == nil {
		t.Error("expected error for invalid cron expression")
	}
}

func TestScheduler_StartAndStop(t *testing.T) {
	cb := &mockCallbacks{}
	s, err := scheduler.New("Asia/Singapore", mockRedis(), cb)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.Start(ctx)
	if err := s.Stop(); err != nil {
		t.Errorf("stop: %v", err)
	}
}

func TestScheduler_MissedJob_UpdatesLastRun(t *testing.T) {
	cb := &mockCallbacks{}
	redis := mockRedis()
	key := "scheduler:last_run:remind"
	redis.Set(context.Background(), key,
		time.Now().Add(-3*time.Hour).Format(time.RFC3339), 0)

	s, _ := scheduler.New("Asia/Singapore", redis, cb)
	s.CheckMissedJobs(context.Background())

	// After catch-up, last_run should be updated to now-ish
	raw, err := redis.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("get last_run: %v", err)
	}
	lastRun, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t.Fatalf("parse last_run: %v", err)
	}
	if time.Since(lastRun) > 5*time.Second {
		t.Error("last_run should have been updated to approximately now")
	}
}
