package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
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
		case *pgtype.Date:
			if v, ok := row[i].(pgtype.Date); ok {
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


// ---------------------------------------------------------------------------
// Additional handler tests
// ---------------------------------------------------------------------------

// tenantScanFn returns a scanFn that populates a Tenant row from the GetTenant query.
func tenantScanFn(tenantUUID pgtype.UUID, name, tz, mentorID string) func(dest ...interface{}) error {
	return func(dest ...interface{}) error {
		if len(dest) < 10 {
			return nil
		}
		if p, ok := dest[0].(*pgtype.UUID); ok {
			*p = tenantUUID
		}
		if p, ok := dest[1].(*string); ok {
			*p = name
		}
		if p, ok := dest[2].(*string); ok {
			*p = tz
		}
		if p, ok := dest[3].(*pgtype.Text); ok {
			*p = pgtype.Text{} // anthropic_key
		}
		if p, ok := dest[4].(*string); ok {
			*p = mentorID
		}
		if p, ok := dest[5].(*[]byte); ok {
			*p = nil // mentor_blend
		}
		if p, ok := dest[6].(*pgtype.Text); ok {
			*p = pgtype.Text{} // bot_token
		}
		if p, ok := dest[7].(*int64); ok {
			*p = 0 // boss_chat_id
		}
		if p, ok := dest[8].(*[]byte); ok {
			*p = []byte("{}") // config
		}
		if p, ok := dest[9].(*pgtype.Timestamptz); ok {
			*p = pgtype.Timestamptz{}
		}
		return nil
	}
}

// TestHandleGetTenant verifies GET /api/v1/tenant returns tenant data.
func TestHandleGetTenant(t *testing.T) {
	tenantUUID := makeTestUUID(0x77)
	db := newMockDBTX()

	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		if len(sql) > 20 && sql[:20] == "-- name: GetTenant :" {
			return &mockRow{scanFn: tenantScanFn(tenantUUID, "Alpha Team", "Asia/Tokyo", "dalio")}
		}
		return &mockRow{err: pgx.ErrNoRows}
	}

	router := setupRouter(db)
	token := generateTestToken(
		"550e8400-e29b-41d4-a716-446655440000",
		"77777777-7777-7777-7777-777777777777",
		"boss",
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenant", nil)
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
		t.Fatal("expected data object in response")
	}

	if name, ok := data["name"].(string); !ok || name != "Alpha Team" {
		t.Errorf("name = %v, want %q", data["name"], "Alpha Team")
	}
	if tz, ok := data["timezone"].(string); !ok || tz != "Asia/Tokyo" {
		t.Errorf("timezone = %v, want %q", data["timezone"], "Asia/Tokyo")
	}
	if mentor, ok := data["mentor_id"].(string); !ok || mentor != "dalio" {
		t.Errorf("mentor_id = %v, want %q", data["mentor_id"], "dalio")
	}
	if id, ok := data["id"].(string); !ok || id == "" {
		t.Errorf("expected non-empty id string, got %v", data["id"])
	}
}

// TestHandleUpdateTenant_ValidBody verifies PUT /api/v1/tenant with valid JSON.
func TestHandleUpdateTenant_ValidBody(t *testing.T) {
	db := newMockDBTX()

	var execCalled bool
	db.execFn = func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
		if len(sql) > 20 && sql[:20] == "-- name: UpdateTenan" {
			execCalled = true
		}
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}

	router := setupRouter(db)
	token := generateTestToken(
		"550e8400-e29b-41d4-a716-446655440000",
		"77777777-7777-7777-7777-777777777777",
		"boss",
	)

	body := `{"name":"Renamed Team","timezone":"America/New_York"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tenant", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if !execCalled {
		t.Error("expected exec to be called for UpdateTenantNameTimezone")
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("expected data object in response")
	}
	if data["name"] != "Renamed Team" {
		t.Errorf("name = %v, want %q", data["name"], "Renamed Team")
	}
	if data["timezone"] != "America/New_York" {
		t.Errorf("timezone = %v, want %q", data["timezone"], "America/New_York")
	}
}

// TestHandleUpdateTenant_InvalidTimezone verifies PUT /api/v1/tenant rejects bad timezone.
func TestHandleUpdateTenant_InvalidTimezone(t *testing.T) {
	db := newMockDBTX()
	router := setupRouter(db)
	token := generateTestToken(
		"550e8400-e29b-41d4-a716-446655440000",
		"77777777-7777-7777-7777-777777777777",
		"boss",
	)

	body := `{"name":"Team","timezone":"Mars/Olympus"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tenant", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	errMsg, _ := resp["error"].(string)
	if !strings.Contains(errMsg, "invalid timezone") {
		t.Errorf("expected error about invalid timezone, got %q", errMsg)
	}
}

