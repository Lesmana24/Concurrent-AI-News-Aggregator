package ai

import (
	"context"
	"news-aggregator/internal/models"
)

// ArticleForAnalysis contains the source url, outlet name, title, and content for comparative bias analysis.
type ArticleForAnalysis struct {
	URL     string `json:"url"`
	Outlet  string `json:"outlet"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// Summarizer provides text summarization capabilities across AI providers.
type Summarizer interface {
	Summarize(ctx context.Context, text string) (summary string, provider string, err error)
}

// QAEngine provides conversational Q&A over provided news context.
type QAEngine interface {
	AnswerQuestion(ctx context.Context, contextArticles, question string) (answer string, provider string, err error)
}

// StreamingQAEngine provides real-time token streaming for Q&A responses.
type StreamingQAEngine interface {
	AnswerQuestionStream(ctx context.Context, contextArticles, question string, outChan chan<- string) (provider string, err error)
}

// BiasAnalyzer evaluates sentiment, framing, and consensus across multiple news articles.
type BiasAnalyzer interface {
	AnalyzeBias(ctx context.Context, articles []ArticleForAnalysis) (result *models.NewsCompareResponse, provider string, err error)
}

// AIService aggregates summarization, standard Q&A, streaming, and comparative bias analysis capabilities.
type AIService interface {
	Summarizer
	QAEngine
	StreamingQAEngine
	BiasAnalyzer
}

// Provider represents an individual AI service provider (e.g. Groq, NVIDIA NIM, Google AI, Local-Fallback).
type Provider interface {
	Name() string
	IsConfigured() bool
	Summarize(ctx context.Context, text string) (string, error)
	AnswerQuestion(ctx context.Context, contextArticles, question string) (string, error)
	AnswerQuestionStream(ctx context.Context, contextArticles, question string, outChan chan<- string) error
	AnalyzeBias(ctx context.Context, articles []ArticleForAnalysis) (*models.NewsCompareResponse, error)
}
