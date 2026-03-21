package memory

import (
	"testing"
	"time"
)

func TestFormatUUID(t *testing.T) {
	id := "550e8400-e29b-41d4-a716-446655440000"
	u, err := parseUUID(id)
	if err != nil {
		t.Fatalf("parse UUID: %v", err)
	}
	got := formatUUID(u)
	if got != id {
		t.Errorf("expected %q, got %q", id, got)
	}
}

func TestFormatUUID_Empty(t *testing.T) {
	id := ""
	_, err := parseUUID(id)
	if err == nil {
		t.Fatal("expected error for empty UUID")
	}
}

func TestMemoryFromRow(t *testing.T) {
	now := time.Now()
	m := Memory{
		ID:         "test-id",
		TenantID:   "tenant-id",
		MemoryType: TypeEmployeeInsight,
		MemoryTier: TierShortTerm,
		Content:    "test content",
		Importance: 0.7,
		CreatedAt:  now,
	}

	if m.MemoryType != TypeEmployeeInsight {
		t.Errorf("expected %q, got %q", TypeEmployeeInsight, m.MemoryType)
	}
	if m.MemoryTier != TierShortTerm {
		t.Errorf("expected %q, got %q", TierShortTerm, m.MemoryTier)
	}
}
