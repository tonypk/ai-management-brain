package brain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	sqlc "github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// HalaOSMapper maps HalaOS webhook events to Brain execution_signals and
// communication_events table rows.
//
// It implements the halaosMapper interface defined in internal/api/halaos_webhook.go.
type HalaOSMapper struct {
	queries *sqlc.Queries
	logger  *slog.Logger
}

// NewHalaOSMapper creates a new HalaOSMapper.
func NewHalaOSMapper(q *sqlc.Queries, logger *slog.Logger) *HalaOSMapper {
	return &HalaOSMapper{queries: q, logger: logger}
}

// ---------------------------------------------------------------------------
// ResolveEmployee resolves a HalaOS employee to a Brain employee UUID using
// a 3-tier matching strategy:
//  1. Match by halaos_employee_id
//  2. Match by halaos_employee_no, then back-fill halaos_employee_id
//  3. Auto-create the employee record
//
// ---------------------------------------------------------------------------
func (m *HalaOSMapper) ResolveEmployee(
	ctx context.Context,
	tenantID pgtype.UUID,
	halaosEmpID int64,
	halaosEmpNo, name string,
) (pgtype.UUID, error) {
	// Tier 1 – match by numeric HalaOS employee ID.
	emp, err := m.queries.GetEmployeeByHalaOSID(ctx, sqlc.GetEmployeeByHalaOSIDParams{
		TenantID:         tenantID,
		HalaosEmployeeID: pgtype.Int8{Int64: halaosEmpID, Valid: halaosEmpID != 0},
	})
	if err == nil {
		return emp.ID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return pgtype.UUID{}, fmt.Errorf("resolve employee by id: %w", err)
	}

	// Tier 2 – match by employee number string, then update the HalaOS ID link.
	if halaosEmpNo != "" {
		emp, err = m.queries.GetEmployeeByHalaOSNo(ctx, sqlc.GetEmployeeByHalaOSNoParams{
			TenantID:         tenantID,
			HalaosEmployeeNo: pgtype.Text{String: halaosEmpNo, Valid: true},
		})
		if err == nil {
			if halaosEmpID != 0 {
				_ = m.queries.UpdateEmployeeHalaOSLink(ctx, sqlc.UpdateEmployeeHalaOSLinkParams{
					ID:               emp.ID,
					HalaosEmployeeID: pgtype.Int8{Int64: halaosEmpID, Valid: true},
					HalaosEmployeeNo: pgtype.Text{String: halaosEmpNo, Valid: true},
				})
			}
			return emp.ID, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return pgtype.UUID{}, fmt.Errorf("resolve employee by no: %w", err)
		}
	}

	// Tier 3 – auto-create an employee record from HalaOS data.
	newEmp, err := m.queries.CreateEmployeeFromHalaOS(ctx, sqlc.CreateEmployeeFromHalaOSParams{
		TenantID:         tenantID,
		Name:             name,
		HalaosEmployeeID: pgtype.Int8{Int64: halaosEmpID, Valid: halaosEmpID != 0},
		HalaosEmployeeNo: pgtype.Text{String: halaosEmpNo, Valid: halaosEmpNo != ""},
	})
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("create employee from halaos: %w", err)
	}
	m.logger.Info("halaos_mapper: auto-created employee",
		"name", name,
		"halaos_employee_id", halaosEmpID,
		"brain_employee_id", formatUUID(newEmp.ID),
	)
	return newEmp.ID, nil
}

// ---------------------------------------------------------------------------
// halaosRiskPayload is the data section of a "hr.risk.updated" event.
// ---------------------------------------------------------------------------
type halaosRiskPayload struct {
	HRCompanyID    int64             `json:"hr_company_id"`
	EmployeeID     int64             `json:"employee_id"`
	EmployeeNo     string            `json:"employee_no"`
	EmployeeName   string            `json:"employee_name"`
	RiskScore      float64           `json:"risk_score"`
	Factors        []halaosRiskFactor `json:"factors"`
}

type halaosRiskFactor struct {
	Factor string `json:"factor"`
	Weight float64 `json:"weight"`
}

