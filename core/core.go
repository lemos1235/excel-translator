package core

import (
	"context"
	"errors"
	"exceltranslator/word"
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

// ProcessFunc 定义了处理Excel/Docx文件的函数类型（基于事件流）
type ProcessFunc func(ctx context.Context, inputFile, outputFile string) (<-chan TranslateEvent, error)

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
	events       chan TranslateEvent
}

// LLMError 表示底层大模型调用失败的错误
type LLMError struct {
	Err error
}

func (e LLMError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "llm error"
}

func (e LLMError) Unwrap() error {
	return e.Err
}

func isLLMError(err error) bool {
	var llmErr LLMError
	return errors.As(err, &llmErr)
}

// EventKind 表示翻译事件类型
type EventKind string

const (
	EventStart      EventKind = "start"
	EventProgress   EventKind = "progress"
	EventTranslated EventKind = "translated"
	EventError      EventKind = "error"
	EventComplete   EventKind = "complete"
)

// TranslateEvent 封装翻译过程中的各种事件信息
type TranslateEvent struct {
	Kind          EventKind
	Original      string
	Translated    string
	File          string
	Stage         string
	Err           error
	ProgressDone  int
	ProgressTotal int
}

// NewTranslator 创建一个新的翻译器实例
func NewTranslator(cfg *config.Config, events chan TranslateEvent) (*Translator, error) {
	client := openai.NewClient(
		option.WithBaseURL(cfg.LLM.APIURL),
		option.WithAPIKey(cfg.LLM.APIKey),
	)
	return &Translator{
		cfg:          cfg,
		openaiClient: &client,
		cache:        NewMemoryCache(),
		events:       events,
	}, nil
}

// emit 触发事件
func (t *Translator) emit(ctx context.Context, event TranslateEvent) {
	if t.events == nil {
		return
	}

	// 防止在通道被关闭后发送引发 panic
	defer func() {
		if r := recover(); r != nil {
			log.Printf("emit skipped on closed channel: %v", r)
		}
	}()

	select {
	case <-ctx.Done():
		return
	case t.events <- event:
	}
}

// ProcessFile 是翻译文档文件的主入口点（事件流形式）
func ProcessFile(ctx context.Context, inputFile, outputFile string) (<-chan TranslateEvent, error) {
	log.Println("开始翻译")

	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("加载配置时出错: %v", err)
		return nil, err
	}

	events := make(chan TranslateEvent, 32)

	go func() {
		startTime := time.Now()
		defer func() {
			elapsedTime := time.Since(startTime)
			if r := recover(); r != nil {
				log.Printf("翻译任务因 panic 终止，耗时: %v. panic: %v", elapsedTime, r)
				events <- TranslateEvent{Kind: EventError, File: inputFile, Err: fmt.Errorf("panic: %v", r)}
				events <- TranslateEvent{Kind: EventComplete, File: inputFile, Err: fmt.Errorf("panic: %v", r)}
				close(events)
				panic(r) // 重新抛出 panic
			}
		}()
		defer close(events)

		translator, err := NewTranslator(cfg, events)
		if err != nil {
			log.Printf("创建翻译器时出错: %v", err)
			events <- TranslateEvent{Kind: EventError, File: inputFile, Stage: "init", Err: err}
			events <- TranslateEvent{Kind: EventComplete, File: inputFile, Err: err}
			return
		}

		translator.emit(ctx, TranslateEvent{
			Kind: EventStart,
			File: inputFile,
		})

		// 检查是 xlsx 还是 docx 文件
		if strings.HasSuffix(inputFile, ".xlsx") {
			err = translator.TranslateExcelFile(ctx, inputFile, outputFile)
		} else if strings.HasSuffix(inputFile, ".docx") {
			err = translator.TranslateDocxFile(ctx, inputFile, outputFile)
		} else {
			err = fmt.Errorf("不支持的文件格式: %s", inputFile)
		}

		elapsedTime := time.Since(startTime)
		if err != nil {
			log.Printf("翻译任务失败，耗时: %v. 错误: %v", elapsedTime, err)
		} else {
			log.Printf("翻译任务完成，耗时: %v", elapsedTime)
		}

		translator.emit(ctx, TranslateEvent{
			Kind: EventComplete,
			File: inputFile,
			Err:  err,
		})
	}()

	return events, nil
}

