package memory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHuggingFaceEmbedder_Embed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var req hfRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Return sentence embeddings: [][]float64
		resp := [][]float64{
			{0.1, 0.2, 0.3},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	embedder := NewHuggingFaceEmbedder("test-model", 128)
	embedder.baseURL = server.URL

	vec, err := embedder.Embed(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vec) != 3 {
		t.Fatalf("expected 3 dims, got %d", len(vec))
	}
	if vec[0] != 0.1 || vec[1] != 0.2 || vec[2] != 0.3 {
		t.Errorf("unexpected values: %v", vec)
	}
}

func TestHuggingFaceEmbedder_EmbedBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req hfRequest
		json.NewDecoder(r.Body).Decode(&req)

		inputs := req.Inputs.([]any)
		data := make([][]float64, len(inputs))
		for i := range inputs {
			data[i] = []float64{float64(i) * 0.1, float64(i) * 0.2, float64(i) * 0.3}
		}

		json.NewEncoder(w).Encode(data)
	}))
	defer server.Close()

	embedder := NewHuggingFaceEmbedder("test-model", 128)
	embedder.baseURL = server.URL

	vecs, err := embedder.EmbedBatch(context.Background(), []string{"hello", "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vecs) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(vecs))
	}
}

func TestHuggingFaceEmbedder_EmptyInput(t *testing.T) {
	embedder := NewHuggingFaceEmbedder("test-model", 128)

	_, err := embedder.Embed(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}
