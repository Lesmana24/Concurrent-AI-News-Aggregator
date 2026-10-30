package ai

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// EmbeddingService generates dense vector embeddings for semantic search.
type EmbeddingService interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// GeminiEmbeddingClient integrates with Google AI Studio text embedding models (gemini-embedding-001 / gemini-embedding-2).
type GeminiEmbeddingClient struct {
	apiKey     string
	model      string
	candidates []string
	httpClient *http.Client
	mu         sync.RWMutex
}

type geminiEmbedRequest struct {
	Model                string             `json:"model"`
	Content              geminiEmbedContent `json:"content"`
	OutputDimensionality int                `json:"outputDimensionality,omitempty"`
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
// Automatically chooses the best working embedding model among supported Gemini models.
func NewGeminiEmbeddingClient(apiKey string) *GeminiEmbeddingClient {
	preferredModel := os.Getenv("GEMINI_EMBEDDING_MODEL")
	if preferredModel == "" {
		preferredModel = "models/gemini-embedding-001"
	} else if !strings.HasPrefix(preferredModel, "models/") {
		preferredModel = "models/" + preferredModel
	}

	candidates := []string{
		preferredModel,
		"models/gemini-embedding-001",
		"models/gemini-embedding-2",
		"models/text-embedding-004",
	}

	return &GeminiEmbeddingClient{
		apiKey:     apiKey,
		model:      preferredModel,
		candidates: candidates,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *GeminiEmbeddingClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	c.mu.RLock()
	currentModel := c.model
	candidates := make([]string, 0, len(c.candidates))
	candidates = append(candidates, currentModel)
	for _, cand := range c.candidates {
		if cand != currentModel {
			candidates = append(candidates, cand)
		}
	}
	c.mu.RUnlock()

	var lastErr error
	for _, modelName := range candidates {
		vec, err := c.tryEmbed(ctx, modelName, text)
		if err == nil {
			c.mu.Lock()
			if c.model != modelName {
				log.Printf("[Embedding] Switched Gemini embedding model to: %s", modelName)
				c.model = modelName
			}
			c.mu.Unlock()
			return vec, nil
		}

		lastErr = err
		// If 404 / model not found or unsupported method, try next candidate
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "404") || strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "not supported") {
			continue
		}
		// If non-model error (e.g. context canceled or fatal auth), return immediately
		break
	}

	return nil, lastErr
}

func (c *GeminiEmbeddingClient) tryEmbed(ctx context.Context, modelName, text string) ([]float32, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s:embedContent?key=%s", modelName, c.apiKey)
	reqBody := geminiEmbedRequest{
		Model:                modelName,
		OutputDimensionality: 768,
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

	vals := embedResp.Embedding.Values
	if len(vals) == 0 {
		return nil, fmt.Errorf("empty embedding returned from gemini api")
	}

	// Guarantee 768 dimensions for pgvector vector(768)
	if len(vals) > 768 {
		vals = vals[:768]
		// L2 re-normalization
		var sumSq float64
		for _, v := range vals {
			sumSq += float64(v * v)
		}
		norm := math.Sqrt(sumSq)
		if norm > 0 {
			for i := range vals {
				vals[i] = float32(float64(vals[i]) / norm)
			}
		}
	} else if len(vals) < 768 {
		padded := make([]float32, 768)
		copy(padded, vals)
		vals = padded
	}

	return vals, nil
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

// ResilientEmbeddingService wraps a primary and fallback embedding service.
// If the primary fails (e.g. rate limit, 404, or network issue), it transparently
// falls back to the deterministic secondary embedding service so queries never crash.
type ResilientEmbeddingService struct {
	primary  EmbeddingService
	fallback EmbeddingService
}

func NewResilientEmbeddingService(primary, fallback EmbeddingService) *ResilientEmbeddingService {
	return &ResilientEmbeddingService{
		primary:  primary,
		fallback: fallback,
	}
}

func (r *ResilientEmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if r.primary != nil {
		vec, err := r.primary.GenerateEmbedding(ctx, text)
		if err == nil && len(vec) == 768 {
			return vec, nil
		}
		log.Printf("[Embedding] Warning: primary embedding service failed (%v). Falling back to resilient local embedding.", err)
	}
	if r.fallback != nil {
		return r.fallback.GenerateEmbedding(ctx, text)
	}
	return nil, fmt.Errorf("no embedding service available")
}

// NewDefaultEmbeddingService initializes GeminiEmbeddingClient with Resilient fallback
// if GEMINI_API_KEY is present, otherwise returns MockEmbeddingService.
func NewDefaultEmbeddingService() EmbeddingService {
	mock := NewMockEmbeddingService(768)
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey != "" && !strings.EqualFold(apiKey, "mock") {
		geminiClient := NewGeminiEmbeddingClient(apiKey)
		return NewResilientEmbeddingService(geminiClient, mock)
	}
	return mock
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
