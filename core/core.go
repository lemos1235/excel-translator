package core

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"exceltranslator/config"
	"exceltranslator/pkg/excel"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/xuri/excelize/v2"
	"golang.org/x/sync/semaphore"
)

// TranslationTask 保存单个单元格翻译的信息
type TranslationTask struct {
	Sheet        string // 工作表名称
	CellCoord    string // 单元格坐标
	OriginalText string // 原始文本
}

// TranslationResult 保存翻译结果
type TranslationResult struct {
	Sheet          string // 工作表名称
	CellCoord      string // 单元格坐标
	TranslatedText string // 翻译后的文本
	Error          error  // 错误信息
}

// ProcessExcelFile 处理单个 Excel 文件的翻译
func ProcessExcelFile(inputFile, outputFile string, onTranslated func(original, translated string)) error {
	log.Println("开始翻译")

	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("加载配置文件时出错: %v", err)
		return err
	}

	// 初始化 OpenAI 客户端（只初始化一次）
	openaiClient := openai.NewClient(
		option.WithBaseURL(cfg.LLM.APIURL),
		option.WithAPIKey(cfg.LLM.APIKey),
	)

	// 打开输入 Excel 文件
	f, err := excelize.OpenFile(inputFile)
	if err != nil {
		log.Printf("打开输入 Excel 文件 '%s' 时出错: %v", inputFile, err)
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("警告: 关闭输入文件时出错: %v", err)
		}
	}()

	// 识别需要翻译的单元格
	tasks := getCellsForTranslation(f)
	if len(tasks) == 0 {
		log.Println("未找到需要翻译的文本单元格")
		return nil
	}

	// 执行翻译
	results := translateCellsWithClient(tasks, &openaiClient, cfg, onTranslated)

	// 更新 Excel 文件中的翻译
	updateExcelWithTranslations(f, results)

	// 保存输出 Excel 文件
	if err := f.SaveAs(outputFile); err != nil {
		log.Fatalf("保存输出文件 '%s' 时出错: %v", outputFile, err)
	}

	// 处理形状中的文本翻译
	shapeTranslator := excel.NewShapeTranslator()
	if err := shapeTranslator.TranslateShapes(outputFile, outputFile, func(text string) (string, error) {
		translatedText, err := TranslateTextWithClient(&openaiClient, text, cfg)
		if err == nil {
			onTranslated(text, translatedText)
		}
		return translatedText, err
	}); err != nil {
		log.Printf("处理形状翻译时出错: %v", err)
	}

	log.Println("翻译任务已完成")
	return nil
}

// getCellsForTranslation 识别需要翻译的单元格
func getCellsForTranslation(f *excelize.File) []TranslationTask {
	tasks := []TranslationTask{}
	sheets := f.GetSheetList()

	totalCellsChecked := 0
	for _, sheetName := range sheets {
		rows, err := f.GetRows(sheetName) // 获取所有包含数据的行
		if err != nil {
			log.Printf("警告: 无法获取工作表 '%s' 的行: %v", sheetName, err)
			continue
		}

		for r, row := range rows {
			for c, cellValue := range row {
				totalCellsChecked++
				cellCoord, _ := excelize.CoordinatesToCellName(c+1, r+1) // excelize 是从 1 开始的

				// 仅翻译非空字符串单元格
				if strings.TrimSpace(cellValue) != "" {
					tasks = append(tasks, TranslationTask{
						Sheet:        sheetName,
						CellCoord:    cellCoord,
						OriginalText: cellValue,
					})
				}
			}
		}
	}
	return tasks
}