// MapRiskUpdated maps a "hr.risk.updated" event to an execution_signal row
// with signal_type "flight_risk".
func (m *HalaOSMapper) MapRiskUpdated(ctx context.Context, tenantID pgtype.UUID, data json.RawMessage) error {
	var p halaosRiskPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("MapRiskUpdated: parse payload: %w", err)
	}

	empID, err := m.ResolveEmployee(ctx, tenantID, p.EmployeeID, p.EmployeeNo, p.EmployeeName)
	if err != nil {
		return fmt.Errorf("MapRiskUpdated: resolve employee: %w", err)
	}

	factors := make([]string, 0, len(p.Factors))
	for _, f := range p.Factors {
		if f.Factor != "" {
			factors = append(factors, f.Factor)
		}
	}
	reasons, _ := json.Marshal(factors)

	var score pgtype.Numeric
	_ = score.Scan(fmt.Sprintf("%.4f", p.RiskScore))

	_, err = m.queries.CreateExecutionSignal(ctx, sqlc.CreateExecutionSignalParams{
		TenantID:    tenantID,
		SubjectType: "employee",
		SubjectID:   empID,
		SignalType:  "flight_risk",
		Score:       score,
		Reasons:     reasons,
		TimeWindow:  pgtype.Text{String: "30d", Valid: true},
	})
	if err != nil {
		return fmt.Errorf("MapRiskUpdated: create execution signal: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// halaosRiskPayload re-used for burnout (different score field name).
// ---------------------------------------------------------------------------
type halaosBurnoutPayload struct {
	HRCompanyID  int64              `json:"hr_company_id"`
	EmployeeID   int64              `json:"employee_id"`
	EmployeeNo   string             `json:"employee_no"`
	EmployeeName string             `json:"employee_name"`
	BurnoutScore float64            `json:"burnout_score"`
	Factors      []halaosRiskFactor `json:"factors"`
}

// MapBurnoutUpdated maps a "hr.burnout.updated" event to an execution_signal row
// with signal_type "burnout_risk".
func (m *HalaOSMapper) MapBurnoutUpdated(ctx context.Context, tenantID pgtype.UUID, data json.RawMessage) error {
	var p halaosBurnoutPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("MapBurnoutUpdated: parse payload: %w", err)
	}

	empID, err := m.ResolveEmployee(ctx, tenantID, p.EmployeeID, p.EmployeeNo, p.EmployeeName)
	if err != nil {
		return fmt.Errorf("MapBurnoutUpdated: resolve employee: %w", err)
	}

	factors := make([]string, 0, len(p.Factors))
	for _, f := range p.Factors {
		if f.Factor != "" {
			factors = append(factors, f.Factor)
		}
	}
	reasons, _ := json.Marshal(factors)

	var score pgtype.Numeric
	_ = score.Scan(fmt.Sprintf("%.4f", p.BurnoutScore))

	_, err = m.queries.CreateExecutionSignal(ctx, sqlc.CreateExecutionSignalParams{
		TenantID:    tenantID,
		SubjectType: "employee",
		SubjectID:   empID,
		SignalType:  "burnout_risk",
		Score:       score,
		Reasons:     reasons,
		TimeWindow:  pgtype.Text{String: "30d", Valid: true},
	})
	if err != nil {
		return fmt.Errorf("MapBurnoutUpdated: create execution signal: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// halaosAttendancePayload is the data section of a "hr.attendance.updated" event.
// ---------------------------------------------------------------------------
type halaosAttendancePayload struct {
	HRCompanyID int64                   `json:"hr_company_id"`
	Anomalies   []halaosAttendanceAnomaly `json:"anomalies"`
}

type halaosAttendanceAnomaly struct {
	EmployeeID   int64  `json:"employee_id"`
	EmployeeNo   string `json:"employee_no"`
	EmployeeName string `json:"employee_name"`
	Type         string `json:"type"`
	Detail       string `json:"detail"`
}

// MapAttendanceUpdated maps a "hr.attendance.updated" event to one
// communication_event row per anomaly.
func (m *HalaOSMapper) MapAttendanceUpdated(ctx context.Context, tenantID pgtype.UUID, data json.RawMessage) error {
	var p halaosAttendancePayload
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("MapAttendanceUpdated: parse payload: %w", err)
	}

	now := time.Now()
	var firstErr error

	for _, anomaly := range p.Anomalies {
		empID, err := m.ResolveEmployee(ctx, tenantID, anomaly.EmployeeID, anomaly.EmployeeNo, anomaly.EmployeeName)
		if err != nil {
			m.logger.Error("halaos_mapper: resolve employee for attendance anomaly",
				"employee_id", anomaly.EmployeeID, "error", err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		payloadMap := map[string]string{
			"type":   anomaly.Type,
			"detail": anomaly.Detail,
		}
		payloadBytes, _ := json.Marshal(payloadMap)

		var confidence pgtype.Numeric
		_ = confidence.Scan("1.0")

		_, err = m.queries.CreateCommunicationEvent(ctx, sqlc.CreateCommunicationEventParams{
			TenantID:   tenantID,
			SourceType: "halaos",
			SourceID:   pgtype.UUID{},
			Platform:   "halaos",
			EventType:  "attendance_anomaly",
			ActorID:    empID,
			TargetID:   pgtype.UUID{},
			Payload:    payloadBytes,
			Confidence: confidence,
			OccurredAt: pgtype.Timestamptz{Time: now, Valid: true},
		})
		if err != nil {
			m.logger.Error("halaos_mapper: create communication event for attendance anomaly",
				"employee_id", anomaly.EmployeeID, "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

// ---------------------------------------------------------------------------
// halaosLeavePayload is the data section of a "hr.leave.updated" event.
// ---------------------------------------------------------------------------
type halaosLeavePayload struct {
	HRCompanyID  int64   `json:"hr_company_id"`
	EmployeeID   int64   `json:"employee_id"`
	EmployeeNo   string  `json:"employee_no"`
	EmployeeName string  `json:"employee_name"`
	LeaveType    string  `json:"leave_type"`
	StartDate    string  `json:"start_date"`
	EndDate      string  `json:"end_date"`
	Status       string  `json:"status"`
	Days         float64 `json:"days"`
}

// MapLeaveUpdated maps a "hr.leave.updated" event to a communication_event row.
func (m *HalaOSMapper) MapLeaveUpdated(ctx context.Context, tenantID pgtype.UUID, data json.RawMessage) error {
	var p halaosLeavePayload
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("MapLeaveUpdated: parse payload: %w", err)
	}

	empID, err := m.ResolveEmployee(ctx, tenantID, p.EmployeeID, p.EmployeeNo, p.EmployeeName)
	if err != nil {
		return fmt.Errorf("MapLeaveUpdated: resolve employee: %w", err)
	}

	payloadMap := map[string]interface{}{
		"leave_type": p.LeaveType,
		"start_date": p.StartDate,
		"end_date":   p.EndDate,
		"status":     p.Status,
		"days":       p.Days,
	}
	payloadBytes, _ := json.Marshal(payloadMap)

	var confidence pgtype.Numeric
	_ = confidence.Scan("1.0")

	_, err = m.queries.CreateCommunicationEvent(ctx, sqlc.CreateCommunicationEventParams{
		TenantID:   tenantID,
		SourceType: "halaos",
		SourceID:   pgtype.UUID{},
		Platform:   "halaos",
		EventType:  "leave_updated",
		ActorID:    empID,
		TargetID:   pgtype.UUID{},
		Payload:    payloadBytes,
		Confidence: confidence,
		OccurredAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("MapLeaveUpdated: create communication event: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// halaosPayrollPayload is the data section of a "hr.payroll.updated" event.
// ---------------------------------------------------------------------------
type halaosPayrollPayload struct {
	HRCompanyID  int64   `json:"hr_company_id"`
	EmployeeID   int64   `json:"employee_id"`
	EmployeeNo   string  `json:"employee_no"`
	EmployeeName string  `json:"employee_name"`
	Period       string  `json:"period"`
	GrossPay     float64 `json:"gross_pay"`
	NetPay       float64 `json:"net_pay"`
	Currency     string  `json:"currency"`
	Status       string  `json:"status"`
}

// MapPayrollUpdated maps a "hr.payroll.updated" event to a communication_event row.
func (m *HalaOSMapper) MapPayrollUpdated(ctx context.Context, tenantID pgtype.UUID, data json.RawMessage) error {
	var p halaosPayrollPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("MapPayrollUpdated: parse payload: %w", err)
	}

	empID, err := m.ResolveEmployee(ctx, tenantID, p.EmployeeID, p.EmployeeNo, p.EmployeeName)
	if err != nil {
		return fmt.Errorf("MapPayrollUpdated: resolve employee: %w", err)
	}

	payloadMap := map[string]interface{}{
		"period":    p.Period,
		"gross_pay": p.GrossPay,
		"net_pay":   p.NetPay,
		"currency":  p.Currency,
		"status":    p.Status,
	}
	payloadBytes, _ := json.Marshal(payloadMap)

	var confidence pgtype.Numeric
	_ = confidence.Scan("1.0")

	_, err = m.queries.CreateCommunicationEvent(ctx, sqlc.CreateCommunicationEventParams{
		TenantID:   tenantID,
		SourceType: "halaos",
		SourceID:   pgtype.UUID{},
		Platform:   "halaos",
		EventType:  "payroll_updated",
		ActorID:    empID,
		TargetID:   pgtype.UUID{},
		Payload:    payloadBytes,
		Confidence: confidence,
		OccurredAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("MapPayrollUpdated: create communication event: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// halaosEmployeePayload is the data section of a "hr.employee.updated" event.
// ---------------------------------------------------------------------------
type halaosEmployeePayload struct {
	HRCompanyID  int64   `json:"hr_company_id"`
	EmployeeID   int64   `json:"employee_id"`
	EmployeeNo   string  `json:"employee_no"`
	EmployeeName string  `json:"employee_name"`
	Department   string  `json:"department"`
	JobTitle     string  `json:"job_title"`
	Status       string  `json:"status"`
	ChangedFields []string `json:"changed_fields"`
}

// MapEmployeeUpdated maps a "hr.employee.updated" event to a communication_event row.
// It also ensures the Brain employee record is linked to the HalaOS employee identifiers.
func (m *HalaOSMapper) MapEmployeeUpdated(ctx context.Context, tenantID pgtype.UUID, data json.RawMessage) error {
	var p halaosEmployeePayload
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("MapEmployeeUpdated: parse payload: %w", err)
	}

	empID, err := m.ResolveEmployee(ctx, tenantID, p.EmployeeID, p.EmployeeNo, p.EmployeeName)
	if err != nil {
		return fmt.Errorf("MapEmployeeUpdated: resolve employee: %w", err)
	}

	payloadMap := map[string]interface{}{
		"department":     p.Department,
		"job_title":      p.JobTitle,
		"status":         p.Status,
		"changed_fields": p.ChangedFields,
	}
	payloadBytes, _ := json.Marshal(payloadMap)

	var confidence pgtype.Numeric
	_ = confidence.Scan("1.0")

	_, err = m.queries.CreateCommunicationEvent(ctx, sqlc.CreateCommunicationEventParams{
		TenantID:   tenantID,
		SourceType: "halaos",
		SourceID:   pgtype.UUID{},
		Platform:   "halaos",
		EventType:  "employee_updated",
		ActorID:    empID,
		TargetID:   pgtype.UUID{},
		Payload:    payloadBytes,
		Confidence: confidence,
		OccurredAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("MapEmployeeUpdated: create communication event: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// deptUUID produces a stable UUID v5 for a department, scoped to the tenant.
// ---------------------------------------------------------------------------
func deptUUID(tenantID pgtype.UUID, departmentName string) pgtype.UUID {
	key := formatUUID(tenantID) + ":dept:" + departmentName
	u := uuid.NewSHA1(uuid.NameSpaceDNS, []byte(key))
	var out pgtype.UUID
	copy(out.Bytes[:], u[:])
	out.Valid = true
	return out
}
