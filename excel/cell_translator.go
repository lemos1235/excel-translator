package excel

import (
	"context"
	"fmt"
	"strings"
	"sync"

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
}

// NewCellTranslator 创建一个新的 CellTranslator 实例
func NewCellTranslator(maxConcurrentRequests int) *CellTranslator {
	return &CellTranslator{
		maxConcurrentRequests: maxConcurrentRequests,
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
func (ct *CellTranslator) TranslateCells(inputFile, outputFile string, translateFunc func(string) (string, error)) {
	f, err := excelize.OpenFile(inputFile)
	if err != nil {
		fmt.Printf("打开输入 Excel 文件 '%s' 时出错: %v", inputFile, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Printf("警告: 关闭输入文件时出错: %v", err)
		}
	}()

	tasks := ct.GetCellsForTranslation(f)
	if len(tasks) == 0 {
		return
	}

	type TranslatedResult struct {
		Sheet      string // 工作表名称
		CellCoord  string // 单元格坐标
		Translated string // 翻译后的文本
	}

	results := make([]TranslatedResult, len(tasks))

	var wg sync.WaitGroup
	sem := semaphore.NewWeighted(int64(ct.maxConcurrentRequests))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg.Add(len(tasks))

	for i, task := range tasks {
		go func(i int, task TranslationTask) {
			defer wg.Done()

			// 获取信号量以限制并发数
			if err := sem.Acquire(ctx, 1); err != nil {
				fmt.Printf("获取信号量失败: %v", err)
				return
			}
			defer sem.Release(1)

			translatedText, tranErr := translateFunc(task.OriginalText)

			if tranErr != nil {
				fmt.Printf("翻译单元格 %s:%s 时出错: %v", task.Sheet, task.CellCoord, tranErr)
				return
			} else if translatedText != "" {
				results[i] = TranslatedResult{
					Sheet:      task.Sheet,
					CellCoord:  task.CellCoord,
					Translated: translatedText,
				}
			}
		}(i, task)
	}

	wg.Wait()

	// 替换内容
	for _, r := range results {
		cellErr := f.SetCellValue(r.Sheet, r.CellCoord, r.Translated)
		if cellErr != nil {
			fmt.Printf("更新单元格 %s:%s 时出错: %v", r.Sheet, r.CellCoord, cellErr)
			return
		}
	}

	// 保存输出文件
	if err := f.SaveAs(outputFile); err != nil {
		fmt.Printf("保存输出文件 '%s' 时出错: %v", outputFile, err)
	}
}
