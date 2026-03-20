package api

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// Metrics collects HTTP request metrics in Prometheus exposition format.
type Metrics struct {
	requestsTotal   sync.Map // key: "method:path:status" → *int64
	requestDuration sync.Map // key: "method:path" → *durationBucket
	activeRequests  atomic.Int64
}

type durationBucket struct {
	sum   atomic.Int64 // microseconds
	count atomic.Int64
}

// NewMetrics creates a new metrics collector.
func NewMetrics() *Metrics {
	return &Metrics{}
}

// Middleware records request count and duration.
func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		m.activeRequests.Add(1)
		start := time.Now()

		c.Next()

		m.activeRequests.Add(-1)
		elapsed := time.Since(start)

		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}
		method := c.Request.Method
		status := c.Writer.Status()

		// Increment request counter
		counterKey := fmt.Sprintf("%s:%s:%d", method, path, status)
		val, _ := m.requestsTotal.LoadOrStore(counterKey, new(int64))
		atomic.AddInt64(val.(*int64), 1)

		// Record duration
		durKey := fmt.Sprintf("%s:%s", method, path)
		bucket, _ := m.requestDuration.LoadOrStore(durKey, &durationBucket{})
		b := bucket.(*durationBucket)
		b.sum.Add(elapsed.Microseconds())
		b.count.Add(1)
	}
}

// Handler serves Prometheus-compatible /metrics endpoint.
func (m *Metrics) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var sb strings.Builder

		sb.WriteString("# HELP brain_http_requests_total Total HTTP requests.\n")
		sb.WriteString("# TYPE brain_http_requests_total counter\n")

		var counterLines []string
		m.requestsTotal.Range(func(key, value any) bool {
			k := key.(string)
			parts := strings.SplitN(k, ":", 3)
			if len(parts) == 3 {
				count := atomic.LoadInt64(value.(*int64))
				counterLines = append(counterLines,
					fmt.Sprintf(`brain_http_requests_total{method="%s",path="%s",status="%s"} %d`,
						parts[0], parts[1], parts[2], count))
			}
			return true
		})
		sort.Strings(counterLines)
		for _, line := range counterLines {
			sb.WriteString(line)
			sb.WriteByte('\n')
		}

		sb.WriteString("\n# HELP brain_http_request_duration_seconds HTTP request duration in seconds.\n")
		sb.WriteString("# TYPE brain_http_request_duration_seconds summary\n")

		var durLines []string
		m.requestDuration.Range(func(key, value any) bool {
			k := key.(string)
			parts := strings.SplitN(k, ":", 2)
			if len(parts) == 2 {
				b := value.(*durationBucket)
				sumSec := float64(b.sum.Load()) / 1e6
				count := b.count.Load()
				durLines = append(durLines,
					fmt.Sprintf(`brain_http_request_duration_seconds_sum{method="%s",path="%s"} %.6f`, parts[0], parts[1], sumSec),
					fmt.Sprintf(`brain_http_request_duration_seconds_count{method="%s",path="%s"} %d`, parts[0], parts[1], count))
			}
			return true
		})
		sort.Strings(durLines)
		for _, line := range durLines {
			sb.WriteString(line)
			sb.WriteByte('\n')
		}

		sb.WriteString(fmt.Sprintf("\n# HELP brain_http_active_requests Currently active requests.\n"))
		sb.WriteString("# TYPE brain_http_active_requests gauge\n")
		sb.WriteString(fmt.Sprintf("brain_http_active_requests %d\n", m.activeRequests.Load()))

		c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(sb.String()))
	}
}
