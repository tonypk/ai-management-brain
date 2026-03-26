package api_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// webhookPayload builds a minimal HalaOS webhook envelope JSON body.
func webhookPayload(id, eventType string, hrCompanyID int64) []byte {
	data := map[string]interface{}{
		"hr_company_id": hrCompanyID,
		"employee_id":   42,
		"employee_no":   "EMP001",
		"employee_name": "Test Employee",
		"risk_score":    0.75,
		"factors":       []map[string]interface{}{},
	}
	dataBytes, _ := json.Marshal(data)

	envelope := map[string]interface{}{
		"id":         id,
		"timestamp":  "2026-03-26T10:00:00Z",
		"event_type": eventType,
		"data":       json.RawMessage(dataBytes),
	}
	body, _ := json.Marshal(envelope)
	return body
}

// computeHMAC computes the sha256 HMAC of body using secret.
func computeHMAC(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// linkScanFn returns a scanFn that populates a HalaosLink row.
func linkScanFn(tenantID pgtype.UUID, secret string, companyID int64) func(dest ...interface{}) error {
	return func(dest ...interface{}) error {
		// Scan order: id, tenant_id, webhook_secret, halaos_company_id, is_active, created_at
		if len(dest) >= 6 {
			if p, ok := dest[0].(*pgtype.UUID); ok {
				*p = makeTestUUID(0x01)
			}
			if p, ok := dest[1].(*pgtype.UUID); ok {
				*p = tenantID
			}
			if p, ok := dest[2].(*string); ok {
				*p = secret
			}
			if p, ok := dest[3].(*int64); ok {
				*p = companyID
			}
			if p, ok := dest[4].(*bool); ok {
				*p = true
			}
			if p, ok := dest[5].(*pgtype.Timestamptz); ok {
				*p = pgtype.Timestamptz{}
			}
		}
		return nil
	}
}

// halaosEventIDScanFn returns a scanFn that populates a UUID (for GetHalaOSEventByKey).
func halaosEventIDScanFn(id pgtype.UUID) func(dest ...interface{}) error {
	return func(dest ...interface{}) error {
		if len(dest) >= 1 {
			if p, ok := dest[0].(*pgtype.UUID); ok {
				*p = id
			}
		}
		return nil
	}
}

// halaosEventScanFn returns a scanFn for CreateHalaOSEvent result.
func halaosEventScanFn(tenantID pgtype.UUID, eventType, idempotencyKey string) func(dest ...interface{}) error {
	return func(dest ...interface{}) error {
		// Scan order: id, tenant_id, event_type, idempotency_key, payload, processed_at
		if len(dest) >= 6 {
			if p, ok := dest[0].(*pgtype.UUID); ok {
				*p = makeTestUUID(0x02)
			}
			if p, ok := dest[1].(*pgtype.UUID); ok {
				*p = tenantID
			}
			if p, ok := dest[2].(*string); ok {
				*p = eventType
			}
			if p, ok := dest[3].(*string); ok {
				*p = idempotencyKey
			}
			if p, ok := dest[4].(*[]byte); ok {
				*p = []byte("{}")
			}
			if p, ok := dest[5].(*pgtype.Timestamptz); ok {
				*p = pgtype.Timestamptz{}
			}
		}
		return nil
	}
}

// TestHalaOSWebhook_MissingSignature verifies that a POST without
// X-Signature-256 header returns 401.
func TestHalaOSWebhook_MissingSignature(t *testing.T) {
	const (
		secret      = "test-webhook-secret"
		companyID   = int64(1001)
		eventID     = "evt-no-sig-001"
		eventType   = "hr.risk.updated"
	)
	tenantID := makeTestUUID(0xAA)

	db := newMockDBTX()
	// GetHalaOSLinkByCompanyID must succeed so we reach signature check.
	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		if strings.Contains(sql, "GetHalaOSLinkByCompanyID") {
			return &mockRow{scanFn: linkScanFn(tenantID, secret, companyID)}
		}
		return &mockRow{err: pgx.ErrNoRows}
	}

	router := setupRouter(db)
	body := webhookPayload(eventID, eventType, companyID)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/halaos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No X-Signature-256 header set.
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := resp["error"]; !ok {
		t.Error("expected error field in response")
	}
}

