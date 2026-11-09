package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCleanSummary_StripsThinkTagsAndXML(t *testing.T) {
	raw := `<think>
Model reasoning process:
1. Analisis fakta utama
2. Identifikasi 3 kalimat
</think>
<summary>
Pemerintah meresmikan jalan tol baru hari ini. Proyek ini memangkas waktu tempuh antar kota secara signifikan. Masyarakat menyambut baik pengoperasian infrastruktur tersebut.
</summary>`

	cleaned := cleanSummary(raw)
	assert.NotContains(t, cleaned, "<think>")
	assert.NotContains(t, cleaned, "</think>")
	assert.NotContains(t, cleaned, "Model reasoning process")
	assert.NotContains(t, cleaned, "<summary>")
	assert.NotContains(t, cleaned, "</summary>")
	assert.Contains(t, cleaned, "Pemerintah meresmikan jalan tol baru")
}

func TestCleanSummary_HandlesPrefixedSentences(t *testing.T) {
	raw := `Sentence 1: Pertumbuhan ekonomi nasional mencapai target tahun ini.
Sentence 2: Sektor manufaktur dan konsumsi domestik menjadi penopang utama.
Sentence 3: Bank sentral optimistis stabilitas moneter akan tetap terjaga.`

	cleaned := cleanSummary(raw)
	assert.NotContains(t, cleaned, "Sentence 1:")
	assert.NotContains(t, cleaned, "Sentence 2:")
	assert.NotContains(t, cleaned, "Sentence 3:")
	assert.Equal(t, "Pertumbuhan ekonomi nasional mencapai target tahun ini. Sektor manufaktur dan konsumsi domestik menjadi penopang utama. Bank sentral optimistis stabilitas moneter akan tetap terjaga.", cleaned)
}

func BenchmarkCleanSummary(b *testing.B) {
	raw := `<think>Internal thoughts</think><summary>Kalimat 1: Berita satu. Kalimat 2: Berita dua. Kalimat 3: Berita tiga.</summary>`
	b.ReportAllocs()
	for b.Loop() {
		_ = cleanSummary(raw)
	}
}

func TestOpenAICompatibleClient_429RetryWithLighterModel(t *testing.T) {
	requestCount := 0
	var receivedModels []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		receivedModels = append(receivedModels, req.Model)

		if requestCount == 1 {
			// First attempt returns 429 Too Many Requests with Retry-After header
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":{"message":"Rate limit exceeded for llama-3.3-70b-versatile"}}`))
			return
		}

		// Second attempt (retry) succeeds with lighter model
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [
				{
					"message": {
						"content": "Sentence 1: Berita berhasil dirangkum. Sentence 2: Informasi fakta lengkap. Sentence 3: Laporan selesai."
					}
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient(
		"Groq",
		"groq",
		server.URL,
		"",
		"test-api-key",
		"llama-3.3-70b-versatile",
		5*time.Second,
	)

	summary, err := client.Summarize(context.Background(), "Teks berita panjang mengenai kebijakan ekonomi nasional.")
	assert.NoError(t, err)
	assert.NotEmpty(t, summary)
	assert.Equal(t, 2, requestCount)
	assert.Equal(t, "llama-3.3-70b-versatile", receivedModels[0])
	assert.Equal(t, "llama-3.1-8b-instant", receivedModels[1], "Should have downgraded to lighter model upon 429")
}

func TestOpenAICompatibleClient_429StreamRetryWithLighterModel(t *testing.T) {
	requestCount := 0
	var receivedModels []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		receivedModels = append(receivedModels, req.Model)

		if requestCount == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":{"message":"Stream rate limit exceeded"}}`))
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Jawaban \"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"stream.\"}}]}\n\ndata: [DONE]\n\n"))
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient(
		"NVIDIA NIM",
		"nvidia",
		server.URL,
		"",
		"test-api-key",
		"meta/llama-3.3-70b-instruct",
		5*time.Second,
	)

	tokenChan := make(chan string, 10)
	err := client.executeChatStream(context.Background(), "System prompt", "User prompt", 100, tokenChan)
	close(tokenChan)

	assert.NoError(t, err)
	assert.Equal(t, 2, requestCount)
	assert.Equal(t, "meta/llama-3.3-70b-instruct", receivedModels[0])
	assert.Equal(t, "meta/llama-3.1-8b-instruct", receivedModels[1], "Should have downgraded to lighter NVIDIA model upon 429")

	var tokens []string
	for tok := range tokenChan {
		tokens = append(tokens, tok)
	}
	assert.Equal(t, []string{"Jawaban ", "stream."}, tokens)
}


