package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"ooxmltranslator/pkg/config"
	"ooxmltranslator/pkg/runner"
)

// maxLogLines 是日志缓冲区保留的最大行数，超出后从头部裁剪。
const maxLogLines = 500

// app 保存界面的全部状态。
// 主线程字段只在视图、菜单和设置弹窗中读写；翻译 goroutine 只通过
// mu 保护后台字段，或通过 win.Update 回到主线程写入界面状态。
type app struct {
	win *mygo.Window
	cfg *config.AppConfig

	// 主线程字段
	inputFile   string
	lastOpenDir string
	lastSaveDir string
	aboutOpen   bool
	logScroll   ui.ScrollState
	settingsWin *mygo.Window
	// presentSettings 打开偏好设置窗口；测试中替换为空函数，避免创建真实窗口。
	presentSettings func()
	// alert 显示提示窗口；测试中替换为记录函数，避免打开真实的系统弹窗。
	alert func(title, message string)

	// 设置弹窗的草稿，取消时不污染 cfg
	settingsDraft       config.AppConfig
	settingsConcurrency float64
	settingsError       string

	// 可注入的配置保存函数，便于测试
	saveConfigFn func(*config.AppConfig) error

	// 翻译 goroutine 写入，视图读取；由 mu 保护
	mu          sync.Mutex
	runID       uint64
	translating bool
	progress    float64
	progressErr bool
	logs        []string
	cancel      context.CancelFunc
}

// runContext 是一次翻译运行所需的不可变快照，避免后台 goroutine
// 读取主线程字段。
type runContext struct {
	id        uint64
	inputFile string
	tempFile  string
	saveDir   string
	cfg       config.AppConfig
}

func newApp(cfg *config.AppConfig) *app {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return &app{cfg: cfg, settingsConcurrency: float64(cfg.Translation.MaxConcurrentRequests)}
}

func (a *app) invalidate() {
	if a.win != nil {
		a.win.Invalidate()
	}
}

// isOfficeFile 判断路径是否为支持的 Office 文件。
func isOfficeFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".xlsx", ".docx", ".pptx":
		return true
	}
	return false
}

// acceptFile 接收用户选择或拖入的文件，校验扩展名后写入界面状态。
func (a *app) acceptFile(path string) {
	if path == "" {
		return
	}
	if !isOfficeFile(path) {
		a.showAlert(mygo.MessageWarning, "不支持的文件",
			"请选择 Office 文件（.xlsx / .docx / .pptx）")
		return
	}

	a.inputFile = path
	a.lastOpenDir = filepath.Dir(path)

	a.mu.Lock()
	a.progress = 0
	a.progressErr = false
	if !a.translating {
		a.logs = nil
	}
	a.mu.Unlock()

	a.invalidate()
}

// startDir 返回文件对话框的起始目录。
func (a *app) startDir(forSave bool) string {
	if forSave && a.lastSaveDir != "" {
		return a.lastSaveDir
	}
	if a.lastOpenDir != "" {
		return a.lastOpenDir
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = os.Getenv("HOME")
	}
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home == "" {
		return ""
	}

	downloads := filepath.Join(home, "Downloads")
	if info, err := os.Stat(downloads); err == nil && info.IsDir() {
		return downloads
	}
	return home
}

func (a *app) appendLog(id uint64, message string) {
	line := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), message)

	a.mu.Lock()
	if a.runID != id {
		a.mu.Unlock()
		return
	}
	a.logs = append(a.logs, line)
	if len(a.logs) > maxLogLines {
		a.logs = append([]string(nil), a.logs[len(a.logs)-maxLogLines:]...)
	}
	a.mu.Unlock()

	a.invalidate()
}

func (a *app) setProgress(id uint64, done, total int) {
	if total <= 0 {
		return
	}
	p := float64(done) / float64(total)
	if p < 0 {
		p = 0
	} else if p > 1 {
		p = 1
	}

	a.mu.Lock()
	if a.runID != id {
		a.mu.Unlock()
		return
	}
	a.progress = p
	a.mu.Unlock()

	a.invalidate()
}

