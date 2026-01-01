package llmservice

import (
	"context"
	"exceltranslator/pkg/logger" // Import the logger package
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
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
		option.WithMaxRetries(3),
	)

	return &LLMService{
		config: config,
		client: &client,
		cache:  make(map[string]string), // Initialize the cache map
		logger: log,                     // Assign the logger
	}
}

func (s *LLMService) TruncateLog(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "...(truncated)"
}

// Translate translates the given text using the configured LLM with retries.
func (s *LLMService) Translate(ctx context.Context, text string) (string, error) {
	// 1. Check cache first
	s.mu.RLock()

	if translated, ok := s.cache[text]; ok {
		s.mu.RUnlock()
		s.logger.Tracef(
			"Cache hit for text: %s -> %s",
			s.TruncateLog(text, 80),
			s.TruncateLog(translated, 200),
		)
		return translated, nil // Cache hit
	}
	s.mu.RUnlock()
	s.logger.Tracef("Cache miss for text: %s", text)

	translatedResult, translateErr := s.doTranslateRequest(ctx, text)
	if translateErr == nil {
		// Store in cache after successful translation
		s.mu.Lock()
		s.cache[text] = translatedResult
		s.mu.Unlock()
		s.logger.Debugf("Translated text:\n%5s: %s\n%5s: %s",
			"Orig", s.TruncateLog(text, 80), "Trans", s.TruncateLog(translatedResult, 200))
		return translatedResult, nil
	}
	return "", translateErr
}

// doTranslateRequest performs the API request using the openai-go library.
func (s *LLMService) doTranslateRequest(ctx context.Context, text string) (string, error) {
	trimmed := strings.TrimSpace(text)

	s.logger.Tracef("Sending request to LLM for trimmed: %s", trimmed)

	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(s.config.Prompt + "\n\n" + trimmed),
		},
		Model:    s.config.Model,
		Metadata: map[string]string{"enable_thinking": "false"},
	}

	chatCompletion, err := s.client.Chat.Completions.New(ctx, params)
	if err == nil {
		if len(chatCompletion.Choices) == 0 {
			s.logger.Warnf("No translation choices found in LLM response.")
			return "", fmt.Errorf("no translation choices found in response")
		}
		result := chatCompletion.Choices[0].Message.Content
		s.logger.Tracef("Received translation result: %s", s.TruncateLog(result, 200))
		return result, nil
	}

	s.logger.Errorf("Failed to create chat completion: %v", err)
	return "", fmt.Errorf("failed to create chat completion: %w", err)
}
