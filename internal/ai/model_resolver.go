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

	var cleanIDs []string
	for _, id := range availableIDs {
		cleanIDs = append(cleanIDs, strings.TrimPrefix(id, "models/"))
	}

	switch strings.ToLower(providerType) {
	case "groq":
		disallowed := []string{"whisper", "guard", "orpheus", "canopylabs"}
		isAllowed := func(id string) bool {
			lower := strings.ToLower(id)
			for _, dis := range disallowed {
				if strings.Contains(lower, dis) {
					return false
				}
			}
			return true
		}

		prefixes := []string{
			"llama-3.3", "llama-3.1", "llama3",
			"openai/gpt-oss-120b", "qwen/qwen3.8-27b", "openai/gpt-oss-20b",
			"qwen", "gpt-oss", "mixtral",
		}
		for _, prefix := range prefixes {
			for _, id := range cleanIDs {
				if isAllowed(id) && strings.Contains(strings.ToLower(id), prefix) {
					return id
				}
			}
		}
		for _, id := range cleanIDs {
			if isAllowed(id) {
				return id
			}
		}

	case "nvidia":
		disallowed := []string{"embed", "clip", "guard", "reward", "video", "synthetic", "deplot", "vision", "parse"}
		isAllowed := func(id string) bool {
			lower := strings.ToLower(id)
			for _, dis := range disallowed {
				if strings.Contains(lower, dis) {
					return false
				}
			}
			return true
		}

		prefixes := []string{
			"meta/llama-3.3", "meta/llama-3.1", "nvidia/llama-3.1-nemotron-70b-instruct",
			"mistralai/mistral-large-2-instruct", "mistralai/mistral-7b-instruct-v0.3",
			"google/gemma-3-12b-it", "deepseek-ai/deepseek-v4.1-flash", "meta/llama3",
		}
		for _, prefix := range prefixes {
			for _, id := range cleanIDs {
				if isAllowed(id) && strings.Contains(strings.ToLower(id), prefix) {
					return id
				}
			}
		}
		for _, id := range cleanIDs {
			if isAllowed(id) {
				return id
			}
		}

	case "gemini":
		disallowed := []string{"embedding", "image", "tts", "preview", "live", "audio", "robotics", "computer-use", "banana", "transcribe"}
		isAllowed := func(id string) bool {
			lower := strings.ToLower(id)
			for _, dis := range disallowed {
				if strings.Contains(lower, dis) {
					return false
				}
			}
			return true
		}

		prefixes := []string{"gemini-3.8-flash", "gemini-3.7-flash", "gemini-3.5-flash", "gemini-flash-latest", "gemini-2.0-flash", "gemini-1.5-flash", "gemini"}
		for _, prefix := range prefixes {
			for _, id := range cleanIDs {
				if isAllowed(id) && strings.Contains(strings.ToLower(id), prefix) {
					return id
				}
			}
		}
		for _, id := range cleanIDs {
			if isAllowed(id) {
				return id
			}
		}
	}

	return cleanIDs[0]
}

func (r *ModelResolver) getStaticFallbackModel(providerType string) string {
	switch strings.ToLower(providerType) {
	case "groq":
		return "llama-3.3-70b-versatile"
	case "nvidia":
		return "meta/llama-3.1-8b-instruct"
	case "gemini":
		return "gemini-3.8-flash"
	default:
		return "default"
	}
}
