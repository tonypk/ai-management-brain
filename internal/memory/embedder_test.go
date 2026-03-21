package memory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVoyageEmbedder_Embed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth header")
		}

		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)

		if req["model"] != "voyage-3-lite" {
			t.Errorf("unexpected model: %v", req["model"])
		}

		resp := map[string]any{
			"data": []map[string]any{
				{"embedding": []float64{0.1, 0.2, 0.3}},
			},
			"usage": map[string]any{"total_tokens": 10},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	embedder := NewVoyageEmbedder("test-key", "voyage-3-lite", 128)
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

func TestVoyageEmbedder_EmbedBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)

		inputs := req["input"].([]any)
		data := make([]map[string]any, len(inputs))
		for i := range inputs {
			data[i] = map[string]any{
				"embedding": []float64{float64(i) * 0.1, float64(i) * 0.2, float64(i) * 0.3},
			}
		}

		resp := map[string]any{
			"data":  data,
			"usage": map[string]any{"total_tokens": 20},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	embedder := NewVoyageEmbedder("test-key", "voyage-3-lite", 128)
	embedder.baseURL = server.URL

	vecs, err := embedder.EmbedBatch(context.Background(), []string{"hello", "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vecs) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(vecs))
	}
}

func TestVoyageEmbedder_EmptyInput(t *testing.T) {
	embedder := NewVoyageEmbedder("test-key", "voyage-3-lite", 128)

	_, err := embedder.Embed(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}
