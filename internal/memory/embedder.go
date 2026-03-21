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

// VoyageEmbedder calls the Voyage AI embeddings API.
type VoyageEmbedder struct {
	apiKey    string
	model     string
	batchSize int
	baseURL   string
	client    *http.Client
}

func NewVoyageEmbedder(apiKey, model string, batchSize int) *VoyageEmbedder {
	return &VoyageEmbedder{
		apiKey:    apiKey,
		model:     model,
		batchSize: batchSize,
		baseURL:   "https://api.voyageai.com",
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

type voyageRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type voyageResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

func (e *VoyageEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("empty input text")
	}

	vecs, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

func (e *VoyageEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
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

func (e *VoyageEmbedder) callAPI(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := voyageRequest{
		Input: texts,
		Model: e.model,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

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
		return nil, fmt.Errorf("voyage API error %d: %s", resp.StatusCode, string(respBody))
	}

	var voyageResp voyageResponse
	if err := json.Unmarshal(respBody, &voyageResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	results := make([][]float32, len(voyageResp.Data))
	for i, d := range voyageResp.Data {
		vec := make([]float32, len(d.Embedding))
		for j, v := range d.Embedding {
			vec[j] = float32(v)
		}
		results[i] = vec
	}

	return results, nil
}