// TestHandleCreateEmployee_ValidData verifies POST /api/v1/employees with valid payload.
func TestHandleCreateEmployee_ValidData(t *testing.T) {
	empUUID := makeTestUUID(0xCC)
	tenantUUID := makeTestUUID(0x77)

	db := newMockDBTX()
	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		if len(sql) > 20 && sql[:20] == "-- name: CreateEmplo" {
			return &mockRow{scanFn: func(dest ...interface{}) error {
				if len(dest) >= 9 {
					if p, ok := dest[0].(*pgtype.UUID); ok {
						*p = empUUID
					}
					if p, ok := dest[1].(*pgtype.UUID); ok {
						*p = tenantUUID
					}
					if p, ok := dest[2].(*string); ok {
						*p = "David"
					}
					if p, ok := dest[3].(*pgtype.Int8); ok {
						*p = pgtype.Int8{}
					}
					if p, ok := dest[4].(*string); ok {
						*p = "philippines"
					}
					if p, ok := dest[5].(*string); ok {
						*p = "member"
					}
					if p, ok := dest[6].(*pgtype.Text); ok {
						*p = pgtype.Text{String: "ABCD1234", Valid: true}
					}
					if p, ok := dest[7].(*bool); ok {
						*p = true
					}
					if p, ok := dest[8].(*pgtype.Timestamptz); ok {
						*p = pgtype.Timestamptz{}
					}
				}
				return nil
			}}
		}
		return &mockRow{err: pgx.ErrNoRows}
	}

	router := setupRouter(db)
	token := generateTestToken(
		"550e8400-e29b-41d4-a716-446655440000",
		"77777777-7777-7777-7777-777777777777",
		"boss",
	)

	body := `{"name":"David","culture_code":"philippines"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/employees", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("expected data object in response")
	}
	if data["name"] != "David" {
		t.Errorf("name = %v, want %q", data["name"], "David")
	}
	if data["culture_code"] != "philippines" {
		t.Errorf("culture_code = %v, want %q", data["culture_code"], "philippines")
	}
	// invite_code should be a non-empty string (generated by generateInviteCode)
	if code, ok := data["invite_code"].(string); !ok || len(code) != 8 {
		t.Errorf("invite_code = %v, want 8-char string", data["invite_code"])
	}
}

// TestHandleCreateEmployee_InvalidCulture verifies POST /api/v1/employees rejects bad culture code.
func TestHandleCreateEmployee_InvalidCulture(t *testing.T) {
	db := newMockDBTX()
	router := setupRouter(db)
	token := generateTestToken(
		"550e8400-e29b-41d4-a716-446655440000",
		"77777777-7777-7777-7777-777777777777",
		"boss",
	)

	body := `{"name":"Eve","culture_code":"atlantis"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/employees", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	errMsg, _ := resp["error"].(string)
	if !strings.Contains(errMsg, "invalid culture code") {
		t.Errorf("expected error about invalid culture code, got %q", errMsg)
	}
}

