package excel

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ShapeTranslator 处理 Excel 文件中形状的翻译
type ShapeTranslator struct {
	tempDir string
}

// NewShapeTranslator 创建一个新的 ShapeTranslator 实例
func NewShapeTranslator() *ShapeTranslator {
	return &ShapeTranslator{}
}

// TranslateShapes 处理 Excel 文件中的形状翻译
func (st *ShapeTranslator) TranslateShapes(inputFile, outputFile string, translateFunc func(string) (string, error)) error {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "excel-translator-*")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	st.tempDir = tempDir
	defer os.RemoveAll(tempDir)

	// 解压 Excel 文件
	if err := st.unzipExcel(inputFile, tempDir); err != nil {
		return fmt.Errorf("解压 Excel 文件失败: %w", err)
	}

	// 处理 drawings 目录
	drawingsDir := filepath.Join(tempDir, "xl", "drawings")
	if err := st.processDrawings(drawingsDir, translateFunc); err != nil {
		return fmt.Errorf("处理 drawings 失败: %w", err)
	}

	// 重新打包为 Excel 文件
	if err := st.zipExcel(tempDir, outputFile); err != nil {
		return fmt.Errorf("重新打包 Excel 文件失败: %w", err)
	}

	return nil
}

// unzipExcel 解压 Excel 文件到指定目录
func (st *ShapeTranslator) unzipExcel(inputFile, destDir string) error {
	r, err := zip.OpenReader(inputFile)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// 获取目标文件路径
		fpath := filepath.Join(destDir, f.Name)

		// 检查文件是否在目标目录内
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("非法文件路径: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// 确保目录存在
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		// 创建目标文件
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		// 打开源文件
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		// 复制内容
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// processDrawings 处理 drawings 目录中的所有 drawing*.xml 文件
func (st *ShapeTranslator) processDrawings(drawingsDir string, translateFunc func(string) (string, error)) error {
	// 查找所有 drawing*.xml 文件
	files, err := filepath.Glob(filepath.Join(drawingsDir, "drawing*.xml"))
	if err != nil {
		return err
	}

	// 正则表达式匹配 <a:t>xxx</a:t>
	re := regexp.MustCompile(`<a:t>(.*?)</a:t>`)

	for _, file := range files {
		// 读取文件内容
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("读取文件 %s 失败: %w", file, err)
		}

		// 替换所有匹配的文本
		newContent := re.ReplaceAllStringFunc(string(content), func(match string) string {
			// 提取原始文本
			text := re.FindStringSubmatch(match)[1]
			if text == "" {
				return match
			}

			// 调用翻译函数
			translated, err := translateFunc(text)
			if err != nil {
				fmt.Printf("翻译文本 '%s' 失败: %v\n", text, err)
				return match
			}

			// 返回替换后的内容
			return fmt.Sprintf("<a:t>%s</a:t>", translated)
		})

		// 写回文件
		if err := os.WriteFile(file, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("写入文件 %s 失败: %w", file, err)
		}
	}

	return nil
}

// zipExcel 将目录重新打包为 Excel 文件
func (st *ShapeTranslator) zipExcel(sourceDir, outputFile string) error {
	// 创建输出文件
	outFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// 创建 zip writer
	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	// 遍历源目录
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 获取相对路径
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// 跳过临时目录本身
		if relPath == "." {
			return nil
		}

		// 创建文件头
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// 设置文件名（使用正斜杠作为分隔符）
		header.Name = filepath.ToSlash(relPath)

		// 如果是目录，添加斜杠
		if info.IsDir() {
			header.Name += "/"
		} else {
			// 设置压缩方法
			header.Method = zip.Deflate
			// 保持原始文件的修改时间
			header.Modified = info.ModTime()
		}

		// 创建写入器
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// 如果是文件，写入内容
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(writer, file)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
