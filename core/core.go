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

// TranslationCallback 是翻译过程中每翻译一个单元格时调用的回调函数
type TranslationCallback func(original, translated string)

// ProcessFunc 定义了处理Excel文件的函数类型
type ProcessFunc func(ctx context.Context, inputFile, outputFile string, onTranslated TranslationCallback) error

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
	onTranslated TranslationCallback
}

// NewTranslator 创建一个新的翻译器实例
func NewTranslator(cfg *config.Config, onTranslated TranslationCallback) (*Translator, error) {
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
func ProcessExcelFile(ctx context.Context, inputFile, outputFile string, onTranslated TranslationCallback) error {
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

	// 使用 defer 确保即使发生 panic 也能记录时间
	defer func() {
		elapsedTime := time.Since(startTime)
		if r := recover(); r != nil {
			log.Printf("翻译任务因 panic 终止，耗时: %v. panic: %v", elapsedTime, r)
			panic(r) // 重新抛出 panic
		}
	}()

	err = translator.ProcessFile(ctx, inputFile, outputFile)

	elapsedTime := time.Since(startTime)
	if err != nil {
		log.Printf("翻译任务失败，耗时: %v. 错误: %v", elapsedTime, err)
	} else {
		log.Printf("翻译任务完成，耗时: %v", elapsedTime)
	}
	return err
}

// ProcessFile 处理单个 Excel 文件的翻译
func (t *Translator) ProcessFile(ctx context.Context, inputFile, outputFile string) error {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 创建翻译回调函数，添加上下文检查
	createTranslateFunc := func(text string) (string, error) {
		// 在每个翻译调用前检查上下文
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		translatedText, err := t.TranslateText(ctx, text)
		if err == nil && t.onTranslated != nil {
			t.onTranslated(text, translatedText)
		}
		return translatedText, err
	}

	// Sheet 名称翻译
	sheetTranslator := excel.NewSheetTranslator(t.cfg.Client.MaxConcurrentRequests)
	err := sheetTranslator.TranslateSheetNames(ctx, inputFile, outputFile, createTranslateFunc)
	if err != nil {
		// 工作表名称翻译失败
		return fmt.Errorf("工作表名称翻译失败: %w", err)
	}

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Cells 翻译
	cellTranslator := excel.NewCellTranslator(t.cfg.Client.MaxConcurrentRequests)
	err = cellTranslator.TranslateCells(ctx, outputFile, outputFile, createTranslateFunc)
	if err != nil {
		// 单元格翻译失败
		return fmt.Errorf("单元格翻译失败: %w", err)
	}

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Shapes 翻译
	shapeTranslator := excel.NewShapeTranslator(t.cfg.Client.MaxConcurrentRequests)
	err = shapeTranslator.TranslateShapes(ctx, outputFile, outputFile, createTranslateFunc)
	if err != nil {
		// 形状翻译失败
		return fmt.Errorf("形状翻译失败: %w", err)
	}

	return nil
}

// TranslateText 将文本发送到翻译 API
func (t *Translator) TranslateText(ctx context.Context, textToTranslate string) (string, error) {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	if strings.TrimSpace(textToTranslate) == "" {
		return "", nil // 没有内容需要翻译
	}

	// 如果启用了仅翻译CJK文本，且文本不包含 CJK 字符，则返回原文
	if t.cfg.Client.OnlyTranslateCJK && !IsCJKText(textToTranslate) {
		return textToTranslate, nil
	}

	// 检查缓存
	if cached, ok := t.cache.Get(textToTranslate); ok {
		return cached, nil
	}

	// 再次检查上下文是否已取消（在 API 调用前）
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	resp, err := t.openaiClient.Chat.Completions.New(ctx,
		openai.ChatCompletionNewParams{
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(t.cfg.Client.Prompt),
				openai.UserMessage(textToTranslate),
			},
			Model:    t.cfg.LLM.Model,
			Metadata: map[string]string{"enable_thinking": "false"},
		})

	if err != nil {
		// OpenAI API 调用失败
		return "", err
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
