package excel

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/sync/semaphore"
)

// ShapeTranslator 处理 Excel 文件中形状的翻译
type ShapeTranslator struct {
	maxConcurrentRequests int
}

// NewShapeTranslator 创建一个新的 ShapeTranslator 实例
func NewShapeTranslator(maxConcurrentRequests int) *ShapeTranslator {
	return &ShapeTranslator{
		maxConcurrentRequests: maxConcurrentRequests,
	}
}

// TranslateShapes 处理 Excel 文件中的形状翻译
func (st *ShapeTranslator) TranslateShapes(inputFile, outputFile string, translateFunc func(string) (string, error)) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "excel-translator-*")
	if err != nil {
		log.Printf("创建临时目录失败: %v", err)
		return
	}
	defer os.RemoveAll(tempDir)

	// 解压 Excel 文件
	if err := st.UnzipExcel(inputFile, tempDir); err != nil {
		log.Printf("解压 Excel 文件失败: %v", err)
		return
	}

	// 处理 drawings 目录
	drawingsDir := filepath.Join(tempDir, "xl", "drawings")
	if err := st.ProcessDrawings(drawingsDir, translateFunc); err != nil {
		log.Printf("处理 drawings 失败: %v", err)
		return
	}

	// 重新打包为 Excel 文件
	if err := st.ZipExcel(tempDir, outputFile); err != nil {
		log.Printf("重新打包 Excel 文件失败: %v", err)
		return
	}
}

// UnzipExcel 解压 Excel 文件到指定目录
func (st *ShapeTranslator) UnzipExcel(inputFile, destDir string) error {
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

// ZipExcel 将目录重新打包为 Excel 文件
func (st *ShapeTranslator) ZipExcel(sourceDir, outputFile string) error {
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

// ProcessDrawings 处理 drawings 目录中的所有 drawing*.xml 文件
func (st *ShapeTranslator) ProcessDrawings(drawingsDir string, translateFunc func(string) (string, error)) error {
	files, err := filepath.Glob(filepath.Join(drawingsDir, "drawing*.xml"))
	if err != nil {
		return err
	}

	for _, file := range files {
		err := st.TranslateDrawingFile(file, translateFunc)
		if err != nil {
			return fmt.Errorf("处理文件 %s 失败: %w", file, err)
		}
	}
	return nil
}

// TranslateDrawingFile 翻译 drawing*.xml 文件
// TranslateDrawingFile 异步翻译 drawing*.xml 文件中的 <a:t> 标签内容
func (st *ShapeTranslator) TranslateDrawingFile(file string, translateFunc func(string) (string, error)) error {
	re := regexp.MustCompile(`<a:t>(.*?)</a:t>`)

	// 读取原始文件内容
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("读取文件 %s 失败: %w", file, err)
	}
	strContent := string(content)

	// 匹配所有标签内容
	matches := re.FindAllStringSubmatchIndex(strContent, -1)
	if len(matches) == 0 {
		log.Printf("文件 %s 中未找到需要翻译的文本。\n", file)
		return nil
	}

	type TranslatedResult struct {
		start, end int
		translated string
	}

	results := make([]TranslatedResult, len(matches))

	wg := sync.WaitGroup{}
	sem := semaphore.NewWeighted(int64(st.maxConcurrentRequests))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg.Add(len(matches))

	for i, match := range matches {
		go func(i int, start, end int) {
			defer wg.Done()
			// 获取信号量以限制并发数
			if err := sem.Acquire(ctx, 1); err != nil {
				log.Printf("获取信号量失败: %v\n", err)
				return
			}
			defer sem.Release(1)

			original := strContent[start:end]
			text := strContent[match[2]:match[3]]

			translated, err := translateFunc(text)
			if err != nil {
				log.Printf("翻译文本 '%s' (文件: %s) 失败: %v\n", text, file, err)
				results[i] = TranslatedResult{start, end, original}
				return
			}

			// 构造替换内容
			results[i] = TranslatedResult{start, end, fmt.Sprintf("<a:t>%s</a:t>", translated)}
		}(i, match[0], match[1])
	}

	wg.Wait()

	// 替换内容（倒序替换避免索引错位）
	var builder strings.Builder
	last := 0
	for _, r := range results {
		builder.WriteString(strContent[last:r.start])
		builder.WriteString(r.translated)
		last = r.end
	}
	builder.WriteString(strContent[last:])

	// 写入文件
	if err := os.WriteFile(file, []byte(builder.String()), 0644); err != nil {
		return fmt.Errorf("写入文件 %s 失败: %w", file, err)
	}

	log.Printf("文件 %s 处理完成。\n", file)
	return nil
}
