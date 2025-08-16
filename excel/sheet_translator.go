package excel

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/xuri/excelize/v2"
)

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

// TranslateSheetNames 翻译工作表名称（同步执行）
func (st *SheetTranslator) TranslateSheetNames(ctx context.Context, inputFile, outputFile string, translateFunc func(string) (string, error)) error {
	f, err := excelize.OpenFile(inputFile)
	if err != nil {
		return fmt.Errorf("打开输入 Excel 文件 '%s' 时出错: %w", inputFile, err)
	}
	defer f.Close()

	sheetNames := f.GetSheetList()

	// 翻译工作表名称
	for _, sheetName := range sheetNames {
		translatedName, tranErr := translateFunc(sheetName)
		if tranErr != nil {
			if !errors.Is(tranErr, context.Canceled) {
				fmt.Printf("翻译工作表名称 '%s' 时出错: %v", sheetName, tranErr)
			}
			continue
		}

		if translatedName != "" && translatedName != sheetName {
			// 处理工作表名称长度限制
			newName := st.truncateSheetName(translatedName)
			uniqueName := st.ensureUniqueSheetName(f, newName, sheetName)

			if err := f.SetSheetName(sheetName, uniqueName); err != nil {
				return fmt.Errorf("重命名工作表 '%s' 为 '%s' 时出错: %w", sheetName, uniqueName, err)
			} else {
				fmt.Printf("工作表 '%s' 已重命名为 '%s'\n", sheetName, uniqueName)
			}
		}
	}

	return f.SaveAs(outputFile)
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
