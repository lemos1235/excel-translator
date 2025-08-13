package excel

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/xuri/excelize/v2"
	"golang.org/x/sync/semaphore"
)

// SheetTranslationTask 保存工作表翻译任务信息
type SheetTranslationTask struct {
	OriginalName string // 原始工作表名称
	NewName      string // 翻译后的工作表名称
}

// SheetTranslator 处理 Excel 工作表名称的翻译
type SheetTranslator struct {
	maxConcurrentRequests int
}

// NewSheetTranslator 创建一个新的 SheetTranslator 实例
func NewSheetTranslator(maxConcurrentRequests int) *SheetTranslator {
	return &SheetTranslator{
		maxConcurrentRequests: maxConcurrentRequests,
	}
}

// TranslateSheetNames 翻译工作表名称
func (st *SheetTranslator) TranslateSheetNames(inputFile, outputFile string, translateFunc func(string) (string, error)) {
	// 打开 Excel 文件
	f, err := excelize.OpenFile(inputFile)
	if err != nil {
		fmt.Printf("打开输入 Excel 文件 '%s' 时出错: %v", inputFile, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Printf("警告: 关闭输入文件时出错: %v", err)
		}
	}()

	// 获取所有工作表名称
	sheetNames := f.GetSheetList()
	if len(sheetNames) == 0 {
		// 没有工作表，直接保存
		if err := f.SaveAs(outputFile); err != nil {
			fmt.Printf("保存输出文件 '%s' 时出错: %v", outputFile, err)
		}
	}

	// 准备翻译任务
	tasks := make([]SheetTranslationTask, len(sheetNames))
	for i, sheetName := range sheetNames {
		tasks[i] = SheetTranslationTask{
			OriginalName: sheetName,
		}
	}

	// 并发翻译工作表名称
	var wg sync.WaitGroup
	sem := semaphore.NewWeighted(int64(st.maxConcurrentRequests))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg.Add(len(tasks))

	for i := range tasks {
		go func(i int) {
			defer wg.Done()

			// 获取信号量以限制并发数
			if err := sem.Acquire(ctx, 1); err != nil {
				fmt.Printf("获取信号量失败: %v", err)
				return
			}
			defer sem.Release(1)

			// 翻译工作表名称
			translatedName, tranErr := translateFunc(tasks[i].OriginalName)
			if tranErr != nil {
				fmt.Printf("翻译工作表名称 '%s' 时出错: %v", tasks[i].OriginalName, tranErr)
				// 如果翻译失败，保持原名称
				tasks[i].NewName = tasks[i].OriginalName
			} else if translatedName != "" {
				// 处理工作表名称长度限制（Excel 最大 31 个字符）
				tasks[i].NewName = st.truncateSheetName(translatedName)
			} else {
				// 如果翻译结果为空，保持原名称
				tasks[i].NewName = tasks[i].OriginalName
			}
		}(i)
	}

	wg.Wait()

	// 重命名工作表
	for _, task := range tasks {
		if task.OriginalName != task.NewName {
			// 确保新名称不与现有名称冲突
			newName := st.ensureUniqueSheetName(f, task.NewName, task.OriginalName)

			err := f.SetSheetName(task.OriginalName, newName)
			if err != nil {
				fmt.Printf("重命名工作表 '%s' 为 '%s' 时出错: %v", task.OriginalName, newName, err)
				continue
			}
			fmt.Printf("工作表 '%s' 已重命名为 '%s'", task.OriginalName, newName)
		}
	}

	// 保存文件
	if err := f.SaveAs(outputFile); err != nil {
		fmt.Printf("保存输出文件 '%s' 时出错: %v", outputFile, err)
	}
}

// ensureUniqueSheetName 确保工作表名称唯一，避免冲突
func (st *SheetTranslator) ensureUniqueSheetName(f *excelize.File, desiredName, originalName string) string {
	existingSheets := f.GetSheetList()

	// 检查是否与现有名称冲突（除了原名称）
	for _, existingName := range existingSheets {
		if existingName == desiredName && existingName != originalName {
			// 名称冲突，添加后缀
			counter := 1
			for {
				candidateName := fmt.Sprintf("%s_%d", desiredName, counter)
				isUnique := true
				for _, existing := range existingSheets {
					if existing == candidateName && existing != originalName {
						isUnique = false
						break
					}
				}
				if isUnique {
					return candidateName
				}
				counter++
			}
		}
	}

	return desiredName
}

// truncateSheetName 截断并清理工作表名称以符合 Excel 限制
func (st *SheetTranslator) truncateSheetName(name string) string {
	const maxSheetNameLength = 31

	// 移除不允许的字符：[ ] : * ? / \
	invalidChars := regexp.MustCompile(`[\[\]:*?/\\]`)
	cleaned := invalidChars.ReplaceAllString(name, "")

	// 移除首尾的单引号
	cleaned = strings.Trim(cleaned, "'")

	// 移除首尾空格
	cleaned = strings.TrimSpace(cleaned)

	// 如果清理后为空，使用默认名称
	if cleaned == "" {
		cleaned = "Sheet"
	}

	// 截断到最大长度
	if len(cleaned) > maxSheetNameLength {
		// 优先保留字符而不是字节，处理多字节字符
		runes := []rune(cleaned)
		if len(runes) > maxSheetNameLength {
			cleaned = string(runes[:maxSheetNameLength])
		}
	}

	// 再次移除尾部空格（截断可能导致的）
	cleaned = strings.TrimSpace(cleaned)

	// 确保不为空
	if cleaned == "" {
		cleaned = "Sheet"
	}

	return cleaned
}
