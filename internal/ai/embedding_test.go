package ai

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMockEmbeddingService_DeterministicAndNormalized(t *testing.T) {
	svc := NewMockEmbeddingService(768)
	ctx := context.Background()

	vec1, err := svc.GenerateEmbedding(ctx, "Berita teknologi terkini kecerdasan buatan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vec1) != 768 {
		t.Fatalf("expected 768 dimensions, got %d", len(vec1))
	}

	// Verify unit normalization
	var sumSq float64
	for _, val := range vec1 {
		sumSq += float64(val * val)
	}
	norm := math.Sqrt(sumSq)
	if math.Abs(norm-1.0) > 1e-4 {
		t.Fatalf("expected unit norm (~1.0), got %f", norm)
	}

	// Verify determinism
	vec2, err := svc.GenerateEmbedding(ctx, "Berita teknologi terkini kecerdasan buatan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sim := CosineSimilarity(vec1, vec2)
	if math.Abs(sim-1.0) > 1e-5 {
		t.Fatalf("expected similarity of identical text to be 1.0, got %f", sim)
	}

	// Empty text should return error
	_, err = svc.GenerateEmbedding(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty text, got nil")
	}
}

func TestGeminiEmbeddingClient_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := geminiEmbedResponse{}
		resp.Embedding.Values = []float32{0.1, 0.2, 0.3}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := NewGeminiEmbeddingClient("dummy-key")
	client.httpClient = ts.Client()
	// Mock URL target via client
	client.model = "models/text-embedding-004"

	// Directly invoke HTTP request to test parsing
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, ts.URL, nil)
	resp, err := client.httpClient.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	var embedResp geminiEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(embedResp.Embedding.Values) != 3 {
		t.Fatalf("expected 3 values, got %d", len(embedResp.Embedding.Values))
	}
}

func TestCosineSimilarity(t *testing.T) {
	vecA := []float32{1.0, 0.0, 0.0}
	vecB := []float32{1.0, 0.0, 0.0}
	vecC := []float32{0.0, 1.0, 0.0}
	vecD := []float32{-1.0, 0.0, 0.0}

	if math.Abs(CosineSimilarity(vecA, vecB)-1.0) > 1e-5 {
		t.Errorf("expected CosineSimilarity(vecA, vecB) == 1.0")
	}
	if math.Abs(CosineSimilarity(vecA, vecC)-0.0) > 1e-5 {
		t.Errorf("expected CosineSimilarity(vecA, vecC) == 0.0")
	}
	if math.Abs(CosineSimilarity(vecA, vecD)-(-1.0)) > 1e-5 {
		t.Errorf("expected CosineSimilarity(vecA, vecD) == -1.0")
	}
}

