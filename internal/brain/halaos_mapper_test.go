package brain

// White-box tests for HalaOSMapper — using the internal `brain` package
// so we can access unexported types and call NewHalaOSMapper directly.

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	sqlc "github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// ---------------------------------------------------------------------------
// Minimal mock DBTX for mapper tests.
// ---------------------------------------------------------------------------

// mapperMockDBTX implements sqlc.DBTX. Each method dispatches via a function
// field so individual tests can install their own behaviour.
type mapperMockDBTX struct {
	queryRowFn func(ctx context.Context, sql string, args ...interface{}) pgx.Row
	execFn     func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

func (m *mapperMockDBTX) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	if m.execFn != nil {
		return m.execFn(ctx, sql, args...)
	}
	return pgconn.NewCommandTag("OK"), nil
}

func (m *mapperMockDBTX) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return &mapperMockRows{done: true}, nil
}

func (m *mapperMockDBTX) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	if m.queryRowFn != nil {
		return m.queryRowFn(ctx, sql, args...)
	}
	return &mapperMockRow{err: pgx.ErrNoRows}
}

// mapperMockRow implements pgx.Row.
type mapperMockRow struct {
	err    error
	scanFn func(dest ...interface{}) error
}

func (r *mapperMockRow) Scan(dest ...interface{}) error {
	if r.scanFn != nil {
		return r.scanFn(dest...)
	}
	return r.err
}

// mapperMockRows is a no-op pgx.Rows implementation (mapper doesn't use Query).
type mapperMockRows struct{ done bool }

func (r *mapperMockRows) Close()                                         {}
func (r *mapperMockRows) Err() error                                     { return nil }
func (r *mapperMockRows) CommandTag() pgconn.CommandTag                  { return pgconn.NewCommandTag("SELECT") }
func (r *mapperMockRows) FieldDescriptions() []pgconn.FieldDescription   { return nil }
func (r *mapperMockRows) RawValues() [][]byte                            { return nil }
func (r *mapperMockRows) Conn() *pgx.Conn                                { return nil }
func (r *mapperMockRows) Next() bool                                     { return false }
func (r *mapperMockRows) Scan(dest ...interface{}) error                 { return pgx.ErrNoRows }
func (r *mapperMockRows) Values() ([]interface{}, error)                 { return nil, nil }

// ---------------------------------------------------------------------------
// Helper: build a testUUID.
// ---------------------------------------------------------------------------
func mapperTestUUID(b byte) pgtype.UUID {
	var u pgtype.UUID
	u.Valid = true
	for i := range u.Bytes {
		u.Bytes[i] = b
	}
	return u
}

// employeeFullScanFn returns a scanFn that populates all 25 Employee fields.
// Only the id field (dest[0]) is varied; the rest get zero/default values.
func employeeFullScanFn(id pgtype.UUID) func(dest ...interface{}) error {
	tenantID := mapperTestUUID(0xAA)
	return func(dest ...interface{}) error {
		// id, tenant_id, name, telegram_id, culture_code, role, invite_code,
		// is_active, created_at, signal_phone, slack_id, lark_id,
		// preferred_channel, job_title, responsibilities, country, language,
		// org_unit_id, execution_score, current_load, strengths, risk_flags,
		// work_scope, halaos_employee_id, halaos_employee_no
		vals := []interface{}{
			id,
			tenantID,
			"Test Employee",
			pgtype.Int8{Int64: 42, Valid: true},
			"default",
			"member",
			pgtype.Text{String: "TESTCODE", Valid: true},
			true,
			pgtype.Timestamptz{},
			pgtype.Text{},
			pgtype.Text{},
			pgtype.Text{},
			"telegram",
			"",
			"",
			"",
			"",
			pgtype.UUID{},
			pgtype.Numeric{},
			pgtype.Text{},
			[]byte("[]"),
			[]byte("[]"),
			[]byte("[]"),
			pgtype.Int8{Int64: 42, Valid: true},
			pgtype.Text{String: "EMP001", Valid: true},
		}
		for i, d := range dest {
			if i >= len(vals) {
				break
			}
			switch ptr := d.(type) {
			case *pgtype.UUID:
				if v, ok := vals[i].(pgtype.UUID); ok {
					*ptr = v
				}
			case *string:
				if v, ok := vals[i].(string); ok {
					*ptr = v
				}
			case *pgtype.Int8:
				if v, ok := vals[i].(pgtype.Int8); ok {
					*ptr = v
				}
			case *bool:
				if v, ok := vals[i].(bool); ok {
					*ptr = v
				}
			case *pgtype.Timestamptz:
				if v, ok := vals[i].(pgtype.Timestamptz); ok {
					*ptr = v
				}
			case *pgtype.Text:
				if v, ok := vals[i].(pgtype.Text); ok {
					*ptr = v
				}
			case *pgtype.Numeric:
				if v, ok := vals[i].(pgtype.Numeric); ok {
					*ptr = v
				}
			case *[]byte:
				if v, ok := vals[i].([]byte); ok {
					*ptr = v
				}
			}
		}
		return nil
	}
}