// TestHandleListReports_WithDate verifies GET /api/v1/reports?date=2026-03-20 returns report data.
func TestHandleListReports_WithDate(t *testing.T) {
	tenantUUID := makeTestUUID(0x77)
	reportUUID := makeTestUUID(0xDD)
	empUUID := makeTestUUID(0xEE)

	db := newMockDBTX()

	// GetReportsByTenantDate is a :many query — uses Query.
	// The query scans: id, tenant_id, employee_id, report_date, answers, blockers, sentiment, submitted_at, employee_name
	// mockRows.Scan handles pgtype.UUID, pgtype.Text, pgtype.Timestamptz, []byte, string
	// but NOT pgtype.Date. The report_date column is scanned by sqlc into pgtype.Date.
	// Since our mockRows doesn't handle pgtype.Date, the field will remain zero-valued,
	// but the handler formats it with .Time.Format("2006-01-02") which yields "0001-01-01".
	// We still verify the rest of the fields work correctly.
	db.queryResults["-- name: GetReportsByTenantDate"] = &mockRows{
		items: [][]interface{}{
			{
				reportUUID,   // id (pgtype.UUID)
				tenantUUID,   // tenant_id (pgtype.UUID)
				empUUID,      // employee_id (pgtype.UUID)
				pgtype.Date{Time: time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC), Valid: true}, // report_date
				[]byte(`[{"q":"What did you do?","a":"Wrote tests"}]`), // answers ([]byte)
				pgtype.Text{String: "none", Valid: true},                // blockers (pgtype.Text)
				pgtype.Text{String: "positive", Valid: true},            // sentiment (pgtype.Text)
				pgtype.Timestamptz{},                                    // submitted_at
				"Alice",                                                 // employee_name (string)
			},
		},
	}

	router := setupRouter(db)
	token := generateTestToken(
		"550e8400-e29b-41d4-a716-446655440000",
		"77777777-7777-7777-7777-777777777777",
		"boss",
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports?date=2026-03-20", nil)
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
		t.Fatalf("expected 1 report, got %d", len(data))
	}

	report, ok := data[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected report to be an object")
	}
	if report["employee_name"] != "Alice" {
		t.Errorf("employee_name = %v, want %q", report["employee_name"], "Alice")
	}
	if report["blockers"] != "none" {
		t.Errorf("blockers = %v, want %q", report["blockers"], "none")
	}
	if report["sentiment"] != "positive" {
		t.Errorf("sentiment = %v, want %q", report["sentiment"], "positive")
	}
}

// TestHandleListReports_MissingDate verifies GET /api/v1/reports without date returns 400.
func TestHandleListReports_MissingDate(t *testing.T) {
	db := newMockDBTX()
	router := setupRouter(db)
	token := generateTestToken(
		"550e8400-e29b-41d4-a716-446655440000",
		"77777777-7777-7777-7777-777777777777",
		"boss",
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	errMsg, _ := resp["error"].(string)
	if !strings.Contains(errMsg, "date") {
		t.Errorf("expected error about missing date, got %q", errMsg)
	}
}

// TestHandleGetSummary_NotFound verifies GET /api/v1/reports/summary?date=2026-03-20
// returns 404 when no summary exists.
func TestHandleGetSummary_NotFound(t *testing.T) {
	db := newMockDBTX()
	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		return &mockRow{err: pgx.ErrNoRows}
	}

	router := setupRouter(db)
	token := generateTestToken(
		"550e8400-e29b-41d4-a716-446655440000",
		"77777777-7777-7777-7777-777777777777",
		"boss",
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/summary?date=2026-03-20", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	errMsg, _ := resp["error"].(string)
	if !strings.Contains(errMsg, "summary not found") {
		t.Errorf("expected 'summary not found' error, got %q", errMsg)
	}
}

// TestHandleGetMentor verifies GET /api/v1/mentor returns current mentor and available list.
func TestHandleGetMentor(t *testing.T) {
	tenantUUID := makeTestUUID(0x77)
	db := newMockDBTX()

	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		if len(sql) > 20 && sql[:20] == "-- name: GetTenant :" {
			return &mockRow{scanFn: tenantScanFn(tenantUUID, "Alpha Team", "Asia/Singapore", "grove")}
		}
		return &mockRow{err: pgx.ErrNoRows}
	}

	router := setupRouter(db)
	token := generateTestToken(
		"550e8400-e29b-41d4-a716-446655440000",
		"77777777-7777-7777-7777-777777777777",
		"boss",
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mentor", nil)
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
		t.Fatal("expected data object in response")
	}

	if data["current_mentor_id"] != "grove" {
		t.Errorf("current_mentor_id = %v, want %q", data["current_mentor_id"], "grove")
	}

	available, ok := data["available_mentors"].([]interface{})
	if !ok {
		t.Fatal("expected available_mentors array in response")
	}
	if len(available) == 0 {
		t.Error("expected at least one available mentor")
	}
	// Verify each available mentor has id, name, description
	for i, m := range available {
		mentor, ok := m.(map[string]interface{})
		if !ok {
			t.Fatalf("available_mentors[%d] is not an object", i)
		}
		if _, ok := mentor["id"]; !ok {
			t.Errorf("available_mentors[%d] missing id", i)
		}
		if _, ok := mentor["name"]; !ok {
			t.Errorf("available_mentors[%d] missing name", i)
		}
		if _, ok := mentor["description"]; !ok {
			t.Errorf("available_mentors[%d] missing description", i)
		}
	}
}