// TestHalaOSWebhook_InvalidSignature verifies that a POST with a wrong HMAC
// returns 401.
func TestHalaOSWebhook_InvalidSignature(t *testing.T) {
	const (
		secret    = "correct-secret"
		companyID = int64(1002)
		eventID   = "evt-bad-sig-001"
		eventType = "hr.risk.updated"
	)
	tenantID := makeTestUUID(0xBB)

	db := newMockDBTX()
	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		if strings.Contains(sql, "GetHalaOSLinkByCompanyID") {
			return &mockRow{scanFn: linkScanFn(tenantID, secret, companyID)}
		}
		return &mockRow{err: pgx.ErrNoRows}
	}

	router := setupRouter(db)
	body := webhookPayload(eventID, eventType, companyID)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/halaos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-256", "sha256=deadbeefdeadbeefdeadbeefdeadbeef")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	errMsg, _ := resp["error"].(string)
	if !strings.Contains(errMsg, "invalid signature") {
		t.Errorf("expected 'invalid signature' error, got %q", errMsg)
	}
}

// TestHalaOSWebhook_UnknownCompany verifies that a POST with valid format but
// a company_id not in halaos_links returns 401 (the handler returns 401 for
// unknown company as a security measure).
func TestHalaOSWebhook_UnknownCompany(t *testing.T) {
	const (
		companyID = int64(9999)
		eventID   = "evt-unknown-co-001"
		eventType = "hr.risk.updated"
	)

	db := newMockDBTX()
	// GetHalaOSLinkByCompanyID returns ErrNoRows (unknown company).
	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		if strings.Contains(sql, "GetHalaOSLinkByCompanyID") {
			return &mockRow{err: pgx.ErrNoRows}
		}
		return &mockRow{err: pgx.ErrNoRows}
	}

	router := setupRouter(db)
	body := webhookPayload(eventID, eventType, companyID)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/halaos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Signature doesn't matter here — we expect to fail before HMAC check.
	req.Header.Set("X-Signature-256", "sha256=anyvalue")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Handler returns 401 for unknown company (security: don't reveal 404).
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHalaOSWebhook_DuplicateEvent verifies that a POST with a previously
// processed event ID returns 200 with "already_processed".
func TestHalaOSWebhook_DuplicateEvent(t *testing.T) {
	const (
		secret    = "idempotency-secret"
		companyID = int64(1003)
		eventID   = "evt-dup-001"
		eventType = "hr.risk.updated"
	)
	tenantID := makeTestUUID(0xCC)
	existingEventID := makeTestUUID(0xDD)

	db := newMockDBTX()
	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		switch {
		case strings.Contains(sql, "GetHalaOSLinkByCompanyID"):
			return &mockRow{scanFn: linkScanFn(tenantID, secret, companyID)}
		case strings.Contains(sql, "GetHalaOSEventByKey"):
			// Return a result (event already exists).
			return &mockRow{scanFn: halaosEventIDScanFn(existingEventID)}
		}
		return &mockRow{err: pgx.ErrNoRows}
	}

	router := setupRouter(db)
	body := webhookPayload(eventID, eventType, companyID)
	sig := computeHMAC(secret, body)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/halaos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-256", "sha256="+sig)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	status, _ := resp["status"].(string)
	if status != "already_processed" {
		t.Errorf("expected status='already_processed', got %q", status)
	}
}

// TestHalaOSWebhook_ValidEvent verifies that a POST with a valid HMAC and
// known company returns 200 with status="ok".
func TestHalaOSWebhook_ValidEvent(t *testing.T) {
	const (
		secret    = "valid-event-secret"
		companyID = int64(1004)
		eventID   = "evt-valid-001"
		eventType = "hr.risk.updated"
	)
	tenantID := makeTestUUID(0xEE)

	db := newMockDBTX()
	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		switch {
		case strings.Contains(sql, "GetHalaOSLinkByCompanyID"):
			return &mockRow{scanFn: linkScanFn(tenantID, secret, companyID)}
		case strings.Contains(sql, "GetHalaOSEventByKey"):
			// Return ErrNoRows — event has not been processed before.
			return &mockRow{err: pgx.ErrNoRows}
		case strings.Contains(sql, "CreateHalaOSEvent"):
			return &mockRow{scanFn: halaosEventScanFn(tenantID, eventType, eventID)}
		}
		return &mockRow{err: pgx.ErrNoRows}
	}

	router := setupRouter(db)
	body := webhookPayload(eventID, eventType, companyID)
	sig := computeHMAC(secret, body)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/halaos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-256", "sha256="+sig)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	status, _ := resp["status"].(string)
	if status != "ok" {
		t.Errorf("expected status='ok', got %q", status)
	}
}

