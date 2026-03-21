package memory

import (
	"context"
	"testing"
)

func TestMemoryEngine_RecallForMentor_NoEmbedder(t *testing.T) {
	engine := NewMemoryEngine(nil, nil, nil, nil, nil, nil)

	result, err := engine.RecallForMentor(context.Background(), "tenant-1", "emp-1", "How are you?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}
