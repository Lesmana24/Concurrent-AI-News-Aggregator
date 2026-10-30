package ai

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// FallbackChain executes a list of AI providers sequentially until one succeeds.
type FallbackChain struct {
	providers []Provider
}

// NewFallbackChain creates a new FallbackChain with the given providers.
func NewFallbackChain(providers ...Provider) *FallbackChain {
	return &FallbackChain{providers: providers}
}

// Summarize iterates through configured providers in priority order.
func (c *FallbackChain) Summarize(ctx context.Context, text string) (string, string, error) {
	if len(c.providers) == 0 {
		return "", "", fmt.Errorf("no AI providers configured in fallback chain")
	}

	var errors []string

	for _, p := range c.providers {
		if !p.IsConfigured() {
			continue
		}

		summary, err := p.Summarize(ctx, text)
		if err == nil && strings.TrimSpace(summary) != "" {
			return summary, p.Name(), nil
		}

		errMsg := fmt.Sprintf("[%s]: %v", p.Name(), err)
		errors = append(errors, errMsg)
		log.Printf("[AI Fallback Chain] Provider %s failed (%v). Attempting next provider...", p.Name(), err)
	}

	return "", "", fmt.Errorf("all AI providers failed: %s", strings.Join(errors, "; "))
}

// AnswerQuestion iterates through configured providers in priority order to answer questions.
func (c *FallbackChain) AnswerQuestion(ctx context.Context, contextArticles, question string) (string, string, error) {
	if len(c.providers) == 0 {
		return "", "", fmt.Errorf("no AI providers configured in fallback chain")
	}

	var errors []string

	for _, p := range c.providers {
		if !p.IsConfigured() {
			continue
		}

		answer, err := p.AnswerQuestion(ctx, contextArticles, question)
		if err == nil && strings.TrimSpace(answer) != "" {
			return answer, p.Name(), nil
		}

		errMsg := fmt.Sprintf("[%s]: %v", p.Name(), err)
		errors = append(errors, errMsg)
		log.Printf("[AI Fallback Chain QA] Provider %s failed (%v). Attempting next provider...", p.Name(), err)
	}

	return "", "", fmt.Errorf("all AI providers failed: %s", strings.Join(errors, "; "))
}
