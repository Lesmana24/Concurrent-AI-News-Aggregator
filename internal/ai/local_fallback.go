package ai

import (
	"context"
	"fmt"
	"news-aggregator/internal/models"
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

func (p *LocalFallbackProvider) AnalyzeBias(ctx context.Context, articles []ArticleForAnalysis) (*models.NewsCompareResponse, error) {
	comparisons := make([]models.MediaFraming, len(articles))
	for i, a := range articles {
		comparisons[i] = models.MediaFraming{
			SourceURL:        a.URL,
			OutletName:       a.Outlet,
			Sentiment:        "Netral",
			SentimentScore:   0.0,
			DominantFraming:  "Pelaporan faktual standar",
			HighlightedFacts: []string{extractSnippet(a.Content, 100)},
		}
	}
	return &models.NewsCompareResponse{
		Topic:            "Analisis Komparatif Berita (Fallback Lokal)",
		ConsensusFacts:   []string{"Semua media melaporkan peristiwa yang sama."},
		MediaComparisons: comparisons,
		Divergences:      []string{"Analisis komparatif mendalam tidak dapat dilakukan saat seluruh cloud LLM sedang offline."},
		ObjectiveSummary: "Layanan cloud AI sedang mengalami kendala teknis. Di atas adalah data awal dari masing-masing portal media.",
		Provider:         p.Name(),
	}, nil
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
