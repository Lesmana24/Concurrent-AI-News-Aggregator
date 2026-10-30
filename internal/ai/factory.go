package ai

import (
	"os"
	"strconv"
	"time"
)

// NewDefaultAIService initializes the standard multi-provider chain based on environment variables:
// Priority: Groq -> NVIDIA NIM -> Google AI Studio -> Local-Fallback
func NewDefaultAIService() AIService {
	timeoutSeconds := 25
	if rawTimeout := os.Getenv("AI_TIMEOUT_SECONDS"); rawTimeout != "" {
		if parsed, err := strconv.Atoi(rawTimeout); err == nil && parsed > 0 {
			timeoutSeconds = parsed
		}
	}
	timeout := time.Duration(timeoutSeconds) * time.Second

	// 1. Groq Cloud
	groqKey := os.Getenv("GROQ_API_KEY")
	groqModel := os.Getenv("GROQ_MODEL")
	if groqModel == "" {
		groqModel = "auto"
	}
	groqProvider := NewGroqProvider(groqKey, groqModel, timeout)

	// 2. NVIDIA NIM
	nvidiaKey := os.Getenv("NVIDIA_NIM_API_KEY")
	if nvidiaKey == "" {
		nvidiaKey = os.Getenv("NVIDIA_API_KEY")
	}
	nvidiaModel := os.Getenv("NVIDIA_NIM_MODEL")
	if nvidiaModel == "" {
		nvidiaModel = os.Getenv("NVIDIA_MODEL")
	}
	if nvidiaModel == "" {
		nvidiaModel = "auto"
	}
	nvidiaProvider := NewNvidiaProvider(nvidiaKey, nvidiaModel, timeout)

	// 3. Google AI Studio (Gemini)
	geminiKey := os.Getenv("GEMINI_API_KEY")
	if geminiKey == "" {
		geminiKey = os.Getenv("GOOGLE_AI_API_KEY")
	}
	if geminiKey == "" {
		geminiKey = os.Getenv("GOOGLE_API_KEY")
	}
	geminiModel := os.Getenv("GEMINI_MODEL")
	if geminiModel == "" {
		geminiModel = "auto"
	}
	geminiProvider := NewGeminiProvider(geminiKey, geminiModel, timeout)

	// 4. Local Text Fallback
	localFallback := NewLocalFallbackProvider()

	return NewFallbackChain(
		groqProvider,
		nvidiaProvider,
		geminiProvider,
		localFallback,
	)
}

// NewDefaultSummarizer provides backward-compatible summarizer initialization.
func NewDefaultSummarizer() Summarizer {
	return NewDefaultAIService()
}