// TestHandleUpdateMentor_ValidMentorID verifies PUT /api/v1/mentor with valid mentor_id.
func TestHandleUpdateMentor_ValidMentorID(t *testing.T) {
	db := newMockDBTX()

	var execCalled bool
	db.execFn = func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
		if len(sql) > 20 && sql[:20] == "-- name: UpdateTenan" {
			execCalled = true
		}
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}

	router := setupRouter(db)
	token := generateTestToken(
		"550e8400-e29b-41d4-a716-446655440000",
		"77777777-7777-7777-7777-777777777777",
		"boss",
	)

	body := `{"mentor_id":"bezos"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/mentor", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if !execCalled {
		t.Error("expected exec to be called for UpdateTenantMentor")
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("expected data object in response")
	}
	if data["mentor_id"] != "bezos" {
		t.Errorf("mentor_id = %v, want %q", data["mentor_id"], "bezos")
	}
}

// TestHandleUpdateMentor_InvalidMentorID verifies PUT /api/v1/mentor rejects invalid mentor.
func TestHandleUpdateMentor_InvalidMentorID(t *testing.T) {
	db := newMockDBTX()
	router := setupRouter(db)
	token := generateTestToken(
		"550e8400-e29b-41d4-a716-446655440000",
		"77777777-7777-7777-7777-777777777777",
		"boss",
	)

	body := `{"mentor_id":"confucius"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/mentor", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	errMsg, _ := resp["error"].(string)
	if !strings.Contains(errMsg, "invalid mentor_id") {
		t.Errorf("expected error about invalid mentor_id, got %q", errMsg)
	}
}

// TestFormatUUID_ValidUUID indirectly tests formatUUID by verifying the UUID
// format in the API response matches the expected hex-dash pattern.
func TestFormatUUID_ValidUUID(t *testing.T) {
	tenantUUID := makeTestUUID(0xAB)
	db := newMockDBTX()

	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		if len(sql) > 20 && sql[:20] == "-- name: GetTenant :" {
			return &mockRow{scanFn: tenantScanFn(tenantUUID, "UUID Test", "UTC", "inamori")}
		}
		return &mockRow{err: pgx.ErrNoRows}
	}

	router := setupRouter(db)
	token := generateTestToken(
		"550e8400-e29b-41d4-a716-446655440000",
		"abababab-abab-abab-abab-abababababab",
		"boss",
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenant", nil)
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
	data := resp["data"].(map[string]interface{})
	id, ok := data["id"].(string)
	if !ok || id == "" {
		t.Fatal("expected non-empty id string")
	}

	expected := "abababab-abab-abab-abab-abababababab"
	if id != expected {
		t.Errorf("formatUUID = %q, want %q", id, expected)
	}
}

