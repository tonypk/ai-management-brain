package worldmodel

import (
	"context"
	"log/slog"
	"math"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// CalculateDecayedConfidence computes confidence with time decay and mention boost.
// Formula: base * 0.95^(daysSince/7) * min(mentionCount/5, 2.0)
// Minimum floor: 0.05
func CalculateDecayedConfidence(base float64, daysSince, mentionCount int) float64 {
	decayFactor := math.Pow(0.95, float64(daysSince)/7.0)
	mentionBoost := math.Min(float64(mentionCount)/5.0, 2.0)
	result := base * decayFactor * mentionBoost
	return math.Max(0.05, math.Min(1.0, result))
}

// DecayRunner runs confidence decay for a tenant.
type DecayRunner struct {
	q *sqlc.Queries
}

func NewDecayRunner(q *sqlc.Queries) *DecayRunner {
	return &DecayRunner{q: q}
}

// RunForAllTenants decays confidence for all tenants.
func (d *DecayRunner) RunForAllTenants(ctx context.Context) error {
	tenants, err := d.q.ListActiveTenants(ctx)
	if err != nil {
		return err
	}

	for _, t := range tenants {
		tid := t.ID
		if err := d.q.DecaySkillConfidence(ctx, tid); err != nil {
			slog.Error("decay skills", "tenant_id", formatUUID(tid), "error", err)
		}
		if err := d.q.DecayRelationshipStrength(ctx, tid); err != nil {
			slog.Error("decay relationships", "tenant_id", formatUUID(tid), "error", err)
		}
	}

	slog.Info("confidence decay completed", "tenants", len(tenants))
	return nil
}
