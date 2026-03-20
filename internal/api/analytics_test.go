package api

import (
	"testing"
)

func TestCalculateHealthScore_FullSubmission(t *testing.T) {
	sentiments := map[string]int{"positive": 8, "neutral": 2}
	score := calculateHealthScore(1.0, sentiments)
	// 40 (100% submission) + 32 (80% positive) + 4 (20% neutral) + 20 base = 96
	if score < 90 || score > 100 {
		t.Errorf("expected high score for full submission + positive, got %d", score)
	}
}

func TestCalculateHealthScore_NoSubmission(t *testing.T) {
	sentiments := map[string]int{}
	score := calculateHealthScore(0.0, sentiments)
	// 0 (0% submission) + 30 (no data neutral) + 20 base = 50
	if score != 50 {
		t.Errorf("expected 50 for no submissions, got %d", score)
	}
}

func TestCalculateHealthScore_MixedSentiment(t *testing.T) {
	sentiments := map[string]int{"positive": 3, "neutral": 3, "negative": 4}
	score := calculateHealthScore(0.5, sentiments)
	// 20 (50% submission) + 12 (30% positive) + 6 (30% neutral) + 20 base = 58
	if score < 50 || score > 65 {
		t.Errorf("expected moderate score, got %d", score)
	}
}

func TestSafeRate(t *testing.T) {
	if r := safeRate(5, 10); r != 0.5 {
		t.Errorf("expected 0.5, got %f", r)
	}
	if r := safeRate(0, 0); r != 0.0 {
		t.Errorf("expected 0.0 for zero total, got %f", r)
	}
}

func TestCalculateHealthScore_AllNegative(t *testing.T) {
	sentiments := map[string]int{"negative": 10}
	score := calculateHealthScore(0.5, sentiments)
	// 20 (50% submission) + 0 (0% positive, 0% neutral) + 20 base = 40
	if score != 40 {
		t.Errorf("expected 40 for all negative, got %d", score)
	}
}

func TestCalculateHealthScore_Cap100(t *testing.T) {
	sentiments := map[string]int{"positive": 10}
	score := calculateHealthScore(2.0, sentiments)
	if score != 100 {
		t.Errorf("expected 100 cap, got %d", score)
	}
}

func TestCalculateHealthScore_AllNeutral(t *testing.T) {
	sentiments := map[string]int{"neutral": 10}
	score := calculateHealthScore(1.0, sentiments)
	// 40 (100% submission) + 0 (positive) + 20 (100% neutral) + 20 base = 80
	if score != 80 {
		t.Errorf("expected 80 for all neutral, got %d", score)
	}
}

func TestCalculateHealthScore_Stressed(t *testing.T) {
	sentiments := map[string]int{"stressed": 10}
	score := calculateHealthScore(1.0, sentiments)
	// 40 (100% submission) + 0 (positive) + 0 (neutral) + 20 base = 60
	if score != 60 {
		t.Errorf("expected 60 for all stressed, got %d", score)
	}
}

func TestSafeRate_ZeroCount(t *testing.T) {
	if r := safeRate(0, 10); r != 0.0 {
		t.Errorf("expected 0.0 for zero count, got %f", r)
	}
}

func TestSafeRate_Equal(t *testing.T) {
	if r := safeRate(10, 10); r != 1.0 {
		t.Errorf("expected 1.0, got %f", r)
	}
}