// executionSignalFullScanFn populates all 9 ExecutionSignal fields.
func executionSignalFullScanFn(tenantID pgtype.UUID) func(dest ...interface{}) error {
	return func(dest ...interface{}) error {
		vals := []interface{}{
			mapperTestUUID(0x33),
			tenantID,
			"employee",
			mapperTestUUID(0x44),
			"flight_risk",
			pgtype.Numeric{},
			[]byte(`["low_engagement"]`),
			pgtype.Text{String: "30d", Valid: true},
			pgtype.Timestamptz{},
		}
		for i, d := range dest {
			if i >= len(vals) {
				break
			}
			switch ptr := d.(type) {
			case *pgtype.UUID:
				if v, ok := vals[i].(pgtype.UUID); ok {
					*ptr = v
				}
			case *string:
				if v, ok := vals[i].(string); ok {
					*ptr = v
				}
			case *[]byte:
				if v, ok := vals[i].([]byte); ok {
					*ptr = v
				}
			case *pgtype.Text:
				if v, ok := vals[i].(pgtype.Text); ok {
					*ptr = v
				}
			case *pgtype.Numeric:
				if v, ok := vals[i].(pgtype.Numeric); ok {
					*ptr = v
				}
			case *pgtype.Timestamptz:
				if v, ok := vals[i].(pgtype.Timestamptz); ok {
					*ptr = v
				}
			}
		}
		return nil
	}
}

// communicationEventFullScanFn populates all CommunicationEvent fields.
func communicationEventFullScanFn(tenantID pgtype.UUID) func(dest ...interface{}) error {
	return func(dest ...interface{}) error {
		vals := []interface{}{
			mapperTestUUID(0x55), // id
			tenantID,             // tenant_id
			"halaos",             // source_type
			pgtype.UUID{},        // source_id
			"halaos",             // platform
			"blindspot_detected", // event_type
			mapperTestUUID(0x66), // actor_id
			pgtype.UUID{},        // target_id
			pgtype.UUID{},        // related_task_id
			pgtype.UUID{},        // related_project_id
			pgtype.UUID{},        // related_goal_id
			[]byte(`{}`),         // payload
			pgtype.Numeric{},     // confidence
			pgtype.Timestamptz{}, // occurred_at
			pgtype.Timestamptz{}, // created_at
		}
		for i, d := range dest {
			if i >= len(vals) {
				break
			}
			switch ptr := d.(type) {
			case *pgtype.UUID:
				if v, ok := vals[i].(pgtype.UUID); ok {
					*ptr = v
				}
			case *string:
				if v, ok := vals[i].(string); ok {
					*ptr = v
				}
			case *[]byte:
				if v, ok := vals[i].([]byte); ok {
					*ptr = v
				}
			case *pgtype.Numeric:
				if v, ok := vals[i].(pgtype.Numeric); ok {
					*ptr = v
				}
			case *pgtype.Timestamptz:
				if v, ok := vals[i].(pgtype.Timestamptz); ok {
					*ptr = v
				}
			}
		}
		return nil
	}
}

// newTestMapper creates a HalaOSMapper backed by the given DBTX mock.
func newTestMapper(db sqlc.DBTX) *HalaOSMapper {
	logger := slog.Default()
	return NewHalaOSMapper(sqlc.New(db), logger)
}

// ---------------------------------------------------------------------------
// ResolveEmployee tests
// ---------------------------------------------------------------------------

