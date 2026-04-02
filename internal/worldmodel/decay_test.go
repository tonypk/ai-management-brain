package worldmodel_test

import (
	"testing"

	"github.com/tonypk/ai-management-brain/internal/worldmodel"
)

func TestCalculateDecayedConfidence(t *testing.T) {
	tests := []struct {
		name         string
		base         float64
		daysSince    int
		mentionCount int
		wantMin      float64
		wantMax      float64
	}{
		{"fresh high skill", 0.80, 0, 5, 0.70, 0.95},
		{"1 week old", 0.80, 7, 3, 0.40, 0.80},
		{"3 months old single mention", 0.80, 90, 1, 0.01, 0.30},
		{"frequently mentioned", 0.80, 30, 10, 0.95, 1.00},
		{"minimum floor", 0.10, 365, 1, 0.05, 0.10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := worldmodel.CalculateDecayedConfidence(tt.base, tt.daysSince, tt.mentionCount)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("CalculateDecayedConfidence(%v, %d, %d) = %v, want [%v, %v]",
					tt.base, tt.daysSince, tt.mentionCount, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}
