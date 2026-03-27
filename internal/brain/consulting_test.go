package brain

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// ---------------------------------------------------------------------------
// Minimal mock DBTX for consulting tests.
// ---------------------------------------------------------------------------

// consultMockDBTX implements sqlc.DBTX with function fields for individual
// tests to install their own behaviour.
type consultMockDBTX struct {
	queryRowFn func(ctx context.Context, sql string, args ...interface{}) pgx.Row
	queryFn    func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	execFn     func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

func (m *consultMockDBTX) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	if m.execFn != nil {
		return m.execFn(ctx, sql, args...)
	}
	return pgconn.NewCommandTag("OK"), nil
}

func (m *consultMockDBTX) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, sql, args...)
	}
	return &consultMockRows{done: true}, nil
}

func (m *consultMockDBTX) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	if m.queryRowFn != nil {
		return m.queryRowFn(ctx, sql, args...)
	}
	return &consultMockRow{err: pgx.ErrNoRows}
}

// consultMockRow implements pgx.Row.
type consultMockRow struct {
	err    error
	scanFn func(dest ...interface{}) error
}

func (r *consultMockRow) Scan(dest ...interface{}) error {
	if r.scanFn != nil {
		return r.scanFn(dest...)
	}
	return r.err
}

// consultMockRows implements pgx.Rows for ListActiveEmployees tests.
type consultMockRows struct {
	employees []sqlc.Employee
	index     int
	done      bool
}

func (r *consultMockRows) Close()                                       {}
func (r *consultMockRows) Err() error                                   { return nil }
func (r *consultMockRows) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("SELECT") }
func (r *consultMockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *consultMockRows) RawValues() [][]byte                          { return nil }
func (r *consultMockRows) Conn() *pgx.Conn                              { return nil }
func (r *consultMockRows) Values() ([]interface{}, error)               { return nil, nil }

func (r *consultMockRows) Next() bool {
	if r.done || r.index >= len(r.employees) {
		return false
	}
	return true
}