// TestResolveEmployee_ExactMatch verifies Tier 1: GetEmployeeByHalaOSID succeeds.
func TestResolveEmployee_ExactMatch(t *testing.T) {
	empID := mapperTestUUID(0xBB)
	tenantID := mapperTestUUID(0xAA)

	db := &mapperMockDBTX{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		if strings.Contains(sql, "GetEmployeeByHalaOSID") {
			return &mapperMockRow{scanFn: employeeFullScanFn(empID)}
		}
		return &mapperMockRow{err: pgx.ErrNoRows}
	}

	mapper := newTestMapper(db)
	gotID, err := mapper.ResolveEmployee(context.Background(), tenantID, 42, "EMP001", "Test Employee")
	if err != nil {
		t.Fatalf("ResolveEmployee: unexpected error: %v", err)
	}
	if gotID != empID {
		t.Errorf("returned employee ID = %v, want %v", gotID, empID)
	}
}

// TestResolveEmployee_FallbackToNo verifies Tier 2: GetEmployeeByHalaOSID fails,
// GetEmployeeByHalaOSNo succeeds.
func TestResolveEmployee_FallbackToNo(t *testing.T) {
	empID := mapperTestUUID(0xCC)
	tenantID := mapperTestUUID(0xAA)

	db := &mapperMockDBTX{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		switch {
		case strings.Contains(sql, "GetEmployeeByHalaOSID"):
			return &mapperMockRow{err: pgx.ErrNoRows}
		case strings.Contains(sql, "GetEmployeeByHalaOSNo"):
			return &mapperMockRow{scanFn: employeeFullScanFn(empID)}
		}
		return &mapperMockRow{err: pgx.ErrNoRows}
	}
	// UpdateEmployeeHalaOSLink is called via Exec.
	db.execFn = func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}

	mapper := newTestMapper(db)
	gotID, err := mapper.ResolveEmployee(context.Background(), tenantID, 42, "EMP001", "Test Employee")
	if err != nil {
		t.Fatalf("ResolveEmployee: unexpected error: %v", err)
	}
	if gotID != empID {
		t.Errorf("returned employee ID = %v, want %v", gotID, empID)
	}
}

// TestResolveEmployee_AutoCreate verifies Tier 3: both lookups fail,
// CreateEmployeeFromHalaOS is called and the new employee ID is returned.
func TestResolveEmployee_AutoCreate(t *testing.T) {
	newEmpID := mapperTestUUID(0xDD)
	tenantID := mapperTestUUID(0xAA)

	db := &mapperMockDBTX{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		switch {
		case strings.Contains(sql, "GetEmployeeByHalaOSID"):
			return &mapperMockRow{err: pgx.ErrNoRows}
		case strings.Contains(sql, "GetEmployeeByHalaOSNo"):
			return &mapperMockRow{err: pgx.ErrNoRows}
		case strings.Contains(sql, "CreateEmployeeFromHalaOS"):
			return &mapperMockRow{scanFn: employeeFullScanFn(newEmpID)}
		}
		return &mapperMockRow{err: pgx.ErrNoRows}
	}

	mapper := newTestMapper(db)
	gotID, err := mapper.ResolveEmployee(context.Background(), tenantID, 99, "EMP099", "New Employee")
	if err != nil {
		t.Fatalf("ResolveEmployee: unexpected error: %v", err)
	}
	if gotID != newEmpID {
		t.Errorf("returned employee ID = %v, want %v", gotID, newEmpID)
	}
}

// ---------------------------------------------------------------------------
// MapRiskUpdated tests
// ---------------------------------------------------------------------------

