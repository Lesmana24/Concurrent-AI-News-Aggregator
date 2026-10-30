package ai

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockProvider struct {
	name       string
	configured bool
	summary    string
	answer     string
	err        error
	called     bool
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) IsConfigured() bool {
	return m.configured
}

func (m *mockProvider) Summarize(ctx context.Context, text string) (string, error) {
	m.called = true
	return m.summary, m.err
}

func (m *mockProvider) AnswerQuestion(ctx context.Context, contextArticles, question string) (string, error) {
	m.called = true
	return m.answer, m.err
}

func TestFallbackChain_FirstProviderSuccess(t *testing.T) {
	p1 := &mockProvider{name: "Groq", configured: true, summary: "Groq Summary", err: nil}
	p2 := &mockProvider{name: "NVIDIA NIM", configured: true, summary: "NVIDIA Summary", err: nil}
	local := NewLocalFallbackProvider()

	chain := NewFallbackChain(p1, p2, local)
	summary, provider, err := chain.Summarize(context.Background(), "Sample text content")

	assert.NoError(t, err)
	assert.Equal(t, "Groq Summary", summary)
	assert.Equal(t, "Groq", provider)
	assert.True(t, p1.called)
	assert.False(t, p2.called)
}

func TestFallbackChain_FailoverToSecondProvider(t *testing.T) {
	p1 := &mockProvider{name: "Groq", configured: true, err: errors.New("rate limit 429")}
	p2 := &mockProvider{name: "NVIDIA NIM", configured: true, summary: "NVIDIA Summary", err: nil}
	local := NewLocalFallbackProvider()

	chain := NewFallbackChain(p1, p2, local)
	summary, provider, err := chain.Summarize(context.Background(), "Sample text content")

	assert.NoError(t, err)
	assert.Equal(t, "NVIDIA Summary", summary)
	assert.Equal(t, "NVIDIA NIM", provider)
	assert.True(t, p1.called)
	assert.True(t, p2.called)
}

func TestFallbackChain_SkipUnconfiguredProvider(t *testing.T) {
	p1 := &mockProvider{name: "Groq", configured: false, summary: "Groq Summary", err: nil}
	p2 := &mockProvider{name: "NVIDIA NIM", configured: true, summary: "NVIDIA Summary", err: nil}
	local := NewLocalFallbackProvider()

	chain := NewFallbackChain(p1, p2, local)
	summary, provider, err := chain.Summarize(context.Background(), "Sample text content")

	assert.NoError(t, err)
	assert.Equal(t, "NVIDIA Summary", summary)
	assert.Equal(t, "NVIDIA NIM", provider)
	assert.False(t, p1.called)
	assert.True(t, p2.called)
}

func TestFallbackChain_AllCloudProvidersFail_LocalFallbackGracefulNotice(t *testing.T) {
	p1 := &mockProvider{name: "Groq", configured: true, err: errors.New("429 Too Many Requests")}
	p2 := &mockProvider{name: "NVIDIA NIM", configured: true, err: errors.New("500 Internal Server Error")}
	p3 := &mockProvider{name: "Google AI Studio", configured: true, err: errors.New("403 Quota Exceeded")}
	local := NewLocalFallbackProvider()

	chain := NewFallbackChain(p1, p2, p3, local)
	summary, provider, err := chain.Summarize(context.Background(), "Presiden Republik Indonesia meresmikan proyek infrastruktur baru di ibu kota nusantara hari ini.")

	assert.NoError(t, err)
	assert.Equal(t, "Local-Fallback", provider)
	assert.Contains(t, summary, "[Pemberitahuan Sistem]")
	assert.Contains(t, summary, "Presiden Republik Indonesia")
	assert.True(t, p1.called)
	assert.True(t, p2.called)
	assert.True(t, p3.called)
}

func TestFallbackChain_QA_SuccessAndFailover(t *testing.T) {
	p1 := &mockProvider{name: "Groq", configured: true, err: errors.New("429 Too Many Requests")}
	p2 := &mockProvider{name: "NVIDIA NIM", configured: true, answer: "Demo akan berlangsung di 61 titik.", err: nil}
	local := NewLocalFallbackProvider()

	chain := NewFallbackChain(p1, p2, local)
	answer, provider, err := chain.AnswerQuestion(context.Background(), "Artikel demo jakarta...", "Ada berapa titik demo?")

	assert.NoError(t, err)
	assert.Equal(t, "NVIDIA NIM", provider)
	assert.Equal(t, "Demo akan berlangsung di 61 titik.", answer)
	assert.True(t, p1.called)
	assert.True(t, p2.called)
}

func TestFallbackChain_QA_AllFail_LocalFallbackNotice(t *testing.T) {
	p1 := &mockProvider{name: "Groq", configured: true, err: errors.New("network timeout")}
	local := NewLocalFallbackProvider()

	chain := NewFallbackChain(p1, local)
	answer, provider, err := chain.AnswerQuestion(context.Background(), "Artikel demo jakarta...", "Ada berapa titik demo?")

	assert.NoError(t, err)
	assert.Equal(t, "Local-Fallback", provider)
	assert.Contains(t, answer, "[Pemberitahuan Sistem]")
	assert.Contains(t, answer, "tanya jawab AI sedang tidak dapat memproses jawaban")
}
