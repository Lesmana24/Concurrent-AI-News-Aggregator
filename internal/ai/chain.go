package ai

import (
	"context"
	"fmt"
	"log"
	"news-aggregator/internal/models"
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

// AnswerQuestionStream iterates through configured providers in priority order and streams response tokens.
func (c *FallbackChain) AnswerQuestionStream(ctx context.Context, contextArticles, question string, outChan chan<- string) (string, error) {
	if len(c.providers) == 0 {
		return "", fmt.Errorf("no AI providers configured in fallback chain")
	}

	var errors []string

	for _, p := range c.providers {
		if !p.IsConfigured() {
			continue
		}

		err := p.AnswerQuestionStream(ctx, contextArticles, question, outChan)
		if err == nil {
			return p.Name(), nil
		}

		errMsg := fmt.Sprintf("[%s]: %v", p.Name(), err)
		errors = append(errors, errMsg)
		log.Printf("[AI Fallback Chain Stream] Provider %s failed (%v). Attempting next provider...", p.Name(), err)
	}

	return "", fmt.Errorf("all AI streaming providers failed: %s", strings.Join(errors, "; "))
}

// AnalyzeBias iterates through configured providers in priority order to evaluate media bias and framing.
func (c *FallbackChain) AnalyzeBias(ctx context.Context, articles []ArticleForAnalysis) (*models.NewsCompareResponse, string, error) {
	if len(c.providers) == 0 {
		return nil, "", fmt.Errorf("no AI providers configured in fallback chain")
	}

	var errors []string

	for _, p := range c.providers {
		if !p.IsConfigured() {
			continue
		}

		res, err := p.AnalyzeBias(ctx, articles)
		if err == nil && res != nil {
			res.Provider = p.Name()
			return res, p.Name(), nil
		}

		errMsg := fmt.Sprintf("[%s]: %v", p.Name(), err)
		errors = append(errors, errMsg)
		log.Printf("[AI Fallback Chain Bias] Provider %s failed (%v). Attempting next provider...", p.Name(), err)
	}

	return nil, "", fmt.Errorf("all AI bias comparison providers failed: %s", strings.Join(errors, "; "))
}
