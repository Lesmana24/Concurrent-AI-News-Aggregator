package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	systemPrompt = `Anda adalah seorang editor berita profesional. Tugas Anda adalah merangkum teks artikel berita yang diberikan.
ATURAN MUTLAK:
1. Rangkuman HARUS ditulis dalam Bahasa Indonesia yang baku dan formal, terlepas dari bahasa asli artikel tersebut.
2. Rangkuman HARUS terdiri dari TEPAT 3 kalimat, tidak boleh lebih dan tidak boleh kurang.
3. Ambil intisari paling penting (Siapa, Apa, Kapan, Di mana, Mengapa).
4. JANGAN tambahkan kata pengantar (seperti "Berikut adalah rangkumannya...") atau kalimat penutup. Langsung berikan teks rangkumannya saja!`

	defaultTemperature = 0.3
	defaultMaxTokens   = 250
	defaultTimeout     = 25 * time.Second
)

// OpenAICompatibleClient is a reusable HTTP client for OpenAI-compatible LLM endpoints.
type OpenAICompatibleClient struct {
	providerName string
	providerType string
	chatURL      string
	modelsURL    string
	apiKey       string
	model        string
	httpClient   *http.Client
	resolver     *ModelResolver
}

// NewOpenAICompatibleClient creates a new OpenAICompatibleClient.
func NewOpenAICompatibleClient(
	providerName, providerType, chatURL, modelsURL, apiKey, model string,
	timeout time.Duration,
) *OpenAICompatibleClient {
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	return &OpenAICompatibleClient{
		providerName: providerName,
		providerType: providerType,
		chatURL:      chatURL,
		modelsURL:    modelsURL,
		apiKey:       strings.TrimSpace(apiKey),
		model:        strings.TrimSpace(model),
		httpClient:   &http.Client{Timeout: timeout},
		resolver:     DefaultResolver(),
	}
}

func (c *OpenAICompatibleClient) Name() string {
	return c.providerName
}

func (c *OpenAICompatibleClient) IsConfigured() bool {
	return c.apiKey != "" && c.chatURL != ""
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    interface{} `json:"code"`
	} `json:"error"`
}

func (c *OpenAICompatibleClient) Summarize(ctx context.Context, text string) (string, error) {
	if !c.IsConfigured() {
		return "", fmt.Errorf("%s is not configured (missing API key or URL)", c.providerName)
	}

	activeModel := c.resolver.ResolveModel(ctx, c.modelsURL, c.apiKey, c.providerType, c.model)

	reqPayload := chatRequest{
		Model: activeModel,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: fmt.Sprintf("Rangkum artikel berikut:\n\n%s", text)},
		},
		Temperature: defaultTemperature,
		MaxTokens:   defaultMaxTokens,
	}

	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.chatURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s request failed: %w", c.providerName, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read %s response: %w", c.providerName, err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s API responded with status %d: %s", c.providerName, resp.StatusCode, string(respBytes))
	}

	var parsedResp chatResponse
	if err := json.Unmarshal(respBytes, &parsedResp); err != nil {
		return "", fmt.Errorf("failed to parse %s JSON response: %w", c.providerName, err)
	}

	if parsedResp.Error != nil && parsedResp.Error.Message != "" {
		return "", fmt.Errorf("%s error: %s", c.providerName, parsedResp.Error.Message)
	}

	if len(parsedResp.Choices) == 0 || parsedResp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("%s returned an empty response", c.providerName)
	}

	summary := strings.TrimSpace(parsedResp.Choices[0].Message.Content)
	return summary, nil
}