func (a *app) startTranslation() {
	a.mu.Lock()
	if a.translating {
		a.mu.Unlock()
		a.showAlert(mygo.MessageInfo, "翻译进行中", "请等待当前翻译完成")
		return
	}
	if a.inputFile == "" {
		a.mu.Unlock()
		a.showAlert(mygo.MessageWarning, "未选择文件", "请先选择要翻译的文件")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	rc := runContext{
		inputFile: a.inputFile,
		tempFile:  tempOutputPath(a.inputFile),
		saveDir:   a.startDir(true),
		cfg:       *a.cfg,
	}
	a.runID++
	rc.id = a.runID
	a.translating = true
	a.progress = 0
	a.progressErr = false
	a.logs = nil
	a.cancel = cancel
	a.mu.Unlock()

	a.appendLog(rc.id, "开始翻译…")

	go func() {
		defer cancel()

		var once sync.Once
		complete := func(err error) {
			once.Do(func() { a.onTranslationDone(rc, err) })
		}

		runErr := runner.RunTranslationWithConfig(ctx, rc.inputFile, rc.tempFile, &rc.cfg, runner.TranslationCallbacks{
			OnTranslated: func(original, translated string) {
				a.appendLog(rc.id, formatTranslated(original, translated))
			},
			OnOverallProgress: func(done, total int) {
				a.setProgress(rc.id, done, total)
			},
			OnError: func(stage string, err error) {
				if errors.Is(err, context.Canceled) {
					return
				}
				if stage == "llm" {
					a.appendLog(rc.id, "翻译模型调用失败，请检查模型配置")
					return
				}
				a.appendLog(rc.id, fmt.Sprintf("翻译失败（阶段: %s）", stage))
			},
			OnComplete: complete,
		})

		// runner 在正常路径和错误路径都会调用 OnComplete；这里兜底，
		// 保证即使将来流程变化也不会留下“翻译中”的界面状态。
		complete(runErr)
	}()
}

func (a *app) stopTranslation() {
	a.mu.Lock()
	if !a.translating {
		a.mu.Unlock()
		return
	}
	a.translating = false
	cancel := a.cancel
	id := a.runID
	a.mu.Unlock()

	a.appendLog(id, "用户停止翻译，正在清理资源…")
	if cancel != nil {
		cancel()
	}
	a.invalidate()
}

func (a *app) onTranslationDone(rc runContext, err error) {
	a.mu.Lock()
	if a.runID != rc.id {
		// 用户在本次运行结束前又开始了新的翻译，旧结果直接丢弃。
		a.mu.Unlock()
		cleanupPath(rc.tempFile)
		return
	}
	a.translating = false
	canceled := errors.Is(err, context.Canceled)
	if err == nil {
		a.progress = 1
		a.progressErr = false
	} else if !canceled {
		a.progressErr = true
	}
	a.mu.Unlock()

	a.invalidate()

	switch {
	case canceled:
		a.appendLog(rc.id, "翻译已停止")
		cleanupPath(rc.tempFile)

	case err != nil:
		message := friendlyError(err)
		a.appendLog(rc.id, "翻译失败: "+message)
		_, _ = mygo.Dialog.Message(mygo.MessageOptions{
			Parent:  a.win,
			Type:    mygo.MessageError,
			Title:   "错误",
			Message: "翻译失败: " + message,
		})
		cleanupPath(rc.tempFile)

	default:
		a.appendLog(rc.id, "翻译完成!")
		a.promptSave(rc)
	}
}

// promptSave 在翻译完成后弹出原生保存对话框。它运行在翻译 goroutine 中，
// 因此可以直接调用会阻塞的 Dialog.Save。
func (a *app) promptSave(rc runContext) {
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent:      a.win,
		Title:       "保存翻译后的文件",
		DefaultPath: defaultSavePath(rc.inputFile, rc.saveDir),
		Filters: []mygo.FileFilter{{
			Name:       "Office 文件",
			Extensions: []string{"xlsx", "docx", "pptx"},
		}},
	})
	if err != nil || path == "" {
		cleanupPath(rc.tempFile)
		a.win.Update(func() {
			a.showAlert(mygo.MessageInfo, "翻译已完成", "已取消保存，翻译结果未写入文件")
		})
		return
	}

	if err := copyFile(rc.tempFile, path); err != nil {
		cleanupPath(rc.tempFile)
		_, _ = mygo.Dialog.Message(mygo.MessageOptions{
			Parent:  a.win,
			Type:    mygo.MessageError,
			Title:   "错误",
			Message: "保存文件失败: " + err.Error(),
		})
		return
	}

	cleanupPath(rc.tempFile)
	dir := filepath.Dir(path)
	a.win.Update(func() {
		a.lastSaveDir = dir
	})
}

// showAlert 显示系统提示窗口，代替容易被忽略的底部 toast。
func (a *app) showAlert(kind mygo.MessageType, title, message string) {
	if a.alert != nil {
		a.alert(title, message)
		return
	}
	_, _ = mygo.Dialog.Message(mygo.MessageOptions{
		Parent:  a.win,
		Type:    kind,
		Title:   title,
		Message: message,
	})
}

func (a *app) chooseInputFile() {
	// 在主线程读取起始目录，避免与拖拽写入 lastOpenDir 产生数据竞争。
	startDir := a.startDir(false)

	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:      a.win,
			Title:       "选择 Office 文件",
			DefaultPath: startDir,
			Filters: []mygo.FileFilter{{
				Name:       "Office 文件",
				Extensions: []string{"xlsx", "docx", "pptx"},
			}},
		})
		if err != nil || len(paths) == 0 {
			return
		}
		a.win.Update(func() { a.acceptFile(paths[0]) })
	}()
}

func (a *app) snapshot() (translating bool, progress float64, progressErr bool, logs []string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.translating, a.progress, a.progressErr, append([]string(nil), a.logs...)
}

func tempOutputPath(inputFile string) string {
	base := filepath.Base(inputFile)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s_translated_%d%s", name, time.Now().UnixNano(), ext))
}

func defaultSavePath(inputFile, dir string) string {
	base := filepath.Base(inputFile)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext) + "_译文" + ext
	if dir == "" {
		return name
	}
	return filepath.Join(dir, name)
}

func formatTranslated(original, translated string) string {
	return truncateRunes(original, 200) + " -> " + truncateRunes(translated, 200)
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

func friendlyError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "翻译超时，请检查网络连接或重试"
	}
	return err.Error()
}

func cleanupPath(path string) {
	if path == "" {
		return
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		// 临时文件清理失败不应打断用户流程
		fmt.Fprintf(os.Stderr, "清理临时文件失败: %v\n", err)
	}
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}