// TestMapRiskUpdated_CreatesSignal verifies that MapRiskUpdated creates an
// execution_signal with signal_type="flight_risk" and the correct score.
func TestMapRiskUpdated_CreatesSignal(t *testing.T) {
	empID := mapperTestUUID(0xBB)
	tenantID := mapperTestUUID(0xAA)

	var capturedSignalType string
	var capturedScore pgtype.Numeric

	db := &mapperMockDBTX{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		switch {
		case strings.Contains(sql, "GetEmployeeByHalaOSID"):
			return &mapperMockRow{scanFn: employeeFullScanFn(empID)}
		case strings.Contains(sql, "CreateExecutionSignal"):
			// Capture the signal_type and score from the args.
			// Args order: tenant_id, subject_type, subject_id, signal_type,
			//             score, reasons, time_window
			if len(args) >= 7 {
				if v, ok := args[3].(string); ok {
					capturedSignalType = v
				}
				if v, ok := args[4].(pgtype.Numeric); ok {
					capturedScore = v
				}
			}
			return &mapperMockRow{scanFn: executionSignalFullScanFn(tenantID)}
		}
		return &mapperMockRow{err: pgx.ErrNoRows}
	}

	data, _ := json.Marshal(halaosRiskPayload{
		HRCompanyID:  1001,
		EmployeeID:   42,
		EmployeeNo:   "EMP001",
		EmployeeName: "Test Employee",
		RiskScore:    0.75,
		Factors: []halaosRiskFactor{
			{Factor: "low_engagement", Weight: 0.5},
		},
	})

	mapper := newTestMapper(db)
	if err := mapper.MapRiskUpdated(context.Background(), tenantID, json.RawMessage(data)); err != nil {
		t.Fatalf("MapRiskUpdated: unexpected error: %v", err)
	}

	if capturedSignalType != "flight_risk" {
		t.Errorf("signal_type = %q, want %q", capturedSignalType, "flight_risk")
	}
	// Verify score is valid (non-nil numeric).
	if !capturedScore.Valid {
		t.Error("expected a valid numeric score to be set")
	}
}

