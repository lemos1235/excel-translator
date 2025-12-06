package word

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/semaphore"
)

// DocumentTranslator 处理 Word 文件的翻译
type DocumentTranslator struct {
	maxConcurrentRequests int
	ctx                   context.Context
	translateFunc         func(string) (string, error)
}

// NewDocumentTranslator 创建一个新的 DocumentTranslator 实例
func NewDocumentTranslator(maxConcurrentRequests int, ctx context.Context, translateFunc func(string) (string, error)) *DocumentTranslator {
	return &DocumentTranslator{
		maxConcurrentRequests: maxConcurrentRequests,
		ctx:                   ctx,
		translateFunc:         translateFunc,
	}
}

// TranslateDocument 处理 Word 文件的翻译
func (st *DocumentTranslator) TranslateDocument(
	inputFile,
	outputFile string,
	onProgress func(original, translated string, err error, done, total int),
) error {
	// 检查上下文是否已取消
	select {
	case <-st.ctx.Done():
		return st.ctx.Err()
	default:
	}

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "word-translator-*")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 解压 Word 文件
	if err := st.UnzipWord(inputFile, tempDir); err != nil {
		return fmt.Errorf("解压 Word 文件失败: %w", err)
	}

	// 检查上下文是否已取消
	select {
	case <-st.ctx.Done():
		return st.ctx.Err()
	default:
	}

	// 处理 document.xml 文件
	documentXmlFile := filepath.Join(tempDir, "word", "document.xml")
	if err := st.TranslateDocumentXmlFile(documentXmlFile, onProgress); err != nil {
		return err
	}

	// 检查上下文是否已取消
	select {
	case <-st.ctx.Done():
		return st.ctx.Err()
	default:
	}

	// 重新打包为 Word 文件
	if err := st.ZipWord(tempDir, outputFile); err != nil {
		return fmt.Errorf("重新打包 Word 文件失败: %w", err)
	}

	return nil
}

// UnzipWord 解压 Word 文件到指定目录
func (st *DocumentTranslator) UnzipWord(inputFile, destDir string) error {
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

// ZipWord 将目录重新打包为 Word 文件
func (st *DocumentTranslator) ZipWord(sourceDir, outputFile string) error {
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

// TranslateDocumentXmlFile 翻译 document.xml 文件
func (st *DocumentTranslator) TranslateDocumentXmlFile(
	filePath string,
	onProgress func(original, translated string, err error, done, total int),
) error {
	// 检查上下文是否已取消
	select {
	case <-st.ctx.Done():
		return st.ctx.Err()
	default:
	}

	re := regexp.MustCompile(`<w:t[^>]*>(.*?)</w:t>`)

	// 读取原始文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件 %s 失败: %w", filePath, err)
	}
	strContent := string(content)

	// 匹配所有标签内容
	matches := re.FindAllStringSubmatchIndex(strContent, -1)
	total := len(matches)
	if total == 0 {
		log.Printf("文件 %s 中未找到需要翻译的文本。\n", filePath)
		return nil
	}

	type TranslatedResult struct {
		start, end int
		translated string
	}

	results := make([]TranslatedResult, len(matches))

	// 初始化所有结果为原始内容
	for i, match := range matches {
		original := strContent[match[0]:match[1]]
		results[i] = TranslatedResult{match[0], match[1], original}
	}

	wg := sync.WaitGroup{}
	sem := semaphore.NewWeighted(int64(st.maxConcurrentRequests))

	// 使用 context 的子 context 来控制 goroutine
	childCtx, childCancel := context.WithCancel(st.ctx)
	defer childCancel()

	wg.Add(len(matches))
	var doneCount int64

	// 错误通道
	errCh := make(chan error, 1)

	for i, match := range matches {
		go func(i int, start, end int) {
			defer wg.Done()

			// 首先检查上下文是否已取消
			select {
			case <-childCtx.Done():
				return
			default:
			}

			// 获取信号量
			if err := sem.Acquire(childCtx, 1); err != nil {
				return
			}
			defer sem.Release(1)

			// 再次检查上下文
			select {
			case <-childCtx.Done():
				return
			default:
			}

			text := strContent[match[2]:match[3]]

			translated, tranErr := st.translateFunc(text)
			current := int(atomic.AddInt64(&doneCount, 1))

			if tranErr != nil {
				if !errors.Is(tranErr, context.Canceled) {
					log.Printf("翻译文本 '%s' (文件: %s) 失败: %v\n", text, filePath, tranErr)
				}

				if onProgress != nil {
					onProgress(text, "", tranErr, current, total)
				}

				childCancel()
				select {
				case errCh <- tranErr:
				default:
				}
				return
			}

			// 构造替换内容，对翻译结果进行XML转义
			escapedTranslated := html.EscapeString(translated)
			results[i] = TranslatedResult{start, end, fmt.Sprintf("<w:t>%s</w:t>", escapedTranslated)}

			if onProgress != nil {
				onProgress(text, translated, nil, current, total)
			}
		}(i, match[0], match[1])
	}

	// 等待完成
	wg.Wait()
	close(errCh)

	// 检查是否有错误
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	default:
	}

	// 检查上下文
	select {
	case <-st.ctx.Done():
		return st.ctx.Err()
	default:
	}

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
	if err := os.WriteFile(filePath, []byte(builder.String()), 0644); err != nil {
		return fmt.Errorf("写入文件 %s 失败: %w", filePath, err)
	}

	log.Printf("文件 %s 处理完成。\n", filePath)
	return nil
}
