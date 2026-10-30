package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestModelResolver_ExplicitModel(t *testing.T) {
	resolver := NewModelResolver(time.Minute)
	res := resolver.ResolveModel(context.Background(), "http://example.com/models", "dummy-key", "groq", "llama-3.1-8b-instant")
	assert.Equal(t, "llama-3.1-8b-instant", res)
}

func TestModelResolver_AutoSelectGroqModel(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-groq-key", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": [
				{"id": "whisper-large-v3"},
				{"id": "llama-3.1-8b-instant"},
				{"id": "llama-3.3-70b-versatile"}
			]
		}`))
	}))
	defer mockServer.Close()

	resolver := NewModelResolver(time.Minute)
	resolved := resolver.ResolveModel(context.Background(), mockServer.URL, "test-groq-key", "groq", "auto")

	assert.Equal(t, "llama-3.3-70b-versatile", resolved)
}

func TestModelResolver_AutoSelectNvidiaModel(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": [
				{"id": "meta/llama-3.1-8b-instruct"},
				{"id": "meta/llama-3.3-70b-instruct"},
				{"id": "nvidia/nemotron-4-340b"}
			]
		}`))
	}))
	defer mockServer.Close()

	resolver := NewModelResolver(time.Minute)
	resolved := resolver.ResolveModel(context.Background(), mockServer.URL, "test-key", "nvidia", "auto")

	assert.Equal(t, "meta/llama-3.1-8b-instruct", resolved)
}

func TestModelResolver_AutoSelectGeminiModel(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": [
				{"id": "gemini-1.5-pro"},
				{"id": "gemini-2.0-flash-exp"},
				{"id": "text-embedding-004"}
			]
		}`))
	}))
	defer mockServer.Close()

	resolver := NewModelResolver(time.Minute)
	resolved := resolver.ResolveModel(context.Background(), mockServer.URL, "test-key", "gemini", "auto")

	assert.Equal(t, "gemini-2.0-flash-exp", resolved)
}

func TestModelResolver_StaticFallbackWhenServerFails(t *testing.T) {
	resolver := NewModelResolver(time.Minute)
	// Invalid port to trigger error
	resolved := resolver.ResolveModel(context.Background(), "http://127.0.0.1:59999/models", "key", "groq", "auto")
	assert.Equal(t, "qwen/qwen3.8-27b", resolved)
}

