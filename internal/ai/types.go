package ai

import "context"

// Summarizer provides text summarization capabilities across AI providers.
type Summarizer interface {
	Summarize(ctx context.Context, text string) (summary string, provider string, err error)
}

// QAEngine provides conversational Q&A over provided news context.
type QAEngine interface {
	AnswerQuestion(ctx context.Context, contextArticles, question string) (answer string, provider string, err error)
}

// AIService aggregates both summarization and Q&A capabilities.
type AIService interface {
	Summarizer
	QAEngine
}

// Provider represents an individual AI service provider (e.g. Groq, NVIDIA NIM, Google AI, Local-Fallback).
type Provider interface {
	Name() string
	IsConfigured() bool
	Summarize(ctx context.Context, text string) (string, error)
	AnswerQuestion(ctx context.Context, contextArticles, question string) (string, error)
}
