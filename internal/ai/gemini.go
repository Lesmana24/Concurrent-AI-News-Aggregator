package ai

import (
	"context"
	"news-aggregator/internal/models"
	"time"
)

const (
	geminiChatURL   = "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions"
	geminiModelsURL = "https://generativelanguage.googleapis.com/v1beta/openai/models"
)

// GeminiProvider implements the Provider interface for Google AI Studio (Gemini).
type GeminiProvider struct {
	client *OpenAICompatibleClient
}

// NewGeminiProvider initializes a new GeminiProvider instance.
func NewGeminiProvider(apiKey, model string, timeout time.Duration) *GeminiProvider {
	if model == "" {
		model = "auto"
	}
	client := NewOpenAICompatibleClient(
		"Google AI Studio",
		"gemini",
		geminiChatURL,
		geminiModelsURL,
		apiKey,
		model,
		timeout,
	)
	return &GeminiProvider{client: client}
}

func (p *GeminiProvider) Name() string {
	return p.client.Name()
}

func (p *GeminiProvider) IsConfigured() bool {
	return p.client.IsConfigured()
}

func (p *GeminiProvider) Summarize(ctx context.Context, text string) (string, error) {
	return p.client.Summarize(ctx, text)
}

func (p *GeminiProvider) AnswerQuestion(ctx context.Context, contextArticles, question string) (string, error) {
	return p.client.AnswerQuestion(ctx, contextArticles, question)
}

func (p *GeminiProvider) AnswerQuestionStream(ctx context.Context, contextArticles, question string, outChan chan<- string) error {
	return p.client.AnswerQuestionStream(ctx, contextArticles, question, outChan)
}

func (p *GeminiProvider) AnalyzeBias(ctx context.Context, articles []ArticleForAnalysis) (*models.NewsCompareResponse, error) {
	return p.client.AnalyzeBias(ctx, articles)
}