// translateCellsWithClient 翻译单元格内容（使用已初始化的客户端）
func translateCellsWithClient(tasks []TranslationTask, client *openai.Client, cfg *config.Config, onTranslated func(original, translated string)) []TranslationResult {
	var wg sync.WaitGroup
	taskChan := make(chan TranslationTask, len(tasks))
	resultChan := make(chan TranslationResult, len(tasks))
	sem := semaphore.NewWeighted(int64(cfg.Client.MaxConcurrentRequests))
	results := make([]TranslationResult, 0, len(tasks))

	ctx := context.Background()

	// 启动工作 goroutine
	for i := range cfg.Client.MaxConcurrentRequests {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for task := range taskChan {
				// 获取信号量
				if err := sem.Acquire(ctx, 1); err != nil {
					log.Printf("获取信号量失败: %v", err)
					continue
				}

				translatedText, err := func() (string, error) {
					// 使用 defer 释放信号量
					defer sem.Release(1)

					translatedText, err := TranslateTextWithClient(client, task.OriginalText, cfg)
					if err == nil {
						onTranslated(task.OriginalText, translatedText)
					}
					return translatedText, err
				}()

				resultChan <- TranslationResult{
					Sheet:          task.Sheet,
					CellCoord:      task.CellCoord,
					TranslatedText: translatedText,
					Error:          err,
				}
			}
		}(i)
	}

	// 将任务发送到通道
	for _, task := range tasks {
		taskChan <- task
	}
	close(taskChan)

	// 等待所有工作进程完成
	wg.Wait()
	close(resultChan)

	// 收集结果
	for result := range resultChan {
		results = append(results, result)
	}

	return results
}

// updateExcelWithTranslations 更新 Excel 文件中的翻译内容
func updateExcelWithTranslations(f *excelize.File, results []TranslationResult) {
	translatedCount := 0
	errorCount := 0
	updateErrors := 0

	for _, result := range results {
		if result.Error != nil {
			errorCount++
			log.Printf("翻译单元格 %s:%s 时出错: %v", result.Sheet, result.CellCoord, result.Error)
			// 跳过错误单元格，保留原文
		} else if result.TranslatedText != "" {
			err := f.SetCellValue(result.Sheet, result.CellCoord, result.TranslatedText)
			if err != nil {
				updateErrors++
				log.Printf("更新 Excel 对象中的单元格 %s:%s 时出错: %v", result.Sheet, result.CellCoord, err)
			} else {
				translatedCount++
			}
		}
		// 如果翻译文本为空（例如，输入只是空白），则不执行任何操作，保留原文
	}

	log.Printf("翻译摘要：%d 个单元格成功翻译，%d 个 API 错误，%d 个 Excel 更新错误。", translatedCount, errorCount, updateErrors)
	if errorCount > 0 {
		log.Println("警告：由于 API 错误，某些单元格无法翻译。已保留原文。")
	}
	if updateErrors > 0 {
		log.Println("警告：某些翻译后的单元格无法写回 Excel 结构。")
	}
}

// IsCJKText 检查文本是否包含中文、日文或韩文字符
func IsCJKText(text string) bool {
	for _, r := range text {
		// Unicode 范围内的 CJK 字符
		if (r >= '\u4e00' && r <= '\u9fff') || // 中文
			(r >= '\u3040' && r <= '\u309f') || // 日文平假名
			(r >= '\u30a0' && r <= '\u30ff') || // 日文片假名
			(r >= '\uac00' && r <= '\ud7af') { // 韩文
			return true
		}
	}
	return false
}

// TranslateTextWithClient 将文本发送到 OpenAI 进行翻译（使用已初始化的客户端）
func TranslateTextWithClient(client *openai.Client, textToTranslate string, cfg *config.Config) (string, error) {
	if strings.TrimSpace(textToTranslate) == "" {
		return "", nil // 没有需要翻译的内容
	}

	// 如果启用了自动检测CJK，且文本不包含CJK字符，直接返回原文
	if cfg.Client.AutoDetectCJK && !IsCJKText(textToTranslate) {
		return textToTranslate, nil
	}

	resp, err := client.Chat.Completions.New(context.Background(),
		openai.ChatCompletionNewParams{
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(cfg.Client.Prompt),
				openai.UserMessage(textToTranslate),
			},
			Model: cfg.LLM.Model,
		})

	if err != nil {
		return "", fmt.Errorf("OpenAI API 调用失败: %w", err)
	}

	if len(resp.Choices) > 0 && resp.Choices[0].Message.Content != "" {
		// 去除 API 响应中可能的前导/尾随空白或换行符
		return strings.TrimSpace(resp.Choices[0].Message.Content), nil
	}

	return "", fmt.Errorf("OpenAI 返回了空的翻译")
}
