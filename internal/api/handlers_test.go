package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/api"
	"github.com/tonypk/ai-management-brain/internal/auth"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

var testSecret = []byte("0123456789abcdef0123456789abcdef")

// mockDBTX implements sqlc.DBTX for testing. It returns predefined rows from
// a registry keyed by SQL query prefix.
type mockDBTX struct {
	queryResults map[string]*mockRows
	queryRowFn   func(ctx context.Context, sql string, args ...interface{}) pgx.Row
	execFn       func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

func newMockDBTX() *mockDBTX {
	return &mockDBTX{queryResults: make(map[string]*mockRows)}
}

func (m *mockDBTX) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	if m.execFn != nil {
		return m.execFn(ctx, sql, args...)
	}
	return pgconn.NewCommandTag("OK"), nil
}

func (m *mockDBTX) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	for prefix, rows := range m.queryResults {
		if len(sql) >= len(prefix) && sql[:len(prefix)] == prefix {
			return rows, nil
		}
	}
	return &mockRows{done: true}, nil
}

func (m *mockDBTX) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	if m.queryRowFn != nil {
		return m.queryRowFn(ctx, sql, args...)
	}
	return &mockRow{err: pgx.ErrNoRows}
}

// mockRow implements pgx.Row.
type mockRow struct {
	err    error
	scanFn func(dest ...interface{}) error
}

func (r *mockRow) Scan(dest ...interface{}) error {
	if r.scanFn != nil {
		return r.scanFn(dest...)
	}
	return r.err
}

// mockRows implements pgx.Rows for list queries.
type mockRows struct {
	items   [][]interface{}
	current int
	done    bool
}

func (r *mockRows) Close()                                         {}
func (r *mockRows) Err() error                                     { return nil }
func (r *mockRows) CommandTag() pgconn.CommandTag                  { return pgconn.NewCommandTag("SELECT") }
func (r *mockRows) FieldDescriptions() []pgconn.FieldDescription   { return nil }
func (r *mockRows) RawValues() [][]byte                            { return nil }
func (r *mockRows) Conn() *pgx.Conn                                { return nil }

func (r *mockRows) Next() bool {
	if r.done || r.current >= len(r.items) {
		return false
	}
	r.current++
	return true
}

func (r *mockRows) Scan(dest ...interface{}) error {
	if r.current == 0 || r.current > len(r.items) {
		return pgx.ErrNoRows
	}
	row := r.items[r.current-1]
	for i, d := range dest {
		if i >= len(row) {
			break
		}
		switch ptr := d.(type) {
		case *pgtype.UUID:
			if v, ok := row[i].(pgtype.UUID); ok {
				*ptr = v
			}
		case *string:
			if v, ok := row[i].(string); ok {
				*ptr = v
			}
		case *bool:
			if v, ok := row[i].(bool); ok {
				*ptr = v
			}
		case *pgtype.Int8:
			if v, ok := row[i].(pgtype.Int8); ok {
				*ptr = v
			}
		case *pgtype.Text:
			if v, ok := row[i].(pgtype.Text); ok {
				*ptr = v
			}
		case *pgtype.Timestamptz:
			if v, ok := row[i].(pgtype.Timestamptz); ok {
				*ptr = v
			}
		case *int64:
			if v, ok := row[i].(int64); ok {
				*ptr = v
			}
		case *float64:
			if v, ok := row[i].(float64); ok {
				*ptr = v
			}
		case *int32:
			if v, ok := row[i].(int32); ok {
				*ptr = v
			}
		case *[]byte:
			if v, ok := row[i].([]byte); ok {
				*ptr = v
			}
		}
	}
	return nil
}

func (r *mockRows) Values() ([]interface{}, error) { return nil, nil }

func makeTestUUID(b byte) pgtype.UUID {
	var u pgtype.UUID
	u.Valid = true
	for i := range u.Bytes {
		u.Bytes[i] = b
	}
	return u
}

func setupRouter(db sqlc.DBTX) *gin.Engine {
	gin.SetMode(gin.TestMode)
	queries := sqlc.New(db)
	return api.NewRouter(api.RouterConfig{
		Queries:   queries,
		JWTSecret: testSecret,
		Redis:     nil, // no rate limiting in tests
	})
}

func generateTestToken(userID, tenantID, role string) string {
	token, _ := auth.GenerateToken(userID, tenantID, role, testSecret)
	return token
}

// TestAuthMiddleware_NoToken verifies that a request without an auth token
// returns 401 Unauthorized.
func TestAuthMiddleware_NoToken(t *testing.T) {
	db := newMockDBTX()
	router := setupRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenant", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := resp["error"]; !ok {
		t.Error("expected error field in response")
	}
}

// TestAuthMiddleware_InvalidToken verifies that an invalid token returns 401.
func TestAuthMiddleware_InvalidToken(t *testing.T) {
	db := newMockDBTX()
	router := setupRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenant", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-here")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// TestRequireRole_Forbidden verifies that a non-boss user cannot access
// boss-only endpoints.
func TestRequireRole_Forbidden(t *testing.T) {
	db := newMockDBTX()
	router := setupRouter(db)

	// Generate a token with "member" role
	token := generateTestToken(
		"550e8400-e29b-41d4-a716-446655440000",
		"660e8400-e29b-41d4-a716-446655440000",
		"member",
	)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/tenant", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := resp["error"]; !ok {
		t.Error("expected error field in response")
	}
}

