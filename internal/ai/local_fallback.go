package ai

import (
	"context"
	"fmt"
	"strings"
)

// LocalFallbackProvider provides a text-based graceful fallback when external LLM APIs fail.
type LocalFallbackProvider struct{}

// NewLocalFallbackProvider creates a new LocalFallbackProvider.
func NewLocalFallbackProvider() *LocalFallbackProvider {
	return &LocalFallbackProvider{}
}

func (p *LocalFallbackProvider) Name() string {
	return "Local-Fallback"
}

func (p *LocalFallbackProvider) IsConfigured() bool {
	return true
}

func (p *LocalFallbackProvider) Summarize(ctx context.Context, text string) (string, error) {
	snippet := extractSnippet(text, 220)

	msg := fmt.Sprintf(
		"[Pemberitahuan Sistem] Ringkasan otomatis AI sedang tidak dapat dibuat karena seluruh penyedia AI (Groq, NVIDIA NIM, Google AI Studio) sedang mengalami kendala teknis atau kuota token telah habis. Silakan coba kembali beberapa saat lagi. Cuplikan awal berita: \"%s\"",
		snippet,
	)

	return msg, nil
}

func extractSnippet(text string, maxLen int) string {
	clean := strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if len(clean) <= maxLen {
		return clean
	}

	truncated := clean[:maxLen]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 50 {
		truncated = truncated[:lastSpace]
	}
	return truncated + "..."
}

