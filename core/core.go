package core

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"exceltranslator/config"
	"exceltranslator/excel"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// CacheManager 定义缓存接口
type CacheManager interface {
	Get(key string) (string, bool)
	Set(key, value string)
}

// MemoryCache 是 CacheManager 的内存实现
type MemoryCache struct {
	cache *sync.Map
}

// NewMemoryCache 创建一个新的内存缓存
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		cache: &sync.Map{},
	}
}

// Get 从缓存中获取值
func (m *MemoryCache) Get(key string) (string, bool) {
	if v, ok := m.cache.Load(key); ok {
		if cached, ok2 := v.(string); ok2 {
			return cached, true
		}
	}
	return "", false
}

// Set 向缓存中添加值
func (m *MemoryCache) Set(key, value string) {
	m.cache.Store(key, value)
}

// Translator 封装翻译逻辑和状态
type Translator struct {
	cfg          *config.Config
	openaiClient *openai.Client
	cache        CacheManager
	onTranslated func(original, translated string)
}

// NewTranslator 创建一个新的翻译器实例
func NewTranslator(cfg *config.Config, onTranslated func(original, translated string)) (*Translator, error) {
	client := openai.NewClient(
		option.WithBaseURL(cfg.LLM.APIURL),
		option.WithAPIKey(cfg.LLM.APIKey),
	)
	return &Translator{
		cfg:          cfg,
		openaiClient: &client,
		cache:        NewMemoryCache(),
		onTranslated: onTranslated,
	}, nil
}

// ProcessExcelFile 是翻译 Excel 文件的主入口点
func ProcessExcelFile(inputFile, outputFile string, onTranslated func(original, translated string)) error {
	log.Println("开始翻译")
	startTime := time.Now()

	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("加载配置时出错: %v", err)
		return err
	}

	// 创建新的翻译器
	translator, err := NewTranslator(cfg, onTranslated)
	if err != nil {
		log.Printf("创建翻译器时出错: %v", err)
		return err
	}

	err = translator.ProcessFile(inputFile, outputFile)

	elapsedTime := time.Since(startTime)
	if err != nil {
		log.Printf("翻译任务失败，耗时: %v. 错误: %v", elapsedTime, err)
	} else {
		log.Printf("翻译任务完成，耗时: %v", elapsedTime)
	}
	return err
}

// ProcessFile 处理单个 Excel 文件的翻译
func (t *Translator) ProcessFile(inputFile, outputFile string) error {
	// Cells 翻译
	cellTranslator := excel.NewCellTranslator(t.cfg.Client.MaxConcurrentRequests)
	cellTranslator.TranslateCells(inputFile, outputFile, func(text string) (string, error) {
		translatedText, err := t.TranslateText(text)
		if err == nil && t.onTranslated != nil {
			t.onTranslated(text, translatedText)
		}
		return translatedText, err
	})

	// Shapes 翻译
	shapeTranslator := excel.NewShapeTranslator(t.cfg.Client.MaxConcurrentRequests)
	shapeTranslator.TranslateShapes(outputFile, outputFile, func(text string) (string, error) {
		translatedText, err := t.TranslateText(text)
		if err == nil && t.onTranslated != nil {
			t.onTranslated(text, translatedText)
		}
		return translatedText, err
	})

	return nil
}

// TranslateText 将文本发送到翻译 API
func (t *Translator) TranslateText(textToTranslate string) (string, error) {
	if strings.TrimSpace(textToTranslate) == "" {
		return "", nil // 没有内容需要翻译
	}

	// 如果启用了自动检测 CJK，且文本不包含 CJK 字符，则返回原文
	if t.cfg.Client.AutoDetectCJK && !IsCJKText(textToTranslate) {
		return textToTranslate, nil
	}

	// 检查缓存
	if cached, ok := t.cache.Get(textToTranslate); ok {
		return cached, nil
	}

	resp, err := t.openaiClient.Chat.Completions.New(context.Background(),
		openai.ChatCompletionNewParams{
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(t.cfg.Client.Prompt),
				openai.UserMessage(textToTranslate),
			},
			Model: t.cfg.LLM.Model,
		})

	if err != nil {
		return "", fmt.Errorf("OpenAI API 调用失败: %w", err)
	}

	if len(resp.Choices) > 0 && resp.Choices[0].Message.Content != "" {
		translated := strings.TrimSpace(resp.Choices[0].Message.Content)
		t.cache.Set(textToTranslate, translated)
		return translated, nil
	}

	return "", fmt.Errorf("OpenAI 返回了空的翻译结果")
}

// IsCJKText 检查文本是否包含中文、日文或韩文字符
func IsCJKText(text string) bool {
	for _, r := range text {
		// CJK 字符的 Unicode 范围
		if (r >= '\u4e00' && r <= '\u9fff') || // 中文
			(r >= '\u3040' && r <= '\u309f') || // 日文平假名
			(r >= '\u30a0' && r <= '\u30ff') || // 日文片假名
			(r >= '\uac00' && r <= '\ud7af') { // 韩文
			return true
		}
	}
	return false
}