func (r *consultMockRows) Scan(dest ...interface{}) error {
	if r.index >= len(r.employees) {
		return pgx.ErrNoRows
	}
	emp := r.employees[r.index]
	r.index++

	// Employee has 26 fields matching the SELECT column order:
	// id, tenant_id, name, telegram_id, culture_code, role, invite_code,
	// is_active, created_at, signal_phone, slack_id, lark_id,
	// preferred_channel, job_title, responsibilities, country, language,
	// org_unit_id, execution_score, current_load, strengths, risk_flags,
	// work_scope, halaos_employee_id, halaos_employee_no
	vals := []interface{}{
		emp.ID,
		emp.TenantID,
		emp.Name,
		emp.TelegramID,
		emp.CultureCode,
		emp.Role,
		emp.InviteCode,
		emp.IsActive,
		emp.CreatedAt,
		emp.SignalPhone,
		emp.SlackID,
		emp.LarkID,
		emp.PreferredChannel,
		emp.JobTitle,
		emp.Responsibilities,
		emp.Country,
		emp.Language,
		emp.OrgUnitID,
		emp.ExecutionScore,
		emp.CurrentLoad,
		emp.Strengths,
		emp.RiskFlags,
		emp.WorkScope,
		emp.HalaosEmployeeID,
		emp.HalaosEmployeeNo,
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

// consultTestUUID creates a pgtype.UUID with all bytes set to the given value.
func consultTestUUID(b byte) pgtype.UUID {
	var u pgtype.UUID
	u.Valid = true
	for i := range u.Bytes {
		u.Bytes[i] = b
	}
	return u
}

// engagementFullScanFn returns a scanFn that populates all 19 Engagement fields.
func engagementFullScanFn(eng sqlc.Engagement) func(dest ...interface{}) error {
	return func(dest ...interface{}) error {
		vals := []interface{}{
			eng.ID,
			eng.TenantID,
			eng.Title,
			eng.ProblemStatement,
			eng.Tier,
			eng.Category,
			eng.Phase,
			eng.DiagnosisQuestions,
			eng.DiagnosisAnswers,
			eng.DiagnosisData,
			eng.Analysis,
			eng.Plan,
			eng.ProgressPct,
			eng.NextCheckAt,
			eng.MentorID,
			eng.CultureCode,
			eng.CreatedAt,
			eng.UpdatedAt,
			eng.ClosedAt,
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
			case *pgtype.Text:
				if v, ok := vals[i].(pgtype.Text); ok {
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

// ---------------------------------------------------------------------------
// TestConsultingTierMaxQuestions
// ---------------------------------------------------------------------------

func TestConsultingTierMaxQuestions(t *testing.T) {
	tests := []struct {
		tier string
		want int
	}{
		{"quick", 2},
		{"standard", 5},
		{"deep", 10},
		{"", 5},       // default falls to standard
		{"unknown", 5}, // unrecognised tier falls to standard
	}
	for _, tc := range tests {
		t.Run(tc.tier, func(t *testing.T) {
			got := tierMaxQuestions(tc.tier)
			if got != tc.want {
				t.Errorf("tierMaxQuestions(%q) = %d, want %d", tc.tier, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestConsultingStripMarkdownJSON
// ---------------------------------------------------------------------------

func TestConsultingStripMarkdownJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain JSON",
			input: `{"tier":"standard","category":"performance"}`,
			want:  `{"tier":"standard","category":"performance"}`,
		},
		{
			name:  "json code block",
			input: "```json\n{\"tier\":\"quick\"}\n```",
			want:  `{"tier":"quick"}`,
		},
		{
			name:  "plain code block",
			input: "```\n{\"key\":\"value\"}\n```",
			want:  `{"key":"value"}`,
		},
		{
			name:  "with leading/trailing whitespace",
			input: "  \n```json\n{\"a\":1}\n```\n  ",
			want:  `{"a":1}`,
		},
		{
			name:  "no code fence markers",
			input: "just some text",
			want:  "just some text",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "single line with backticks not a fence",
			input: "```only one line```",
			want:  "```only one line```",
		},
		{
			name:  "multi-line content inside fence",
			input: "```json\n{\"a\":1,\n\"b\":2}\n```",
			want:  "{\"a\":1,\n\"b\":2}",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := stripMarkdownJSON(tc.input)
			if got != tc.want {
				t.Errorf("stripMarkdownJSON(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestConsultingSafePercent
// ---------------------------------------------------------------------------

func TestConsultingSafePercent(t *testing.T) {
	tests := []struct {
		name       string
		done       int
		total      int
		wantResult float64
	}{
		{"zero total returns zero", 0, 0, 0},
		{"zero done returns zero", 0, 10, 0},
		{"half done", 5, 10, 50},
		{"all done", 10, 10, 100},
		{"partial", 3, 7, float64(3) / float64(7) * 100},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := safePercent(tc.done, tc.total)
			if got != tc.wantResult {
				t.Errorf("safePercent(%d, %d) = %f, want %f", tc.done, tc.total, got, tc.wantResult)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestConsultingFormatPlanText
// ---------------------------------------------------------------------------

func TestConsultingFormatPlanText(t *testing.T) {
	ce := &ConsultingEngine{}

	t.Run("full plan", func(t *testing.T) {
		plan := planGenerationResponse{
			Summary:          "Improve team velocity by 30%",
			ExpectedOutcomes: []string{"Higher throughput", "Better morale"},
			Timeline:         "4 weeks",
			Actions: []planAction{
				{
					Title:       "Implement daily standups",
					Priority:    "high",
					OwnerName:   "Alice",
					Description: "15-minute daily sync meetings",
				},
				{
					Title:       "Set up CI/CD pipeline",
					Priority:    "",
					OwnerName:   "",
					Description: "",
				},
			},
		}

		text := ce.formatPlanText(plan)

		if !strings.Contains(text, "CONSULTING PLAN") {
			t.Error("expected header 'CONSULTING PLAN'")
		}
		if !strings.Contains(text, "Improve team velocity by 30%") {
			t.Error("expected plan summary")
		}
		if !strings.Contains(text, "Higher throughput") {
			t.Error("expected first outcome")
		}
		if !strings.Contains(text, "Better morale") {
			t.Error("expected second outcome")
		}
		if !strings.Contains(text, "Timeline: 4 weeks") {
			t.Error("expected timeline")
		}
		if !strings.Contains(text, "Actions (2)") {
			t.Error("expected actions count header")
		}
		if !strings.Contains(text, "[HIGH] Implement daily standups") {
			t.Error("expected first action with HIGH priority")
		}
		if !strings.Contains(text, "Owner: Alice") {
			t.Error("expected owner Alice")
		}
		if !strings.Contains(text, "15-minute daily sync meetings") {
			t.Error("expected action description")
		}
		// Second action should default to MEDIUM priority and unassigned owner
		if !strings.Contains(text, "[MEDIUM] Set up CI/CD pipeline") {
			t.Error("expected second action with default MEDIUM priority")
		}
		if !strings.Contains(text, "Owner: unassigned") {
			t.Error("expected default owner 'unassigned'")
		}
	})

	t.Run("empty plan", func(t *testing.T) {
		plan := planGenerationResponse{}
		text := ce.formatPlanText(plan)

		if !strings.Contains(text, "CONSULTING PLAN") {
			t.Error("expected header even for empty plan")
		}
		if strings.Contains(text, "Expected Outcomes") {
			t.Error("should not show outcomes section for empty outcomes")
		}
		if strings.Contains(text, "Timeline") {
			t.Error("should not show timeline section for empty timeline")
		}
		if strings.Contains(text, "Actions") {
			t.Error("should not show actions section for empty actions")
		}
	})

	t.Run("plan with outcomes but no actions", func(t *testing.T) {
		plan := planGenerationResponse{
			Summary:          "Assessment only",
			ExpectedOutcomes: []string{"Clarity on next steps"},
			Timeline:         "1 week",
		}
		text := ce.formatPlanText(plan)

		if !strings.Contains(text, "Clarity on next steps") {
			t.Error("expected outcomes")
		}
		if !strings.Contains(text, "Timeline: 1 week") {
			t.Error("expected timeline")
		}
		if strings.Contains(text, "Actions") {
			t.Error("should not show actions section for zero actions")
		}
	})
}

// ---------------------------------------------------------------------------
// TestConsultingBuildTeamList
// ---------------------------------------------------------------------------

func TestConsultingBuildTeamList(t *testing.T) {
	tenantID := consultTestUUID(0xAA)

	t.Run("with employees", func(t *testing.T) {
		employees := []sqlc.Employee{
			{
				ID:       consultTestUUID(0x01),
				TenantID: tenantID,
				Name:     "Alice",
				Role:     "engineer",
				IsActive: true,
			},
			{
				ID:       consultTestUUID(0x02),
				TenantID: tenantID,
				Name:     "Bob",
				Role:     "designer",
				IsActive: true,
			},
		}

		db := &consultMockDBTX{}
		db.queryFn = func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			if strings.Contains(sql, "is_active = true") {
				return &consultMockRows{employees: employees}, nil
			}
			return &consultMockRows{done: true}, nil
		}

		queries := sqlc.New(db)
		ce := &ConsultingEngine{queries: queries}

		result := ce.buildTeamList(context.Background(), tenantID)

		if !strings.Contains(result, "Alice (engineer)") {
			t.Errorf("expected 'Alice (engineer)' in result, got: %s", result)
		}
		if !strings.Contains(result, "Bob (designer)") {
			t.Errorf("expected 'Bob (designer)' in result, got: %s", result)
		}
	})

	t.Run("employee with empty role defaults to team member", func(t *testing.T) {
		employees := []sqlc.Employee{
			{
				ID:       consultTestUUID(0x03),
				TenantID: tenantID,
				Name:     "Charlie",
				Role:     "",
				IsActive: true,
			},
		}

		db := &consultMockDBTX{}
		db.queryFn = func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			if strings.Contains(sql, "is_active = true") {
				return &consultMockRows{employees: employees}, nil
			}
			return &consultMockRows{done: true}, nil
		}

		queries := sqlc.New(db)
		ce := &ConsultingEngine{queries: queries}

		result := ce.buildTeamList(context.Background(), tenantID)

		if !strings.Contains(result, "Charlie (team member)") {
			t.Errorf("expected 'Charlie (team member)' in result, got: %s", result)
		}
	})

	t.Run("no employees returns fallback message", func(t *testing.T) {
		db := &consultMockDBTX{}
		db.queryFn = func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return &consultMockRows{done: true}, nil
		}

		queries := sqlc.New(db)
		ce := &ConsultingEngine{queries: queries}

		result := ce.buildTeamList(context.Background(), tenantID)

		if result != "No team data available." {
			t.Errorf("expected fallback message, got: %s", result)
		}
	})

	t.Run("query error returns fallback message", func(t *testing.T) {
		db := &consultMockDBTX{}
		db.queryFn = func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return nil, pgx.ErrNoRows
		}

		queries := sqlc.New(db)
		ce := &ConsultingEngine{queries: queries}

		result := ce.buildTeamList(context.Background(), tenantID)

		if result != "No team data available." {
			t.Errorf("expected fallback message on error, got: %s", result)
		}
	})
}

// ---------------------------------------------------------------------------
// TestConsultingReviewAction
// ---------------------------------------------------------------------------

func TestConsultingReviewAction(t *testing.T) {
	actionID := consultTestUUID(0xCC)

	t.Run("approve action succeeds", func(t *testing.T) {
		var capturedSQL string
		db := &consultMockDBTX{}
		db.execFn = func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			capturedSQL = sql
			return pgconn.NewCommandTag("UPDATE 1"), nil
		}

		queries := sqlc.New(db)
		ce := &ConsultingEngine{queries: queries}

		err := ce.ReviewAction(context.Background(), actionID, true)
		if err != nil {
			t.Fatalf("ReviewAction(approve): unexpected error: %v", err)
		}
		if !strings.Contains(capturedSQL, "approved") {
			t.Errorf("expected approve SQL, got: %s", capturedSQL)
		}
	})

	t.Run("reject action succeeds", func(t *testing.T) {
		var capturedSQL string
		db := &consultMockDBTX{}
		db.execFn = func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			capturedSQL = sql
			return pgconn.NewCommandTag("UPDATE 1"), nil
		}

		queries := sqlc.New(db)
		ce := &ConsultingEngine{queries: queries}

		err := ce.ReviewAction(context.Background(), actionID, false)
		if err != nil {
			t.Fatalf("ReviewAction(reject): unexpected error: %v", err)
		}
		if !strings.Contains(capturedSQL, "rejected") {
			t.Errorf("expected reject SQL, got: %s", capturedSQL)
		}
	})

	t.Run("approve action db error", func(t *testing.T) {
		db := &consultMockDBTX{}
		db.execFn = func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, pgx.ErrNoRows
		}

		queries := sqlc.New(db)
		ce := &ConsultingEngine{queries: queries}

		err := ce.ReviewAction(context.Background(), actionID, true)
		if err == nil {
			t.Fatal("expected error on DB failure, got nil")
		}
		if !strings.Contains(err.Error(), "consulting: approve action") {
			t.Errorf("expected wrapped error message, got: %v", err)
		}
	})

	t.Run("reject action db error", func(t *testing.T) {
		db := &consultMockDBTX{}
		db.execFn = func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, pgx.ErrNoRows
		}

		queries := sqlc.New(db)
		ce := &ConsultingEngine{queries: queries}

		err := ce.ReviewAction(context.Background(), actionID, false)
		if err == nil {
			t.Fatal("expected error on DB failure, got nil")
		}
		if !strings.Contains(err.Error(), "consulting: reject action") {
			t.Errorf("expected wrapped error message, got: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TestConsultingStartEngagement — tests the error path when LLM is nil.
// ---------------------------------------------------------------------------

func TestConsultingStartEngagement_NilLLM(t *testing.T) {
	tenantID := consultTestUUID(0xAA)

	db := &consultMockDBTX{}
	queries := sqlc.New(db)
	// contextService is nil — StartEngagement handles this with a warning and
	// falls back to contextData = "{}".
	// llm is nil — Chat call will panic, so we verify the engine propagates
	// errors correctly.
	ce := &ConsultingEngine{
		queries:        queries,
		contextService: nil,
		llm:            nil,
	}

	// With nil llm, StartEngagement should panic or return an error.
	// We use a recover to verify it doesn't silently succeed.
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Expected: nil pointer dereference on ce.llm.Chat
				t.Logf("StartEngagement correctly panicked with nil LLM: %v", r)
			}
		}()

		_, _, err := ce.StartEngagement(context.Background(), tenantID, "test problem", "inamori", "default")
		if err != nil {
			// If it returned an error instead of panicking, that's also acceptable.
			t.Logf("StartEngagement returned error with nil LLM: %v", err)
			return
		}
		t.Error("expected panic or error with nil LLM, but got neither")
	}()
}

// ---------------------------------------------------------------------------
// TestConsultingAnswerQuestion — tests the error path when engagement
// does not exist.
// ---------------------------------------------------------------------------

func TestConsultingAnswerQuestion_EngagementNotFound(t *testing.T) {
	engagementID := consultTestUUID(0xBB)

	db := &consultMockDBTX{}
	// GetEngagement will fail with ErrNoRows since queryRowFn returns default.
	queries := sqlc.New(db)
	ce := &ConsultingEngine{queries: queries}

	_, _, _, err := ce.AnswerQuestion(context.Background(), engagementID, "my answer")
	if err == nil {
		t.Fatal("expected error for non-existent engagement, got nil")
	}
	if !strings.Contains(err.Error(), "consulting: get engagement") {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestConsultingConsultParseUUID
// ---------------------------------------------------------------------------

func TestConsultingConsultParseUUID(t *testing.T) {
	t.Run("valid UUID", func(t *testing.T) {
		u, err := consultParseUUID("01020304-0506-0708-090a-0b0c0d0e0f10")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !u.Valid {
			t.Error("expected valid UUID")
		}
	})

	t.Run("invalid UUID", func(t *testing.T) {
		_, err := consultParseUUID("not-a-uuid")
		if err == nil {
			t.Fatal("expected error for invalid UUID")
		}
	})

	t.Run("empty string", func(t *testing.T) {
		_, err := consultParseUUID("")
		if err == nil {
			t.Fatal("expected error for empty string")
		}
	})
}
