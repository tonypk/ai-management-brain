package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Embedder generates vector embeddings from text.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// HuggingFaceEmbedder calls the free HuggingFace Inference API for sentence embeddings.
// Default model: sentence-transformers/all-MiniLM-L6-v2 (384 dimensions).
type HuggingFaceEmbedder struct {
	model     string
	batchSize int
	baseURL   string
	client    *http.Client
}

func NewHuggingFaceEmbedder(model string, batchSize int) *HuggingFaceEmbedder {
	if model == "" {
		model = "sentence-transformers/all-MiniLM-L6-v2"
	}
	return &HuggingFaceEmbedder{
		model:     model,
		batchSize: batchSize,
		baseURL:   "https://api-inference.huggingface.co",
		client:    &http.Client{Timeout: 60 * time.Second},
	}
}

type hfRequest struct {
	Inputs  any            `json:"inputs"`
	Options map[string]any `json:"options,omitempty"`
}

func (e *HuggingFaceEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("empty input text")
	}

	vecs, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

func (e *HuggingFaceEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("empty input texts")
	}

	var allResults [][]float32

	for i := 0; i < len(texts); i += e.batchSize {
		end := i + e.batchSize
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[i:end]

		results, err := e.callAPI(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("batch %d: %w", i/e.batchSize, err)
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

func (e *HuggingFaceEmbedder) callAPI(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := hfRequest{
		Inputs:  texts,
		Options: map[string]any{"wait_for_model": true},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/pipeline/feature-extraction/%s", e.baseURL, e.model)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("huggingface API error %d: %s", resp.StatusCode, string(respBody))
	}

	// HuggingFace returns [][]float64 for sentence-transformers models
	var embeddings [][]float64
	if err := json.Unmarshal(respBody, &embeddings); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w (body: %s)", err, string(respBody))
	}

	results := make([][]float32, len(embeddings))
	for i, emb := range embeddings {
		vec := make([]float32, len(emb))
		for j, v := range emb {
			vec[j] = float32(v)
		}
		results[i] = vec
	}

	return results, nil
}
