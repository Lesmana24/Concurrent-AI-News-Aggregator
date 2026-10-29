package ai

import (
	"context"
	"time"
)

const (
	nvidiaChatURL   = "https://integrate.api.nvidia.com/v1/chat/completions"
	nvidiaModelsURL = "https://integrate.api.nvidia.com/v1/models"
)

// NvidiaProvider implements the Provider interface for NVIDIA NIM.
type NvidiaProvider struct {
	client *OpenAICompatibleClient
}

// NewNvidiaProvider initializes a new NvidiaProvider instance.
func NewNvidiaProvider(apiKey, model string, timeout time.Duration) *NvidiaProvider {
	if model == "" {
		model = "auto"
	}
	client := NewOpenAICompatibleClient(
		"NVIDIA NIM",
		"nvidia",
		nvidiaChatURL,
		nvidiaModelsURL,
		apiKey,
		model,
		timeout,
	)
	return &NvidiaProvider{client: client}
}

func (p *NvidiaProvider) Name() string {
	return p.client.Name()
}

func (p *NvidiaProvider) IsConfigured() bool {
	return p.client.IsConfigured()
}

func (p *NvidiaProvider) Summarize(ctx context.Context, text string) (string, error) {
	return p.client.Summarize(ctx, text)
}

