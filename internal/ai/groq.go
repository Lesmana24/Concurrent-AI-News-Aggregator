package ai

import (
	"context"
	"news-aggregator/internal/models"
	"time"
)

const (
	groqChatURL   = "https://api.groq.com/openai/v1/chat/completions"
	groqModelsURL = "https://api.groq.com/openai/v1/models"
)

// GroqProvider implements the Provider interface for Groq Cloud.
type GroqProvider struct {
	client *OpenAICompatibleClient
}

// NewGroqProvider initializes a new GroqProvider instance.
func NewGroqProvider(apiKey, model string, timeout time.Duration) *GroqProvider {
	if model == "" {
		model = "auto"
	}
	client := NewOpenAICompatibleClient(
		"Groq",
		"groq",
		groqChatURL,
		groqModelsURL,
		apiKey,
		model,
		timeout,
	)
	return &GroqProvider{client: client}
}

func (p *GroqProvider) Name() string {
	return p.client.Name()
}

func (p *GroqProvider) IsConfigured() bool {
	return p.client.IsConfigured()
}

func (p *GroqProvider) Summarize(ctx context.Context, text string) (string, error) {
	return p.client.Summarize(ctx, text)
}

func (p *GroqProvider) AnswerQuestion(ctx context.Context, contextArticles, question string) (string, error) {
	return p.client.AnswerQuestion(ctx, contextArticles, question)
}

func (p *GroqProvider) AnswerQuestionStream(ctx context.Context, contextArticles, question string, outChan chan<- string) error {
	return p.client.AnswerQuestionStream(ctx, contextArticles, question, outChan)
}

func (p *GroqProvider) AnalyzeBias(ctx context.Context, articles []ArticleForAnalysis) (*models.NewsCompareResponse, error) {
	return p.client.AnalyzeBias(ctx, articles)
}
