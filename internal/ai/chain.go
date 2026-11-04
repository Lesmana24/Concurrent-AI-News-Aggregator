package ai

import (
	"context"
	"fmt"
	"log"
	"news-aggregator/internal/models"
	"strings"
	"sync/atomic"
)

// StreamControlReset is an internal sentinel string emitted to signal that a provider failed mid-stream
// and the downstream consumer must reset/discard previously accumulated tokens before the next provider streams.
const StreamControlReset = "\x00SSE_STREAM_RESET\x00"

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
// It isolates each provider's stream with an intermediary channel to prevent partial token contamination across fallbacks.
func (c *FallbackChain) AnswerQuestionStream(ctx context.Context, contextArticles, question string, outChan chan<- string) (string, error) {
	if len(c.providers) == 0 {
		return "", fmt.Errorf("no AI providers configured in fallback chain")
	}

	var errors []string

	for _, p := range c.providers {
		if !p.IsConfigured() {
			continue
		}

		providerChan := make(chan string, 20)
		var tokensFromThisProvider int64
		forwarderDone := make(chan struct{})

		// Intercept and forward tokens while counting emitted chunks
		go func() {
			defer close(forwarderDone)
			for token := range providerChan {
				atomic.AddInt64(&tokensFromThisProvider, 1)
				select {
				case <-ctx.Done():
					return
				case outChan <- token:
				}
			}
		}()

		err := p.AnswerQuestionStream(ctx, contextArticles, question, providerChan)
		close(providerChan)
		<-forwarderDone

		if err == nil {
			return p.Name(), nil
		}

		errMsg := fmt.Sprintf("[%s]: %v", p.Name(), err)
		errors = append(errors, errMsg)
		log.Printf("[AI Fallback Chain Stream] Provider %s failed (%v)", p.Name(), err)

		// Check if this provider leaked any tokens before failing
		if atomic.LoadInt64(&tokensFromThisProvider) > 0 {
			// Tokens were already emitted to outChan!
			// Emit a reset signal so downstream consumer discards corrupted partial tokens
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case outChan <- StreamControlReset:
				log.Printf("[AI Fallback Chain Stream] Emitted StreamControlReset after %s mid-stream failure (%d tokens emitted)", p.Name(), tokensFromThisProvider)
			}
		}
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