// TestDashboardStatsEndpoint verifies the dashboard endpoint returns the
// expected JSON structure with mock data.
func TestDashboardStatsEndpoint(t *testing.T) {
	tenantUUID := makeTestUUID(0x66)

	db := newMockDBTX()

	// Mock QueryRow for GetTenant, CountReportsByTenantDate, GetLatestSummary
	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		// GetTenant query
		if len(sql) > 20 && sql[:20] == "-- name: GetTenant :" {
			return &mockRow{scanFn: func(dest ...interface{}) error {
				// id, name, timezone, anthropic_key, mentor_id, mentor_blend, bot_token, boss_chat_id, config, created_at
				if len(dest) >= 10 {
					if p, ok := dest[0].(*pgtype.UUID); ok {
						*p = tenantUUID
					}
					if p, ok := dest[1].(*string); ok {
						*p = "Test Team"
					}
					if p, ok := dest[2].(*string); ok {
						*p = "Asia/Singapore"
					}
					if p, ok := dest[3].(*pgtype.Text); ok {
						*p = pgtype.Text{}
					}
					if p, ok := dest[4].(*string); ok {
						*p = "inamori"
					}
					if p, ok := dest[5].(*[]byte); ok {
						*p = nil
					}
					if p, ok := dest[6].(*pgtype.Text); ok {
						*p = pgtype.Text{}
					}
					if p, ok := dest[7].(*int64); ok {
						*p = 12345
					}
					if p, ok := dest[8].(*[]byte); ok {
						*p = []byte("{}")
					}
					if p, ok := dest[9].(*pgtype.Timestamptz); ok {
						*p = pgtype.Timestamptz{}
					}
				}
				return nil
			}}
		}

		// CountReportsByTenantDate query
		if len(sql) > 30 && sql[:13] == "-- name: Coun" {
			return &mockRow{scanFn: func(dest ...interface{}) error {
				if p, ok := dest[0].(*int64); ok {
					*p = 3
				}
				return nil
			}}
		}

		// GetLatestSummary query
		if len(sql) > 20 && sql[:20] == "-- name: GetLatestSu" {
			return &mockRow{err: pgx.ErrNoRows}
		}

		return &mockRow{err: pgx.ErrNoRows}
	}

	// Mock Query for ListActiveEmployees
	empUUID1 := makeTestUUID(0x01)
	empUUID2 := makeTestUUID(0x02)
	db.queryResults["-- name: ListActiveEmployees"] = &mockRows{
		items: [][]interface{}{
			{empUUID1, tenantUUID, "Alice", pgtype.Int8{Int64: 111, Valid: true}, "default", "member", pgtype.Text{}, true, pgtype.Timestamptz{}},
			{empUUID2, tenantUUID, "Bob", pgtype.Int8{Int64: 222, Valid: true}, "philippines", "member", pgtype.Text{}, true, pgtype.Timestamptz{}},
		},
	}

	router := setupRouter(db)

	token := generateTestToken(
		"550e8400-e29b-41d4-a716-446655440000",
		"66666666-6666-6666-6666-666666666666",
		"boss",
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("expected data field in response")
	}

	// Verify structure
	if _, ok := data["employee_count"]; !ok {
		t.Error("missing employee_count in dashboard data")
	}
	if _, ok := data["today_submissions"]; !ok {
		t.Error("missing today_submissions in dashboard data")
	}
	if _, ok := data["current_mentor"]; !ok {
		t.Error("missing current_mentor in dashboard data")
	}
	if _, ok := data["last_summary_date"]; !ok {
		t.Error("missing last_summary_date in dashboard data")
	}

	// Verify values
	if empCount, ok := data["employee_count"].(float64); ok {
		if int(empCount) != 2 {
			t.Errorf("employee_count = %v, want 2", empCount)
		}
	}
	if submissions, ok := data["today_submissions"].(float64); ok {
		if int(submissions) != 3 {
			t.Errorf("today_submissions = %v, want 3", submissions)
		}
	}
	if mentor, ok := data["current_mentor"].(string); ok {
		if mentor != "inamori" {
			t.Errorf("current_mentor = %q, want %q", mentor, "inamori")
		}
	}
	// last_summary_date should be empty string since we returned ErrNoRows
	if summaryDate, ok := data["last_summary_date"].(string); ok {
		if summaryDate != "" {
			t.Errorf("last_summary_date = %q, want empty string", summaryDate)
		}
	}
}

// TestPublicRoutes_NoAuth verifies that public routes (auth endpoints) work without tokens.
func TestPublicRoutes_NoAuth(t *testing.T) {
	db := newMockDBTX()
	router := setupRouter(db)

	tests := []struct {
		name   string
		method string
		path   string
		// We expect either 400 (bad JSON) or 404 or some non-401 status
		// The point is it should NOT be 401 (no auth middleware)
	}{
		{"login without body", http.MethodPost, "/api/v1/auth/login"},
		{"register without body", http.MethodPost, "/api/v1/auth/register"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusUnauthorized {
				t.Errorf("%s returned 401 — public routes should not require auth", tt.name)
			}
		})
	}
}

// TestListEmployees_Authenticated verifies the list employees endpoint works
// with a valid token.
func TestListEmployees_Authenticated(t *testing.T) {
	tenantUUID := makeTestUUID(0xAA)
	empUUID := makeTestUUID(0xBB)

	db := newMockDBTX()
	db.queryResults["-- name: ListActiveEmployees"] = &mockRows{
		items: [][]interface{}{
			{empUUID, tenantUUID, "Charlie", pgtype.Int8{}, "default", "member", pgtype.Text{String: "ABC123", Valid: true}, true, pgtype.Timestamptz{}},
		},
	}

	router := setupRouter(db)
	token := generateTestToken(
		"550e8400-e29b-41d4-a716-446655440000",
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"boss",
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	data, ok := resp["data"].([]interface{})
	if !ok {
		t.Fatal("expected data array in response")
	}
	if len(data) != 1 {
		t.Errorf("expected 1 employee, got %d", len(data))
	}
}
