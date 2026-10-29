package ai

import "context"

// Summarizer provides text summarization capabilities across AI providers.
type Summarizer interface {
	Summarize(ctx context.Context, text string) (summary string, provider string, err error)
}

// Provider represents an individual AI service provider (e.g. Groq, NVIDIA NIM, Google AI, Local-Fallback).
type Provider interface {
	Name() string
	IsConfigured() bool
	Summarize(ctx context.Context, text string) (string, error)
}

