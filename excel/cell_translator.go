package excel

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/xuri/excelize/v2"
	"golang.org/x/sync/semaphore"
)

// TranslationTask 保存单个单元格翻译的信息
type TranslationTask struct {
	Sheet        string // 工作表名称
	CellCoord    string // 单元格坐标
	OriginalText string // 原始文本
}

// CellTranslator 处理 Excel 单元格中文本的翻译
type CellTranslator struct {
	maxConcurrentRequests int
	ctx                   context.Context
	translateFunc         func(string) (string, error)
}

// NewCellTranslator 创建一个新的 CellTranslator 实例
func NewCellTranslator(maxConcurrentRequests int, ctx context.Context, translateFunc func(string) (string, error)) *CellTranslator {
	return &CellTranslator{
		maxConcurrentRequests: maxConcurrentRequests,
		ctx:                   ctx,
		translateFunc:         translateFunc,
	}
}

// GetCellsForTranslation 识别需要翻译的单元格
func (ct *CellTranslator) GetCellsForTranslation(f *excelize.File) []TranslationTask {
	var tasks []TranslationTask
	sheets := f.GetSheetList()

	totalCellsChecked := 0
	for _, sheetName := range sheets {
		rows, err := f.GetRows(sheetName) // 获取所有包含数据的行
		if err != nil {
			fmt.Printf("警告: 无法获取工作表 '%s' 的行: %v", sheetName, err)
			continue
		}

		for r, row := range rows {
			for c, cellValue := range row {
				totalCellsChecked++
				cellCoord, err := excelize.CoordinatesToCellName(c+1, r+1) // excelize 是从 1 开始的
				if err != nil {
					fmt.Printf("警告: 无法转换坐标 (%d, %d): %v", c+1, r+1, err)
					continue
				}

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

// TranslateCells 翻译单元格内容并实时更新到 Excel 文件
func (ct *CellTranslator) TranslateCells(
	inputFile,
	outputFile string,
	onProgress func(original, translated string, err error, done, total int),
) error {
	// 检查上下文是否已取消
	select {
	case <-ct.ctx.Done():
		return ct.ctx.Err()
	default:
	}

	f, err := excelize.OpenFile(inputFile)
	if err != nil {
		return fmt.Errorf("打开输入 Excel 文件 '%s' 时出错: %w", inputFile, err)
	}

	tasks := ct.GetCellsForTranslation(f)
	total := len(tasks)

	if total == 0 {
		if err := f.SaveAs(outputFile); err != nil {
			return fmt.Errorf("保存输出文件 '%s' 时出错: %w", outputFile, err)
		}
		return nil
	}

	type TranslatedResult struct {
		Sheet      string // 工作表名称
		CellCoord  string // 单元格坐标
		Translated string // 翻译后的文本
	}

	results := make([]TranslatedResult, len(tasks))

	var wg sync.WaitGroup
	sem := semaphore.NewWeighted(int64(ct.maxConcurrentRequests))

	wg.Add(len(tasks))
	var doneCount int64

	// 创建子 context 用于更好的 goroutine 控制
	childCtx, childCancel := context.WithCancel(ct.ctx)

	// 错误通道，用于在 goroutine 中捕获第一个严重错误
	errCh := make(chan error, 1)

	go func() {
		defer func() {
			if err := f.Close(); err != nil {
				fmt.Printf("警告: 关闭输入文件时出错: %v", err)
			}
		}()
		defer childCancel()

		for i, task := range tasks {
			go func(i int, task TranslationTask) {
				defer wg.Done()

				// 首先检查上下文是否已取消
				select {
				case <-childCtx.Done():
					return
				default:
				}

				// 获取信号量以限制并发数，使用 select 处理取消
				acquireDone := make(chan error, 1)
				go func() {
					acquireDone <- sem.Acquire(childCtx, 1)
				}()

				select {
				case <-childCtx.Done():
					return
				case err := <-acquireDone:
					if err != nil {
						return
					}
				}
				defer sem.Release(1)

				// 再次检查上下文状态
				select {
				case <-childCtx.Done():
					return
				default:
				}

				translatedText, tranErr := ct.translateFunc(task.OriginalText)

				current := int(atomic.AddInt64(&doneCount, 1))

				if tranErr != nil {
					// 只在非取消错误时记录日志
					if !errors.Is(tranErr, context.Canceled) {
						fmt.Printf("翻译单元格 %s:%s 时出错: %v", task.Sheet, task.CellCoord, tranErr)
					}

					// 报告错误
					if onProgress != nil {
						onProgress(task.OriginalText, "", tranErr, current, total)
					}

					// 取消后续任务
					childCancel()

					// 尝试发送错误到错误通道（非阻塞）
					select {
					case errCh <- tranErr:
					default:
					}
					return
				}

				if translatedText != "" {
					results[i] = TranslatedResult{
						Sheet:      task.Sheet,
						CellCoord:  task.CellCoord,
						Translated: translatedText,
					}
				}

				if onProgress != nil {
					onProgress(task.OriginalText, translatedText, nil, current, total)
				}
			}(i, task)
		}

		wg.Wait()
		close(errCh)
	}()

	// 等待所有任务完成
	wg.Wait()

	// 检查是否有错误发生
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	default:
	}

	// 检查上下文是否已取消
	select {
	case <-ct.ctx.Done():
		return ct.ctx.Err()
	default:
	}

	// 替换内容
	for _, r := range results {
		// 跳过未成功翻译的结果（零值）
		if r.Sheet == "" || r.CellCoord == "" || r.Translated == "" {
			continue
		}

		cellErr := f.SetCellValue(r.Sheet, r.CellCoord, r.Translated)
		if cellErr != nil {
			return fmt.Errorf("更新单元格 %s:%s 时出错: %w", r.Sheet, r.CellCoord, cellErr)
		}
	}

	// 保存输出文件
	if err := f.SaveAs(outputFile); err != nil {
		return fmt.Errorf("保存输出文件 '%s' 时出错: %w", outputFile, err)
	}

	return nil
}