// TranslateExcelFile 处理 Excel 文件的翻译
func (t *Translator) TranslateExcelFile(ctx context.Context, inputFile, outputFile string) error {
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
		return translatedText, err
	}

	// Sheet 名称翻译
	sheetTranslator := excel.NewSheetTranslator(t.cfg.Client.MaxConcurrentRequests, ctx, createTranslateFunc)
	err := sheetTranslator.TranslateSheetNames(inputFile, outputFile)
	if err != nil {
		// 工作表名称翻译失败
		t.emit(ctx, TranslateEvent{Kind: EventError, File: inputFile, Stage: "sheet", Err: err})
		return fmt.Errorf("工作表名称翻译失败: %w", err)
	}

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Cells 翻译
	cellTranslator := excel.NewCellTranslator(t.cfg.Client.MaxConcurrentRequests, ctx, createTranslateFunc)
	cellEvents, err := cellTranslator.TranslateCells(outputFile, outputFile)
	if err != nil {
		t.emit(ctx, TranslateEvent{Kind: EventError, File: inputFile, Stage: "cell", Err: err})
		return fmt.Errorf("单元格翻译失败: %w", err)
	}

	var cellErr error
	for ev := range cellEvents {
		if ev.Err != nil {
			cellErr = ev.Err
			if !isLLMError(ev.Err) {
				t.emit(ctx, TranslateEvent{Kind: EventError, File: inputFile, Stage: "cell", Err: ev.Err, ProgressDone: ev.Done, ProgressTotal: ev.Total})
			}
			break
		}
		t.emit(ctx, TranslateEvent{
			Kind:          EventTranslated,
			File:          inputFile,
			Stage:         "cell",
			Original:      ev.Task.OriginalText,
			Translated:    ev.Translated,
			ProgressDone:  ev.Done,
			ProgressTotal: ev.Total,
		})
		t.emit(ctx, TranslateEvent{
			Kind:          EventProgress,
			File:          inputFile,
			Stage:         "cell",
			ProgressDone:  ev.Done,
			ProgressTotal: ev.Total,
		})
	}
	if cellErr != nil {
		return fmt.Errorf("单元格翻译失败: %w", cellErr)
	}

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Shapes 翻译
	shapeTranslator := excel.NewShapeTranslator(t.cfg.Client.MaxConcurrentRequests, ctx, createTranslateFunc)
	err = shapeTranslator.TranslateShapes(outputFile, outputFile)
	if err != nil {
		// 形状翻译失败
		if !isLLMError(err) {
			t.emit(ctx, TranslateEvent{Kind: EventError, File: inputFile, Stage: "shape", Err: err})
		}
		return fmt.Errorf("形状翻译失败: %w", err)
	}

	return nil
}

// TranslateDocxFile 处理 Docx 文件的翻译
func (t *Translator) TranslateDocxFile(ctx context.Context, inputFile, outputFile string) error {
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
		return translatedText, err
	}

	// 文档内容翻译
	documentTranslator := word.NewDocumentTranslator(t.cfg.Client.MaxConcurrentRequests, ctx, createTranslateFunc)
	docEvents, err := documentTranslator.TranslateDocument(inputFile, outputFile)
	if err != nil {
		t.emit(ctx, TranslateEvent{Kind: EventError, File: inputFile, Stage: "docx", Err: err})
		return fmt.Errorf("文档内容翻译失败: %w", err)
	}

	var docErr error
	for ev := range docEvents {
		if ev.Err != nil {
			docErr = ev.Err
			if !isLLMError(ev.Err) {
				t.emit(ctx, TranslateEvent{Kind: EventError, File: inputFile, Stage: "docx", Err: ev.Err, ProgressDone: ev.Done, ProgressTotal: ev.Total})
			}
			break
		}
		t.emit(ctx, TranslateEvent{
			Kind:          EventTranslated,
			File:          inputFile,
			Stage:         "docx",
			Original:      ev.Text,
			Translated:    ev.Translated,
			ProgressDone:  ev.Done,
			ProgressTotal: ev.Total,
		})
		t.emit(ctx, TranslateEvent{
			Kind:          EventProgress,
			File:          inputFile,
			Stage:         "docx",
			ProgressDone:  ev.Done,
			ProgressTotal: ev.Total,
		})
	}

	if docErr != nil {
		return fmt.Errorf("文档内容翻译失败: %w", docErr)
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

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		resp, err := t.openaiClient.Chat.Completions.New(ctx,
			openai.ChatCompletionNewParams{
				Messages: []openai.ChatCompletionMessageParamUnion{
					openai.AssistantMessage(t.cfg.Client.Prompt),
					openai.UserMessage(textToTranslate),
				},
				Model:    t.cfg.LLM.Model,
				Metadata: map[string]string{"enable_thinking": "false"},
			})

		if err == nil && len(resp.Choices) > 0 && resp.Choices[0].Message.Content != "" {
			translated := strings.TrimSpace(resp.Choices[0].Message.Content)
			t.cache.Set(textToTranslate, translated)
			return translated, nil
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("OpenAI 返回了空的翻译结果")
		}

		// 失败重试，最多两次
		if attempt < 3 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(200 * time.Millisecond):
			}
		}
	}

	// 多次失败，提交错误事件并返回
	llmErr := LLMError{Err: lastErr}
	t.emit(ctx, TranslateEvent{Kind: EventError, Stage: "llm", Err: llmErr})
	return "", llmErr
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
