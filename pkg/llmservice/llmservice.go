package llmservice

import (
	"context"
	"exceltranslator/pkg/logger" // Import the logger package
	"fmt"
	"sync"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// LLMServiceConfig holds the configuration for the LLM service.
type LLMServiceConfig struct {
	BaseURL string
	APIKey  string
	Model   string
	Prompt  string // Base prompt for translation
}

// LLMService provides translation capabilities using an OpenAI-compatible API.
type LLMService struct {
	config LLMServiceConfig
	client *openai.Client
	cache  map[string]string // Cache for translated text
	mu     sync.RWMutex      // Mutex for cache access
	logger *logger.Logger    // Logger instance
}

// NewLLMService creates a new LLMService instance.
func NewLLMService(config LLMServiceConfig, log *logger.Logger) *LLMService {
	baseURL := config.BaseURL

	client := openai.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey(config.APIKey),
		option.WithRequestTimeout(60*time.Second),
	)

	return &LLMService{
		config: config,
		client: &client,
		cache:  make(map[string]string), // Initialize the cache map
		logger: log,                     // Assign the logger
	}
}

// Translate translates the given text using the configured LLM with retries.
func (s *LLMService) Translate(ctx context.Context, text string) (string, error) {
	// 1. Check cache first
	s.mu.RLock()
	if translated, ok := s.cache[text]; ok {
		s.mu.RUnlock()
		s.logger.Tracef(
			"Cache hit for text: %s -> %s",
			truncateForLog(text, 80),
			truncateForLog(translated, 200),
		)
		return translated, nil // Cache hit
	}
	s.mu.RUnlock()
	s.logger.Tracef("Cache miss for text: %s", text)

	maxRetries := 2 // From PLAN.md: "最多重试两次" (retry up to two times)
	var lastErr error
	var translatedResult string

	for i := 0; i <= maxRetries; i++ {
		translatedResult, lastErr = s.doTranslateRequest(ctx, text)
		if lastErr == nil {
			// Store in cache after successful translation
			s.mu.Lock()
			s.cache[text] = translatedResult
			s.mu.Unlock()
			s.logger.Debugf("Translated text:\n\t[src] %s\n\t[dst] %s",
				truncateForLog(text, 80),
				truncateForLog(translatedResult, 200))
			return translatedResult, nil
		}

		// Handle retry logic
		if i < maxRetries {
			s.logger.Warnf("LLM translation failed, retrying (attempt %d/%d): %v for text: %s", i+1, maxRetries+1, lastErr, text)
			time.Sleep(2 * time.Second) // Simple backoff, can be more sophisticated
		} else {
			// Last attempt failed, return the error
			s.logger.Errorf("LLM translation failed after %d retries: %v for text: %s", maxRetries, lastErr, text)
			return "", fmt.Errorf("LLM translation failed after %d retries: %w", maxRetries, lastErr)
		}
	}
	s.logger.Errorf("Unexpected error in retry logic for text: %s, last error: %v", text, lastErr)
	return "", fmt.Errorf("unexpected error in retry logic, last error: %w", lastErr) // Should not be reached in normal flow
}

// doTranslateRequest performs the actual API request using the openai-go library.
func (s *LLMService) doTranslateRequest(ctx context.Context, text string) (string, error) {
	// Construct the prompt with the text to translate
	s.logger.Tracef("Sending request to LLM for text: %s", text)

	chatCompletion, err := s.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.AssistantMessage(s.config.Prompt),
			openai.UserMessage(text),
		},
		Model:    s.config.Model,
		Metadata: map[string]string{"enable_thinking": "false"},
	})

	if err != nil {
		s.logger.Errorf("Failed to create chat completion: %v", err)
		return "", fmt.Errorf("failed to create chat completion: %w", err)
	}

	if len(chatCompletion.Choices) == 0 {
		s.logger.Warnf("No translation choices found in LLM response.")
		return "", fmt.Errorf("no translation choices found in response")
	}

	result := chatCompletion.Choices[0].Message.Content
	s.logger.Tracef("Received translation result: %s", truncateForLog(result, 200))
	return result, nil
}

func truncateForLog(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "...(truncated)"
}
