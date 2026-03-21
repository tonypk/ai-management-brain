package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandleOpenClawStatus_NoTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/v1/openclaw/status", nil)
	handler := handleOpenClawStatus(nil)
	handler(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleOpenClawReport_NoTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/v1/openclaw/report?period=weekly", nil)
	handler := handleOpenClawReport(nil)
	handler(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleOpenClawReport_InvalidPeriod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/v1/openclaw/report?period=yearly", nil)
	c.Set("tenant_id", "550e8400-e29b-41d4-a716-446655440000")
	handler := handleOpenClawReport(nil)
	handler(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// TestHandleOpenClawReport_InvalidPeriod_ErrorMessage verifies the error message for bad period.
func TestHandleOpenClawReport_InvalidPeriod_ErrorMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/v1/openclaw/report?period=daily", nil)
	c.Set("tenant_id", "550e8400-e29b-41d4-a716-446655440000")
	handler := handleOpenClawReport(nil)
	handler(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	errMsg, _ := resp["error"].(string)
	if !strings.Contains(errMsg, "weekly") || !strings.Contains(errMsg, "monthly") {
		t.Errorf("error should mention valid periods, got %q", errMsg)
	}
}

// TestHandleOpenClawReport_ValidPeriodWeekly_NilQueries verifies that a valid "weekly" period
// with a valid tenant but nil queries causes a panic caught by Recovery (500).
func TestHandleOpenClawReport_ValidPeriodWeekly_NilQueries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/test", func(c *gin.Context) {
		c.Set("tenant_id", "550e8400-e29b-41d4-a716-446655440000")
		handleOpenClawReport(nil)(c)
	})

	req, _ := http.NewRequest("GET", "/test?period=weekly", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 (nil queries panic recovered)", w.Code)
	}
}

// TestHandleOpenClawReport_ValidPeriodMonthly_NilQueries verifies that "monthly" period
// with nil queries also triggers a recovered panic.
func TestHandleOpenClawReport_ValidPeriodMonthly_NilQueries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/test", func(c *gin.Context) {
		c.Set("tenant_id", "550e8400-e29b-41d4-a716-446655440000")
		handleOpenClawReport(nil)(c)
	})

	req, _ := http.NewRequest("GET", "/test?period=monthly", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 (nil queries panic recovered)", w.Code)
	}
}

func TestHandleOpenClawCommand_NoBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/openclaw/command", nil)
	c.Set("tenant_id", "550e8400-e29b-41d4-a716-446655440000")
	handler := handleOpenClawCommand(nil)
	handler(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// TestHandleOpenClawCommand_UnknownCommand verifies that an unrecognized command returns 400
// with the "unknown command" error and a list of available commands.
func TestHandleOpenClawCommand_UnknownCommand(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"command": "do something unknown"}`
	c.Request, _ = http.NewRequest("POST", "/api/v1/openclaw/command", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("tenant_id", "550e8400-e29b-41d4-a716-446655440000")
	handler := handleOpenClawCommand(nil)
	handler(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	errMsg, _ := resp["error"].(string)
	if errMsg != "unknown command" {
		t.Errorf("error = %q, want %q", errMsg, "unknown command")
	}
	cmds, ok := resp["available_commands"].([]interface{})
	if !ok || len(cmds) == 0 {
		t.Error("expected non-empty available_commands list")
	}
}

// TestHandleOpenClawCommand_EmptyCommand verifies that an empty command string is rejected.
func TestHandleOpenClawCommand_EmptyCommand(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"command": ""}`
	c.Request, _ = http.NewRequest("POST", "/api/v1/openclaw/command", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("tenant_id", "550e8400-e29b-41d4-a716-446655440000")
	handler := handleOpenClawCommand(nil)
	handler(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// TestHandleOpenClawCommand_ListMentors verifies that "list mentors" command returns mentor data
// without needing database access (it reads from the in-memory mentorDescriptions map).
func TestHandleOpenClawCommand_ListMentors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"command": "list mentors"}`
	c.Request, _ = http.NewRequest("POST", "/api/v1/openclaw/command", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("tenant_id", "550e8400-e29b-41d4-a716-446655440000")
	handler := handleOpenClawCommand(nil)
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["result"] != "mentors" {
		t.Errorf("result = %v, want %q", resp["result"], "mentors")
	}
	mentors, ok := resp["mentors"].([]interface{})
	if !ok || len(mentors) == 0 {
		t.Error("expected non-empty mentors list")
	}
	// Verify each mentor has id, name, description
	for i, m := range mentors {
		mentor, ok := m.(map[string]interface{})
		if !ok {
			t.Fatalf("mentors[%d] is not an object", i)
		}
		if _, ok := mentor["id"]; !ok {
			t.Errorf("mentors[%d] missing id", i)
		}
		if _, ok := mentor["name"]; !ok {
			t.Errorf("mentors[%d] missing name", i)
		}
		if _, ok := mentor["description"]; !ok {
			t.Errorf("mentors[%d] missing description", i)
		}
	}
}

// TestHandleOpenClawCommand_NoTenant verifies that missing tenant_id returns 400.
func TestHandleOpenClawCommand_NoTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"command": "list mentors"}`
	c.Request, _ = http.NewRequest("POST", "/api/v1/openclaw/command", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	// tenant_id not set
	handler := handleOpenClawCommand(nil)
	handler(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleOpenClawAlerts_NoTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/v1/openclaw/alerts", nil)
	handler := handleOpenClawAlerts(nil)
	handler(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// TestHandleOpenClawAlerts_ValidTenant_NilQueries verifies that a valid tenant with nil queries
// causes a panic caught by Recovery middleware (500).
func TestHandleOpenClawAlerts_ValidTenant_NilQueries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/test", func(c *gin.Context) {
		c.Set("tenant_id", "550e8400-e29b-41d4-a716-446655440000")
		handleOpenClawAlerts(nil)(c)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 (nil queries panic recovered)", w.Code)
	}
}

// TestHandleOpenClawStatus_ValidTenant_NilQueries verifies that a valid tenant with nil queries
// causes a panic caught by Recovery middleware (500).
func TestHandleOpenClawStatus_ValidTenant_NilQueries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/test", func(c *gin.Context) {
		c.Set("tenant_id", "550e8400-e29b-41d4-a716-446655440000")
		handleOpenClawStatus(nil)(c)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 (nil queries panic recovered)", w.Code)
	}
}

// TestHandleOpenClawStatus_NoTenant_ErrorMessage verifies the error body for missing tenant.
func TestHandleOpenClawStatus_NoTenant_ErrorMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/v1/openclaw/status", nil)
	handler := handleOpenClawStatus(nil)
	handler(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	errMsg, _ := resp["error"].(string)
	if errMsg != "invalid tenant" {
		t.Errorf("error = %q, want %q", errMsg, "invalid tenant")
	}
}

// TestHandleOpenClawAlerts_NoTenant_ErrorMessage verifies the error body for missing tenant.
func TestHandleOpenClawAlerts_NoTenant_ErrorMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/v1/openclaw/alerts", nil)
	handler := handleOpenClawAlerts(nil)
	handler(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	errMsg, _ := resp["error"].(string)
	if errMsg != "invalid tenant" {
		t.Errorf("error = %q, want %q", errMsg, "invalid tenant")
	}
}
