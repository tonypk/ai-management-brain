package memory

import (
	"context"
	"testing"
)

type mockLLM struct {
	response string
	err      error
}

func (m *mockLLM) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return m.response, m.err
}

func TestExtractor_FromReport(t *testing.T) {
	llm := &mockLLM{
		response: `[{"content":"Employee mentioned project deadline pressure","type":"employee_insight","importance":0.7}]`,
	}
	embedder := &mockEmbedder{
		vec: []float32{0.1, 0.2, 0.3},
	}

	ext := NewExtractor(llm, embedder)

	memories, err := ext.FromReport(context.Background(), ReportInput{
		TenantID:   "tenant-1",
		EmployeeID: "emp-1",
		ReportID:   "report-1",
		Content:    "Today was stressful, project deadline is approaching fast.",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(memories) == 0 {
		t.Fatal("expected at least one memory extracted")
	}
	if memories[0].MemoryType != TypeEmployeeInsight {
		t.Errorf("expected %q, got %q", TypeEmployeeInsight, memories[0].MemoryType)
	}
	if memories[0].TenantID != "tenant-1" {
		t.Errorf("expected tenant-1, got %q", memories[0].TenantID)
	}
	if memories[0].SourceType != SourceReport {
		t.Errorf("expected %q, got %q", SourceReport, memories[0].SourceType)
	}
}

func TestExtractor_EmptyReport(t *testing.T) {
	llm := &mockLLM{response: "[]"}
	embedder := &mockEmbedder{vec: []float32{0.1, 0.2, 0.3}}

	ext := NewExtractor(llm, embedder)
	memories, err := ext.FromReport(context.Background(), ReportInput{
		TenantID:   "tenant-1",
		EmployeeID: "emp-1",
		ReportID:   "report-1",
		Content:    "Nothing special today.",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(memories) != 0 {
		t.Errorf("expected 0 memories, got %d", len(memories))
	}
}

// mockEmbedder for testing — used by other test files in this package
type mockEmbedder struct {
	vec []float32
	err error
}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return m.vec, m.err
}

func (m *mockEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = m.vec
	}
	return result, m.err
}