// TestFormatUUID_InvalidUUID tests that formatUUID returns empty string for
// a UUID with Valid=false.
func TestFormatUUID_InvalidUUID(t *testing.T) {
	invalidUUID := pgtype.UUID{Valid: false}
	db := newMockDBTX()

	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		if len(sql) > 20 && sql[:20] == "-- name: GetTenant :" {
			return &mockRow{scanFn: func(dest ...interface{}) error {
				if len(dest) >= 10 {
					if p, ok := dest[0].(*pgtype.UUID); ok {
						*p = invalidUUID
					}
					if p, ok := dest[1].(*string); ok {
						*p = "Test"
					}
					if p, ok := dest[2].(*string); ok {
						*p = "UTC"
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
						*p = 0
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
		return &mockRow{err: pgx.ErrNoRows}
	}

	router := setupRouter(db)
	token := generateTestToken(
		"550e8400-e29b-41d4-a716-446655440000",
		"77777777-7777-7777-7777-777777777777",
		"boss",
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenant", nil)
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
	data := resp["data"].(map[string]interface{})
	id, _ := data["id"].(string)
	if id != "" {
		t.Errorf("expected empty string for invalid UUID, got %q", id)
	}
}

// TestParseDate_InvalidDate indirectly tests parseDate by sending an invalid
// date to the reports endpoint.
func TestParseDate_InvalidDate(t *testing.T) {
	db := newMockDBTX()
	router := setupRouter(db)
	token := generateTestToken(
		"550e8400-e29b-41d4-a716-446655440000",
		"77777777-7777-7777-7777-777777777777",
		"boss",
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports?date=not-a-date", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	errMsg, _ := resp["error"].(string)
	if !strings.Contains(errMsg, "invalid date") {
		t.Errorf("expected 'invalid date' error, got %q", errMsg)
	}
}

// TestParseDate_ValidDate indirectly tests parseDate by sending a valid date
// to the summary endpoint (which uses parseDate).
func TestParseDate_ValidDate(t *testing.T) {
	db := newMockDBTX()
	// Return ErrNoRows for the summary — we just want to verify parseDate accepted the format.
	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		return &mockRow{err: pgx.ErrNoRows}
	}

	router := setupRouter(db)
	token := generateTestToken(
		"550e8400-e29b-41d4-a716-446655440000",
		"77777777-7777-7777-7777-777777777777",
		"boss",
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/summary?date=2026-03-20", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should be 404 (not found), NOT 400 (bad date). This confirms parseDate succeeded.
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 (valid date, no summary), got %d: %s", w.Code, w.Body.String())
	}
}

// TestGenerateInviteCode_ThroughCreateEmployee verifies that the invite code
// generated by handleCreateEmployee is 8-char uppercase hex.
func TestGenerateInviteCode_ThroughCreateEmployee(t *testing.T) {
	db := newMockDBTX()

	var capturedInviteCode string
	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		if len(sql) > 20 && sql[:20] == "-- name: CreateEmplo" {
			// args: tenant_id, name, telegram_id, culture_code, role, invite_code
			if len(args) >= 6 {
				if ic, ok := args[5].(pgtype.Text); ok {
					capturedInviteCode = ic.String
				}
			}
			empUUID := makeTestUUID(0xDD)
			tenantUUID := makeTestUUID(0x77)
			return &mockRow{scanFn: func(dest ...interface{}) error {
				if len(dest) >= 9 {
					if p, ok := dest[0].(*pgtype.UUID); ok {
						*p = empUUID
					}
					if p, ok := dest[1].(*pgtype.UUID); ok {
						*p = tenantUUID
					}
					if p, ok := dest[2].(*string); ok {
						*p = "Frank"
					}
					if p, ok := dest[3].(*pgtype.Int8); ok {
						*p = pgtype.Int8{}
					}
					if p, ok := dest[4].(*string); ok {
						*p = "default"
					}
					if p, ok := dest[5].(*string); ok {
						*p = "member"
					}
					if p, ok := dest[6].(*pgtype.Text); ok {
						*p = pgtype.Text{String: capturedInviteCode, Valid: true}
					}
					if p, ok := dest[7].(*bool); ok {
						*p = true
					}
					if p, ok := dest[8].(*pgtype.Timestamptz); ok {
						*p = pgtype.Timestamptz{}
					}
				}
				return nil
			}}
		}
		return &mockRow{err: pgx.ErrNoRows}
	}

	router := setupRouter(db)
	token := generateTestToken(
		"550e8400-e29b-41d4-a716-446655440000",
		"77777777-7777-7777-7777-777777777777",
		"boss",
	)

	body := `{"name":"Frank"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/employees", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the invite code format: 8 characters, uppercase hex
	if len(capturedInviteCode) != 8 {
		t.Errorf("invite code length = %d, want 8", len(capturedInviteCode))
	}
	for _, ch := range capturedInviteCode {
		if !((ch >= '0' && ch <= '9') || (ch >= 'A' && ch <= 'F')) {
			t.Errorf("invite code contains non-uppercase-hex char: %c in %q", ch, capturedInviteCode)
			break
		}
	}
}
