package ai

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestGeminiEmbeddingClient_CandidateFailover(t *testing.T) {
	attempt := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if strings.Contains(r.URL.Path, "text-embedding-004") {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    404,
					"message": "models/text-embedding-004 is not found for API version v1beta",
				},
			})
			return
		}

		resp := geminiEmbedResponse{}
		// Return 1000 items to test 768-dimension truncation and normalization
		resp.Embedding.Values = make([]float32, 1000)
		for i := range resp.Embedding.Values {
			resp.Embedding.Values[i] = 1.0
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := NewGeminiEmbeddingClient("dummy-key")
	client.httpClient = ts.Client()
	client.candidates = []string{"models/text-embedding-004", "models/gemini-embedding-001"}
	client.model = "models/text-embedding-004"

	// Point client to test server using DefaultTransport to avoid recursion
	defaultTransport := http.DefaultTransport
	client.httpClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = ts.Listener.Addr().String()
		return defaultTransport.RoundTrip(req)
	})

	vec, err := client.GenerateEmbedding(context.Background(), "berita demo")
	if err != nil {
		t.Fatalf("expected candidate failover to succeed, got error: %v", err)
	}

	if len(vec) != 768 {
		t.Fatalf("expected vector to be resized to 768, got %d", len(vec))
	}

	if client.model != "models/gemini-embedding-001" {
		t.Fatalf("expected active model to switch to gemini-embedding-001, got %s", client.model)
	}
}

func TestResilientEmbeddingService_FallbackToMock(t *testing.T) {
	failingPrimary := &failingEmbeddingService{}
	mockFallback := NewMockEmbeddingService(768)

	resilient := NewResilientEmbeddingService(failingPrimary, mockFallback)

	vec, err := resilient.GenerateEmbedding(context.Background(), "berita ekonomi")
	if err != nil {
		t.Fatalf("expected resilient embedding to succeed with fallback, got: %v", err)
	}

	if len(vec) != 768 {
		t.Fatalf("expected 768 dimensions, got %d", len(vec))
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

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type failingEmbeddingService struct{}

func (f *failingEmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return nil, errors.New("gemini embedding api error (404): models/text-embedding-004 is not found")
}
