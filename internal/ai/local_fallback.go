package ai

import (
	"context"
	"fmt"
	"strings"
	"time"
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

func (p *LocalFallbackProvider) AnswerQuestion(ctx context.Context, contextArticles, question string) (string, error) {
	msg := "[Pemberitahuan Sistem] Layanan tanya jawab AI sedang tidak dapat memproses jawaban karena seluruh penyedia AI (Groq, NVIDIA NIM, Google AI Studio) sedang mengalami kendala teknis atau kuota token telah habis. Silakan coba kembali beberapa saat lagi."
	return msg, nil
}

func (p *LocalFallbackProvider) AnswerQuestionStream(ctx context.Context, contextArticles, question string, outChan chan<- string) error {
	msg := "[Pemberitahuan Sistem] Layanan tanya jawab AI sedang tidak dapat memproses jawaban karena seluruh penyedia AI (Groq, NVIDIA NIM, Google AI Studio) sedang mengalami kendala teknis atau kuota token telah habis. Silakan coba kembali beberapa saat lagi."
	words := strings.Split(msg, " ")
	for i, w := range words {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case outChan <- w + " ":
			if i%4 == 0 {
				time.Sleep(15 * time.Millisecond)
			}
		}
	}
	return nil
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
