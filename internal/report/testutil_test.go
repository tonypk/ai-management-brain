package report_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// mockRedisClient implements the RedisClient interface used by collector
type mockRedisClient struct {
	mu   sync.Mutex
	data map[string]string
	ttls map[string]time.Duration
}

func mockRedis() *mockRedisClient {
	return &mockRedisClient{
		data: make(map[string]string),
		ttls: make(map[string]time.Duration),
	}
}

func (m *mockRedisClient) Get(ctx context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[key]
	if !ok {
		return "", ErrNil
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
	m.ttls[key] = ttl
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

// ErrNil sentinel for "key not found"
var ErrNil = fmt.Errorf("redis: nil")