// TestMapRiskUpdated_InvalidJSON verifies that MapRiskUpdated returns an error
// on malformed JSON.
func TestMapRiskUpdated_InvalidJSON(t *testing.T) {
	tenantID := mapperTestUUID(0xAA)
	db := &mapperMockDBTX{}
	mapper := newTestMapper(db)

	err := mapper.MapRiskUpdated(context.Background(), tenantID, json.RawMessage(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// ---------------------------------------------------------------------------
// MapBurnoutUpdated tests
// ---------------------------------------------------------------------------

// TestMapBurnoutUpdated_CreatesSignal verifies signal_type="burnout_risk".
func TestMapBurnoutUpdated_CreatesSignal(t *testing.T) {
	empID := mapperTestUUID(0xBB)
	tenantID := mapperTestUUID(0xAA)

	var capturedSignalType string

	db := &mapperMockDBTX{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		switch {
		case strings.Contains(sql, "GetEmployeeByHalaOSID"):
			return &mapperMockRow{scanFn: employeeFullScanFn(empID)}
		case strings.Contains(sql, "CreateExecutionSignal"):
			if len(args) >= 4 {
				if v, ok := args[3].(string); ok {
					capturedSignalType = v
				}
			}
			return &mapperMockRow{scanFn: executionSignalFullScanFn(tenantID)}
		}
		return &mapperMockRow{err: pgx.ErrNoRows}
	}

	data, _ := json.Marshal(halaosBurnoutPayload{
		HRCompanyID:  1001,
		EmployeeID:   42,
		EmployeeNo:   "EMP001",
		EmployeeName: "Test Employee",
		BurnoutScore: 0.8,
		Factors:      []halaosRiskFactor{{Factor: "overwork", Weight: 0.9}},
	})

	mapper := newTestMapper(db)
	if err := mapper.MapBurnoutUpdated(context.Background(), tenantID, json.RawMessage(data)); err != nil {
		t.Fatalf("MapBurnoutUpdated: unexpected error: %v", err)
	}
	if capturedSignalType != "burnout_risk" {
		t.Errorf("signal_type = %q, want %q", capturedSignalType, "burnout_risk")
	}
}

// ---------------------------------------------------------------------------
// MapAttendanceUpdated tests
// ---------------------------------------------------------------------------

// TestMapAttendanceUpdated_CreatesEvent verifies event_type="attendance_anomaly".
func TestMapAttendanceUpdated_CreatesEvent(t *testing.T) {
	empID := mapperTestUUID(0xBB)
	tenantID := mapperTestUUID(0xAA)

	var capturedEventType string

	db := &mapperMockDBTX{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		switch {
		case strings.Contains(sql, "GetEmployeeByHalaOSID"):
			return &mapperMockRow{scanFn: employeeFullScanFn(empID)}
		case strings.Contains(sql, "CreateCommunicationEvent"):
			// args: tenant_id, source_type, source_id, platform, event_type,
			//       actor_id, target_id, payload, confidence, occurred_at
			if len(args) >= 5 {
				if v, ok := args[4].(string); ok {
					capturedEventType = v
				}
			}
			return &mapperMockRow{scanFn: communicationEventFullScanFn(tenantID)}
		}
		return &mapperMockRow{err: pgx.ErrNoRows}
	}

	data, _ := json.Marshal(halaosAttendancePayload{
		HRCompanyID: 1001,
		Anomalies: []halaosAttendanceAnomaly{
			{
				EmployeeID:   42,
				EmployeeNo:   "EMP001",
				EmployeeName: "Test Employee",
				Type:         "late_arrival",
				Detail:       "45 minutes late",
			},
		},
	})

	mapper := newTestMapper(db)
	if err := mapper.MapAttendanceUpdated(context.Background(), tenantID, json.RawMessage(data)); err != nil {
		t.Fatalf("MapAttendanceUpdated: unexpected error: %v", err)
	}
	if capturedEventType != "attendance_anomaly" {
		t.Errorf("event_type = %q, want %q", capturedEventType, "attendance_anomaly")
	}
}

// ---------------------------------------------------------------------------
// MapLeaveUpdated tests
// ---------------------------------------------------------------------------

// TestMapLeaveUpdated_CreatesEvent verifies event_type="leave_updated".
func TestMapLeaveUpdated_CreatesEvent(t *testing.T) {
	empID := mapperTestUUID(0xBB)
	tenantID := mapperTestUUID(0xAA)

	var capturedEventType string

	db := &mapperMockDBTX{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		switch {
		case strings.Contains(sql, "GetEmployeeByHalaOSID"):
			return &mapperMockRow{scanFn: employeeFullScanFn(empID)}
		case strings.Contains(sql, "CreateCommunicationEvent"):
			if len(args) >= 5 {
				if v, ok := args[4].(string); ok {
					capturedEventType = v
				}
			}
			return &mapperMockRow{scanFn: communicationEventFullScanFn(tenantID)}
		}
		return &mapperMockRow{err: pgx.ErrNoRows}
	}

	data, _ := json.Marshal(halaosLeavePayload{
		HRCompanyID:  1001,
		EmployeeID:   42,
		EmployeeNo:   "EMP001",
		EmployeeName: "Test Employee",
		LeaveType:    "vacation",
		StartDate:    "2026-04-01",
		EndDate:      "2026-04-05",
		Status:       "approved",
		Days:         5,
	})

	mapper := newTestMapper(db)
	if err := mapper.MapLeaveUpdated(context.Background(), tenantID, json.RawMessage(data)); err != nil {
		t.Fatalf("MapLeaveUpdated: unexpected error: %v", err)
	}
	if capturedEventType != "leave_updated" {
		t.Errorf("event_type = %q, want %q", capturedEventType, "leave_updated")
	}
}

// ---------------------------------------------------------------------------
// MapEmployeeUpdated tests
// ---------------------------------------------------------------------------

// TestMapEmployeeUpdated_CreatesEvent verifies event_type="employee_updated".
func TestMapEmployeeUpdated_CreatesEvent(t *testing.T) {
	empID := mapperTestUUID(0xBB)
	tenantID := mapperTestUUID(0xAA)

	var capturedEventType string

	db := &mapperMockDBTX{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		switch {
		case strings.Contains(sql, "GetEmployeeByHalaOSID"):
			return &mapperMockRow{scanFn: employeeFullScanFn(empID)}
		case strings.Contains(sql, "CreateCommunicationEvent"):
			if len(args) >= 5 {
				if v, ok := args[4].(string); ok {
					capturedEventType = v
				}
			}
			return &mapperMockRow{scanFn: communicationEventFullScanFn(tenantID)}
		}
		return &mapperMockRow{err: pgx.ErrNoRows}
	}

	data, _ := json.Marshal(halaosEmployeePayload{
		HRCompanyID:   1001,
		EmployeeID:    42,
		EmployeeNo:    "EMP001",
		EmployeeName:  "Test Employee",
		Department:    "Engineering",
		JobTitle:      "Senior Engineer",
		Status:        "active",
		ChangedFields: []string{"job_title"},
	})

	mapper := newTestMapper(db)
	if err := mapper.MapEmployeeUpdated(context.Background(), tenantID, json.RawMessage(data)); err != nil {
		t.Fatalf("MapEmployeeUpdated: unexpected error: %v", err)
	}
	if capturedEventType != "employee_updated" {
		t.Errorf("event_type = %q, want %q", capturedEventType, "employee_updated")
	}
}
