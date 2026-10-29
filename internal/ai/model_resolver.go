package ai

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type cachedModel struct {
	modelName string
	expiresAt time.Time
}

// ModelResolver dynamically discovers and caches active models for AI providers.
type ModelResolver struct {
	mu     sync.RWMutex
	cache  map[string]cachedModel
	ttl    time.Duration
	client *http.Client
}

// NewModelResolver creates a new ModelResolver instance.
func NewModelResolver(ttl time.Duration) *ModelResolver {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &ModelResolver{
		cache:  make(map[string]cachedModel),
		ttl:    ttl,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

var defaultResolver = NewModelResolver(24 * time.Hour)

// DefaultResolver returns the shared ModelResolver instance.
func DefaultResolver() *ModelResolver {
	return defaultResolver
}

type modelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// ResolveModel returns an active model name. If configuredModel is empty or "auto",
// it queries the provider's /models endpoint or selects from a fallback priority list.
func (r *ModelResolver) ResolveModel(ctx context.Context, modelsURL, apiKey, providerType, configuredModel string) string {
	cleanModel := strings.TrimSpace(configuredModel)
	if cleanModel != "" && !strings.EqualFold(cleanModel, "auto") {
		return cleanModel
	}

	cacheKey := providerType + ":" + modelsURL

	r.mu.RLock()
	if item, found := r.cache[cacheKey]; found && time.Now().Before(item.expiresAt) {
		r.mu.RUnlock()
		return item.modelName
	}
	r.mu.RUnlock()

	discovered := r.fetchActiveModel(ctx, modelsURL, apiKey, providerType)
	if discovered == "" {
		discovered = r.getStaticFallbackModel(providerType)
	}

	r.mu.Lock()
	r.cache[cacheKey] = cachedModel{
		modelName: discovered,
		expiresAt: time.Now().Add(r.ttl),
	}
	r.mu.Unlock()

	log.Printf("[%s] Auto-resolved active model: %s", providerType, discovered)
	return discovered
}

func (r *ModelResolver) fetchActiveModel(ctx context.Context, modelsURL, apiKey, providerType string) string {
	if modelsURL == "" || apiKey == "" {
		return ""
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		log.Printf("[%s] Warning: unable to query models endpoint (%v), using default candidate", providerType, err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		log.Printf("[%s] Warning: models endpoint returned status %d: %s", providerType, resp.StatusCode, string(b))
		return ""
	}

	var res modelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return ""
	}

	var ids []string
	for _, m := range res.Data {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}

	return r.selectBestModel(providerType, ids)
}

func (r *ModelResolver) selectBestModel(providerType string, availableIDs []string) string {
	if len(availableIDs) == 0 {
		return ""
	}

	switch strings.ToLower(providerType) {
	case "groq":
		// Priority: llama-3.3 -> llama-3.1 -> llama3 -> any chat/instruct model
		prefixes := []string{"llama-3.3", "llama-3.1", "llama3-70b", "llama3-8b", "mixtral"}
		for _, prefix := range prefixes {
			for _, id := range availableIDs {
				if strings.Contains(strings.ToLower(id), prefix) && !strings.Contains(strings.ToLower(id), "whisper") {
					return id
				}
			}
		}
	case "nvidia":
		// Priority: meta/llama-3.3 -> meta/llama-3.1 -> meta/llama3 -> mistral
		prefixes := []string{"meta/llama-3.3", "meta/llama-3.1", "meta/llama3", "mistralai/mistral-large"}
		for _, prefix := range prefixes {
			for _, id := range availableIDs {
				if strings.Contains(strings.ToLower(id), prefix) {
					return id
				}
			}
		}
	case "gemini":
		// Priority: gemini-2.0-flash -> gemini-1.5-flash -> gemini-1.5-pro -> any gemini
		prefixes := []string{"gemini-2.0-flash", "gemini-1.5-flash", "gemini-1.5-pro", "gemini"}
		for _, prefix := range prefixes {
			for _, id := range availableIDs {
				if strings.Contains(strings.ToLower(id), prefix) {
					return id
				}
			}
		}
	}

	// Fallback to first available model if no priority matched
	return availableIDs[0]
}

func (r *ModelResolver) getStaticFallbackModel(providerType string) string {
	switch strings.ToLower(providerType) {
	case "groq":
		return "llama-3.3-70b-versatile"
	case "nvidia":
		return "meta/llama-3.1-8b-instruct"
	case "gemini":
		return "gemini-1.5-flash"
	default:
		return "default"
	}
}
