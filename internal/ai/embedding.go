package ai

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

// EmbeddingService generates dense vector embeddings for semantic search.
type EmbeddingService interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// GeminiEmbeddingClient integrates with Google AI Studio text-embedding-004.
type GeminiEmbeddingClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

type geminiEmbedRequest struct {
	Model   string       `json:"model"`
	Content geminiEmbedContent `json:"content"`
}

type geminiEmbedContent struct {
	Parts []geminiEmbedPart `json:"parts"`
}

type geminiEmbedPart struct {
	Text string `json:"text"`
}

type geminiEmbedResponse struct {
	Embedding struct {
		Values []float32 `json:"values"`
	} `json:"embedding"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

// NewGeminiEmbeddingClient creates an embedding client using Google AI Studio.
func NewGeminiEmbeddingClient(apiKey string) *GeminiEmbeddingClient {
	return &GeminiEmbeddingClient{
		apiKey: apiKey,
		model:  "models/text-embedding-004",
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *GeminiEmbeddingClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s:embedContent?key=%s", c.model, c.apiKey)
	reqBody := geminiEmbedRequest{
		Model: c.model,
		Content: geminiEmbedContent{
			Parts: []geminiEmbedPart{{Text: text}},
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embed request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	var embedResp geminiEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode gemini embedding response: %w", err)
	}

	if embedResp.Error != nil && embedResp.Error.Message != "" {
		return nil, fmt.Errorf("gemini embedding api error (%d): %s", embedResp.Error.Code, embedResp.Error.Message)
	}

	if len(embedResp.Embedding.Values) == 0 {
		return nil, fmt.Errorf("empty embedding returned from gemini api")
	}

	return embedResp.Embedding.Values, nil
}

// MockEmbeddingService creates deterministic normalized vector embeddings (default 768 dimensions)
// for testing and fallback environments without live network access.
type MockEmbeddingService struct {
	Dimension int
}

func NewMockEmbeddingService(dimension ...int) *MockEmbeddingService {
	dim := 768
	if len(dimension) > 0 && dimension[0] > 0 {
		dim = dimension[0]
	}
	return &MockEmbeddingService{Dimension: dim}
}

func (m *MockEmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Produce deterministic pseudo-random vector based on SHA256 of text
	hash := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(text))))
	seed := int64(binary.BigEndian.Uint64(hash[:8]))
	rng := rand.New(rand.NewSource(seed))

	vec := make([]float32, m.Dimension)
	var sumSq float64
	for i := 0; i < m.Dimension; i++ {
		val := rng.Float32()*2.0 - 1.0
		vec[i] = val
		sumSq += float64(val * val)
	}

	// Normalize vector to unit length
	norm := math.Sqrt(sumSq)
	if norm > 0 {
		for i := 0; i < m.Dimension; i++ {
			vec[i] = float32(float64(vec[i]) / norm)
		}
	}

	return vec, nil
}

// NewDefaultEmbeddingService initializes GeminiEmbeddingClient if GEMINI_API_KEY is present,
// otherwise returns MockEmbeddingService.
func NewDefaultEmbeddingService() EmbeddingService {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey != "" && !strings.EqualFold(apiKey, "mock") {
		return NewGeminiEmbeddingClient(apiKey)
	}
	return NewMockEmbeddingService(768)
}

// CosineSimilarity computes cosine similarity between two float32 vectors.
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

