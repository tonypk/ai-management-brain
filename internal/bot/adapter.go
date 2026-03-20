package bot

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// DBAdapter bridges sqlc.Queries to bot's CommandQuerier and IdentityQuerier interfaces.
type DBAdapter struct {
	q *sqlc.Queries
}

// NewDBAdapter creates a new DB adapter.
func NewDBAdapter(q *sqlc.Queries) *DBAdapter {
	return &DBAdapter{q: q}
}

// --- CommandQuerier ---

func (a *DBAdapter) GetTenantByBossChatID(ctx context.Context, bossChatID int64) (*Tenant, error) {
	t, err := a.q.GetTenantByBossChatID(ctx, bossChatID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return sqlcTenantToBot(t), nil
}

func (a *DBAdapter) CreateTenant(ctx context.Context, params CreateTenantParams) (*Tenant, error) {
	t, err := a.q.CreateTenant(ctx, sqlc.CreateTenantParams{
		Name:       params.Name,
		Timezone:   params.Timezone,
		MentorID:   params.MentorID,
		BossChatID: params.BossChatID,
		Config:     []byte("{}"),
	})
	if err != nil {
		return nil, err
	}
	return sqlcTenantToBot(t), nil
}

func (a *DBAdapter) ListEmployeesByTenant(ctx context.Context, tenantID string) ([]Employee, error) {
	uid, err := parseUUID(tenantID)
	if err != nil {
		return nil, err
	}
	emps, err := a.q.ListActiveEmployees(ctx, uid)
	if err != nil {
		return nil, err
	}
	result := make([]Employee, len(emps))
	for i, e := range emps {
		result[i] = *sqlcEmployeeToBot(e)
	}
	return result, nil
}

func (a *DBAdapter) CreateEmployee(ctx context.Context, params CreateEmployeeParams) (*Employee, error) {
	uid, err := parseUUID(params.TenantID)
	if err != nil {
		return nil, err
	}
	e, err := a.q.CreateEmployee(ctx, sqlc.CreateEmployeeParams{
		TenantID:    uid,
		Name:        params.Name,
		CultureCode: params.CultureCode,
		Role:        "member",
		InviteCode:  pgtype.Text{String: params.InviteCode, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	return sqlcEmployeeToBot(e), nil
}

func (a *DBAdapter) GetEmployeeByInviteCode(ctx context.Context, code string) (*Employee, error) {
	e, err := a.q.GetEmployeeByInviteCode(ctx, pgtype.Text{String: code, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return sqlcEmployeeToBot(e), nil
}

func (a *DBAdapter) UpdateEmployeeTelegramID(ctx context.Context, employeeID string, telegramID int64) error {
	uid, err := parseUUID(employeeID)
	if err != nil {
		return err
	}
	return a.q.UpdateEmployeeTelegramID(ctx, sqlc.UpdateEmployeeTelegramIDParams{
		ID:         uid,
		TelegramID: pgtype.Int8{Int64: telegramID, Valid: true},
	})
}

func (a *DBAdapter) UpdateTenantMentor(ctx context.Context, tenantID, mentorID string) error {
	uid, err := parseUUID(tenantID)
	if err != nil {
		return err
	}
	return a.q.UpdateTenantMentor(ctx, sqlc.UpdateTenantMentorParams{
		ID:       uid,
		MentorID: mentorID,
	})
}

// --- IdentityQuerier ---

func (a *DBAdapter) GetEmployeeByTelegramID(ctx context.Context, telegramID int64) (*Employee, error) {
	e, err := a.q.GetEmployeeByTelegramID(ctx, pgtype.Int8{Int64: telegramID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return sqlcEmployeeToBot(e), nil
}

// --- helpers ---

func sqlcTenantToBot(t sqlc.Tenant) *Tenant {
	return &Tenant{
		ID:         formatUUID(t.ID),
		Name:       t.Name,
		BossChatID: t.BossChatID,
		MentorID:   t.MentorID,
		Timezone:   t.Timezone,
	}
}

func sqlcEmployeeToBot(e sqlc.Employee) *Employee {
	emp := &Employee{
		ID:          formatUUID(e.ID),
		Name:        e.Name,
		TenantID:    formatUUID(e.TenantID),
		CultureCode: e.CultureCode,
	}
	if e.TelegramID.Valid {
		emp.TelegramID = e.TelegramID.Int64
	}
	if e.InviteCode.Valid {
		emp.InviteCode = e.InviteCode.String
	}
	return emp
}

func formatUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func parseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return u, fmt.Errorf("parse UUID %q: %w", s, err)
	}
	return u, nil
}