// TestHalaOSWebhook_ValidEvent_WithoutPrefix verifies that the handler also
// accepts signatures WITHOUT the "sha256=" prefix.
func TestHalaOSWebhook_ValidEvent_WithoutPrefix(t *testing.T) {
	const (
		secret    = "no-prefix-secret"
		companyID = int64(1005)
		eventID   = "evt-noprefix-001"
		eventType = "hr.burnout.updated"
	)
	tenantID := makeTestUUID(0xFF)

	db := newMockDBTX()
	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		switch {
		case strings.Contains(sql, "GetHalaOSLinkByCompanyID"):
			return &mockRow{scanFn: linkScanFn(tenantID, secret, companyID)}
		case strings.Contains(sql, "GetHalaOSEventByKey"):
			return &mockRow{err: pgx.ErrNoRows}
		case strings.Contains(sql, "CreateHalaOSEvent"):
			return &mockRow{scanFn: halaosEventScanFn(tenantID, eventType, eventID)}
		}
		return &mockRow{err: pgx.ErrNoRows}
	}

	// Build a burnout payload.
	data := map[string]interface{}{
		"hr_company_id": companyID,
		"employee_id":   99,
		"employee_no":   "EMP099",
		"employee_name": "Burnout Employee",
		"burnout_score": 0.85,
		"factors":       []interface{}{},
	}
	dataBytes, _ := json.Marshal(data)
	envelope := map[string]interface{}{
		"id":         eventID,
		"timestamp":  "2026-03-26T11:00:00Z",
		"event_type": eventType,
		"data":       json.RawMessage(dataBytes),
	}
	body, _ := json.Marshal(envelope)
	sig := computeHMAC(secret, body) // no "sha256=" prefix

	router := setupRouter(db)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/halaos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-256", sig) // bare hex, no prefix
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	status, _ := resp["status"].(string)
	if status != "ok" {
		t.Errorf("expected status='ok', got %q", status)
	}
}

// TestHalaOSWebhook_MissingEnvelopeID verifies that a POST with an empty
// event ID returns 400.
func TestHalaOSWebhook_MissingEnvelopeID(t *testing.T) {
	db := newMockDBTX()
	router := setupRouter(db)

	// Envelope with missing id field.
	envelope := map[string]interface{}{
		"id":         "",
		"timestamp":  "2026-03-26T10:00:00Z",
		"event_type": "hr.risk.updated",
		"data":       json.RawMessage(`{"hr_company_id":1001}`),
	}
	body, _ := json.Marshal(envelope)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/halaos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHalaOSWebhook_MissingHRCompanyID verifies that an event with no
// hr_company_id returns 400.
func TestHalaOSWebhook_MissingHRCompanyID(t *testing.T) {
	db := newMockDBTX()
	router := setupRouter(db)

	// Data section lacks hr_company_id.
	envelope := map[string]interface{}{
		"id":         "evt-no-company-001",
		"timestamp":  "2026-03-26T10:00:00Z",
		"event_type": "hr.risk.updated",
		"data":       json.RawMessage(`{"employee_id":1}`),
	}
	body, _ := json.Marshal(envelope)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/halaos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHalaOSWebhook_MapperNil verifies that a valid event with nil mapper
// (HalaOSMapper not wired) still returns 200 — dispatch is skipped but
// the event is logged for idempotency.
func TestHalaOSWebhook_MapperNil(t *testing.T) {
	const (
		secret    = "mapper-nil-secret"
		companyID = int64(1006)
		eventID   = "evt-mapper-nil-001"
		eventType = "hr.risk.updated"
	)
	tenantID := makeTestUUID(0x11)

	db := newMockDBTX()
	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		switch {
		case strings.Contains(sql, "GetHalaOSLinkByCompanyID"):
			return &mockRow{scanFn: linkScanFn(tenantID, secret, companyID)}
		case strings.Contains(sql, "GetHalaOSEventByKey"):
			return &mockRow{err: pgx.ErrNoRows}
		case strings.Contains(sql, "CreateHalaOSEvent"):
			return &mockRow{scanFn: halaosEventScanFn(tenantID, eventType, eventID)}
		}
		return &mockRow{err: pgx.ErrNoRows}
	}

	// setupRouter wires HalaOSMapper=nil, which is fine — events are logged only.
	router := setupRouter(db)

	body := webhookPayload(eventID, eventType, companyID)
	sig := computeHMAC(secret, body)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/halaos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-256", "sha256="+sig)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	status, _ := resp["status"].(string)
	if status != "ok" {
		t.Errorf("expected status='ok', got %q", status)
	}
}

