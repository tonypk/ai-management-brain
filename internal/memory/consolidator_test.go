package memory

import (
	"context"
	"testing"
	"time"
)

func TestConsolidator_CosineSimilarity(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{1, 0, 0}
	sim := cosineSimilarity(a, b)
	if sim < 0.99 {
		t.Errorf("identical vectors should have similarity ~1.0, got %f", sim)
	}

	c := []float32{0, 1, 0}
	sim = cosineSimilarity(a, c)
	if sim > 0.01 {
		t.Errorf("orthogonal vectors should have similarity ~0.0, got %f", sim)
	}
}

func TestConsolidator_ClusterMemories(t *testing.T) {
	memories := []Memory{
		{ID: "m1", Content: "Stressed about deadline", Embedding: []float32{0.9, 0.1, 0.0}},
		{ID: "m2", Content: "Worried about project timeline", Embedding: []float32{0.85, 0.15, 0.0}},
		{ID: "m3", Content: "Enjoys team meetings", Embedding: []float32{0.0, 0.1, 0.9}},
	}

	clusters := clusterMemories(memories, 0.8)

	found := false
	for _, cluster := range clusters {
		if len(cluster) == 2 {
			found = true
		}
	}
	if !found {
		t.Error("expected m1 and m2 to be clustered together")
	}
}

// Unused _ to reference time package
var _ = time.Now

// Unused _ to reference context package
var _ = context.Background
