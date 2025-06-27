package gui2

import (
	"exceltranslator/core"
	"fmt"
	"github.com/richardwilkes/unison/enums/side"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/richardwilkes/toolbox/fatal"
	"github.com/richardwilkes/unison"
	"github.com/richardwilkes/unison/enums/align"
)

// AppState 保存GUI的状态
type AppState struct {
	originalFilename  string           // 用户选择的原始文件名(用于显示)
	inputTempFile     string           // 实际输入文件的临时路径
	tempFile          string           // 临时输出文件路径
	translatedName    string           // 将生成的译文文件名
	status            string           // 当前状态信息
	currentOriginal   string           // 当前正在翻译的原文
	currentTranslated string           // 当前翻译的结果
	processing        bool             // 任何后台操作进行中为true
	translationDone   bool             // 标识翻译是否已完成
	processFunc       core.ProcessFunc // 翻译函数签名
}

// AppWindow 是应用程序主窗口
type AppWindow struct {
	*unison.Window
	state           *AppState
	selectFileBtn   *unison.Button
	reselectBtn     *unison.Button
	translateBtn    *unison.Button
	nextBtn         *unison.Button
	statusLabel     *unison.Label
	filenameLabel   *unison.Label
	originalLabel   *unison.Label
	translatedLabel *unison.Label
}

// CreateGUI 初始化并运行基于Unison的GUI
func CreateGUI(processFunc core.ProcessFunc) {
	unison.Start(unison.StartupFinishedCallback(func() {
		_, err := NewAppWindow(processFunc)
		fatal.IfErr(err)
	}))
}

// NewAppWindow 创建并返回一个新的窗口
func NewAppWindow(processFunc core.ProcessFunc) (*AppWindow, error) {
	// 创建窗口
	wnd, err := unison.NewWindow("Excel 翻译器")
	if err != nil {
		return nil, err
	}

	state := &AppState{
		processFunc: processFunc,
		status:      "",
	}

	app := &AppWindow{
		Window: wnd,
		state:  state,
	}

	// 初始化UI元素
	app.initUI()

	// 调整窗口大小
	app.Pack()

	// 设置窗口大小
	rect := app.FrameRect()
	rect.Width = 350  // 设置一个合适的宽度
	rect.Height = 194 // 设置一个合适的高度

	// 计算窗口居中位置
	primaryDisplay := unison.PrimaryDisplay()
	displayBounds := primaryDisplay.Usable
	rect.X = displayBounds.X + (displayBounds.Width-rect.Width)/2
	rect.Y = displayBounds.Y + (displayBounds.Height-rect.Height)/2

	app.SetFrameRect(rect)

	// 将窗口显示在最前面
	app.ToFront()

	return app, nil
}

// initUI 初始化所有UI元素
func (a *AppWindow) initUI() {
	content := a.Content()
	content.SetBorder(unison.NewEmptyBorder(unison.NewUniformInsets(10)))

	// 使用单列布局，设置水平居中
	content.SetLayout(&unison.FlexLayout{
		Columns:  1,
		HSpacing: unison.StdHSpacing,
		VSpacing: 10,
		HAlign:   align.Middle, // 所有内容水平居中
		VAlign:   align.Middle, // 所有内容垂直居中
	})

	// 初始化所有控件
	a.createLabels()
	a.createButtons()

	// 初始状态只显示选择文件按钮
	a.updateUIForCurrentState()
}

// createLabels 创建所有标签控件
func (a *AppWindow) createLabels() {
	// 状态标签
	a.statusLabel = createLabel()

	// 文件名标签
	a.filenameLabel = createLabel()

	// 原文标签
	a.originalLabel = createLabel()

	// 译文标签
	a.translatedLabel = createLabel()
}

func createLabel() *unison.Label {
	label := unison.NewLabel()
	fontDesc := label.Font.Descriptor()
	fontDesc.Size = 11
	label.Font = fontDesc.Font()
	label.HAlign = align.Middle
	return label
}

// createButtons 创建所有按钮控件
func (a *AppWindow) createButtons() {
	// 设置按钮主题
	buttonInk := &unison.ThemeColor{Light: unison.RGB(19, 122, 80), Dark: unison.RGB(19, 122, 80)}
	fontDesc := unison.SystemFont.Descriptor()
	fontDesc.Size = 11
	unison.DefaultButtonTheme = unison.ButtonTheme{
		TextDecoration: unison.TextDecoration{
			Font:            fontDesc.Font(),
			BackgroundInk:   buttonInk,
			OnBackgroundInk: unison.ThemeOnFocus,
		},
		EdgeInk:             unison.ThemeSurfaceEdge,
		SelectionInk:        buttonInk,
		OnSelectionInk:      unison.ThemeOnFocus,
		Gap:                 unison.StdIconGap,
		CornerRadius:        4,
		HMargin:             8,
		VMargin:             1,
		DrawableOnlyHMargin: 3,
		DrawableOnlyVMargin: 3,
		ClickAnimationTime:  100 * time.Millisecond,
		HAlign:              align.Middle,
		VAlign:              align.Middle,
		Side:                side.Left,
		HideBase:            false,
	}
	// 定义通用的按钮布局数据
	buttonLayoutData := &unison.FlexLayoutData{
		HAlign: align.Middle,
		VAlign: align.Middle,
	}

	// 创建一个通用的按钮尺寸设置器
	buttonSizer := func(hint unison.Size) (min, pref, max unison.Size) {
		size := unison.Size{Width: 80, Height: 30}
		return size, size, size
	}

	// 选择文件按钮
	a.selectFileBtn = unison.NewButton()
	a.selectFileBtn.SetTitle("选择文件")
	a.selectFileBtn.SetSizer(buttonSizer)
	a.selectFileBtn.ClickCallback = a.handleSelectFile
	a.selectFileBtn.SetLayoutData(buttonLayoutData)
	a.selectFileBtn.SetFocusable(false)

	// 重新选择按钮
	a.reselectBtn = unison.NewButton()
	a.reselectBtn.SetTitle("重新选择")
	a.reselectBtn.SetSizer(buttonSizer)
	a.reselectBtn.ClickCallback = a.handleSelectFile
	a.reselectBtn.SetLayoutData(buttonLayoutData)
	a.reselectBtn.SetFocusable(false)

	// 翻译按钮
	a.translateBtn = unison.NewButton()
	a.translateBtn.SetTitle("翻译")
	a.translateBtn.SetSizer(buttonSizer)
	a.translateBtn.ClickCallback = a.handleTranslate
	a.translateBtn.SetLayoutData(buttonLayoutData)
	a.translateBtn.SetFocusable(false)

	// 下一个按钮
	a.nextBtn = unison.NewButton()
	a.nextBtn.SetTitle("下一个")
	a.nextBtn.SetSizer(buttonSizer)
	a.nextBtn.ClickCallback = a.handleNext
	a.nextBtn.SetLayoutData(buttonLayoutData)
	a.nextBtn.SetFocusable(false)
}

// createCenteredSpacer 创建一个居中对齐的空白间距方法
func createCenteredSpacer(height int) *unison.Panel {
	h := float32(height)
	spacer := unison.NewPanel()
	spacer.SetSizer(func(hint unison.Size) (min, pref, max unison.Size) {
		return unison.Size{Width: 0, Height: h},
			unison.Size{Width: hint.Width, Height: h},
			unison.Size{Width: 10000, Height: h}
	})
	spacer.SetLayoutData(&unison.FlexLayoutData{
		HAlign: align.Fill,
		VAlign: align.Middle,
	})
	return spacer
}

// updateUIForCurrentState 根据当前状态更新UI显示
func (a *AppWindow) updateUIForCurrentState() {
	// 直接处理UI更新
	content := a.Content()
	content.RemoveAllChildren()

	if a.state.processing {
		// 处理中状态：显示状态和翻译进度
		a.statusLabel.SetTitle(a.state.status)
		a.statusLabel.SetLayoutData(&unison.FlexLayoutData{
			HAlign: align.Middle,
			VAlign: align.Middle,
		})
		content.AddChild(a.statusLabel)

		if a.state.currentOriginal != "" {
			// 限制显示长度，避免过长
			origText := limitTextLength(a.state.currentOriginal, 40)
			transText := limitTextLength(a.state.currentTranslated, 40)

			a.originalLabel.SetTitle("原文: " + origText)
			a.originalLabel.SetLayoutData(&unison.FlexLayoutData{
				HAlign: align.Middle,
				VAlign: align.Middle,
			})

			a.translatedLabel.SetTitle("译文: " + transText)
			a.translatedLabel.SetLayoutData(&unison.FlexLayoutData{
				HAlign: align.Middle,
				VAlign: align.Middle,
			})

			// 添加空白间距
			content.AddChild(createCenteredSpacer(10))
			content.AddChild(a.originalLabel)
			content.AddChild(a.translatedLabel)
		}
	} else if a.state.translationDone {
		// 翻译完成状态：显示状态和下一个按钮
		a.statusLabel.SetTitle(a.state.status)
		a.statusLabel.SetLayoutData(&unison.FlexLayoutData{
			HAlign: align.Middle,
			VAlign: align.Middle,
		})
		content.AddChild(a.statusLabel)

		// 添加空白间距
		content.AddChild(createCenteredSpacer(10))
		content.AddChild(a.nextBtn)
	} else if a.state.inputTempFile != "" {
		// 已选择文件状态：显示文件名和操作按钮
		a.filenameLabel.SetTitle("已选择: " + a.state.originalFilename)
		a.filenameLabel.SetLayoutData(&unison.FlexLayoutData{
			HAlign: align.Middle,
			VAlign: align.Middle,
		})
		content.AddChild(a.filenameLabel)

		if a.state.status != "" {
			a.statusLabel.SetTitle(a.state.status)
			a.statusLabel.SetLayoutData(&unison.FlexLayoutData{
				HAlign: align.Middle,
				VAlign: align.Middle,
			})
			content.AddChild(a.statusLabel)
		}

		// 添加空白间距
		content.AddChild(createCenteredSpacer(5))
		content.AddChild(a.reselectBtn)
		content.AddChild(a.translateBtn)
	} else {
		// 初始状态：显示选择文件按钮
		if a.state.status != "" {
			a.statusLabel.SetTitle(a.state.status)
			a.statusLabel.SetLayoutData(&unison.FlexLayoutData{
				HAlign: align.Middle,
				VAlign: align.Middle,
			})
			content.AddChild(a.statusLabel)

			// 添加空白间距
			content.AddChild(createCenteredSpacer(10))
		}
		content.AddChild(a.selectFileBtn)
	}

	a.MarkForRedraw()
}

// handleSelectFile 处理文件选择按钮点击
func (a *AppWindow) handleSelectFile() {
	if a.state.processing {
		return
	}

	// 清理之前的临时文件
	if a.state.tempFile != "" {
		_ = os.Remove(a.state.tempFile)
		_ = os.Remove(filepath.Dir(a.state.tempFile))
	}

	// 重置状态
	a.state.tempFile = ""
	a.state.originalFilename = ""
	a.state.translatedName = ""
	a.state.inputTempFile = ""
	a.state.translationDone = false
	a.state.currentOriginal = ""
	a.state.currentTranslated = ""

	// 更新UI状态
	a.state.processing = true
	a.state.status = "正在选择..."
	a.updateUIForCurrentState()

	// 使用InvokeTask确保在主线程上创建对话框
	unison.InvokeTask(func() {
		openDialog := unison.NewOpenDialog()
		openDialog.SetAllowsMultipleSelection(false)
		openDialog.SetAllowedExtensions("xlsx")

		if openDialog.RunModal() {
			paths := openDialog.Paths()
			if len(paths) == 0 {
				// 用户取消或出错
				a.state.status = "已取消"
			} else {
				path := paths[0]
				// 文件选择成功
				a.state.inputTempFile = path
				a.state.originalFilename = filepath.Base(path)
				a.state.translatedName = getTranslatedFilename(a.state.originalFilename)
				a.state.status = ""
			}
		} else {
			// 对话框被取消
			a.state.status = "已取消"
		}

		a.state.processing = false
		a.updateUIForCurrentState()
	})
}

// handleTranslate 处理翻译按钮点击
func (a *AppWindow) handleTranslate() {
	if a.state.processing || a.state.inputTempFile == "" || a.state.translationDone {
		return
	}

	// 设置状态
	a.state.processing = true
	a.state.status = "正在翻译..."

	// 设置临时文件路径
	a.state.tempFile = createTempFilePath(a.state.translatedName)

	// 更新UI
	a.updateUIForCurrentState()

	// 启动翻译处理
	go func() {
		err := a.state.processFunc(a.state.inputTempFile, a.state.tempFile, func(original, translated string) {
			// 确保UI更新在主线程执行
			unison.InvokeTask(func() {
				a.state.currentOriginal = original
				a.state.currentTranslated = translated
				a.updateUIForCurrentState()
			})
		})

		// 在主线程上处理结果
		unison.InvokeTask(func() {
			if err != nil {
				// 翻译失败
				a.state.status = "处理失败: " + err.Error()
				a.state.processing = false
				a.updateUIForCurrentState()
			} else {
				// 翻译成功，保存文件
				a.state.status = "翻译成功"
				a.saveTranslatedFile()
			}
		})
	}()
}

// saveTranslatedFile 保存翻译后的文件
func (a *AppWindow) saveTranslatedFile() {
	// 在主线程上显示保存对话框
	unison.InvokeTask(func() {
		saveDialog := unison.NewSaveDialog()
		saveDialog.SetInitialFileName(a.state.translatedName)

		if saveDialog.RunModal() {
			path := saveDialog.Path()
			if len(path) == 0 {
				// 用户取消
				a.state.status = "已取消保存"
				a.state.processing = false
				a.updateUIForCurrentState()
				return
			}

			a.state.status = "正在保存"
			a.updateUIForCurrentState()

			// 文件复制操作可以在后台线程进行
			go func() {
				// 将临时文件复制到选择的路径
				err := copyFile(a.state.tempFile, path)

				// 在主线程上更新UI
				unison.InvokeTask(func() {
					if err != nil {
						a.state.status = "保存失败: " + err.Error()
					} else {
						a.state.status = "保存成功"
						a.state.translationDone = true

						// 尝试删除临时文件和目录
						_ = os.Remove(a.state.tempFile)
						_ = os.Remove(filepath.Dir(a.state.tempFile))
					}

					a.state.processing = false
					a.updateUIForCurrentState()
				})
			}()
		} else {
			// 用户取消
			a.state.status = "已取消保存"
			a.state.processing = false
			a.updateUIForCurrentState()
		}
	})
}

// handleNext 处理"下一个"按钮点击
func (a *AppWindow) handleNext() {
	if a.state.processing || !a.state.translationDone {
		return
	}

	// 清理临时文件
	if a.state.tempFile != "" {
		_ = os.Remove(a.state.tempFile)
		_ = os.Remove(filepath.Dir(a.state.tempFile))
	}

	// 重置所有状态
	a.state.tempFile = ""
	a.state.originalFilename = ""
	a.state.translatedName = ""
	a.state.inputTempFile = ""
	a.state.status = ""
	a.state.translationDone = false
	a.state.currentOriginal = ""
	a.state.currentTranslated = ""

	// 更新UI
	a.updateUIForCurrentState()
}

// createTempFilePath 创建一个临时文件路径，确保目录存在
func createTempFilePath(suggestedName string) string {
	tempDir := os.TempDir()

	// 使用时间戳作为临时目录名
	timestamp := time.Now().Format("20060102-150405")
	subDir := filepath.Join(tempDir, "excel-trans-"+timestamp)

	// 确保目录存在
	_ = os.MkdirAll(subDir, 0755)

	// 使用时间戳作为临时文件名，但保留原始扩展名
	ext := filepath.Ext(suggestedName)
	return filepath.Join(subDir, "temp_"+timestamp+ext)
}

// getTranslatedFilename 获取翻译后的文件名
func getTranslatedFilename(filename string) string {
	if filename == "" {
		// 使用时间戳生成唯一文件名
		timestamp := time.Now().Format("20060102-150405")
		return "excel_" + timestamp + "_译文.xlsx"
	}

	ext := filepath.Ext(filename)
	baseWithoutExt := strings.ReplaceAll(filename, ext, "")

	// 限制基础文件名长度，避免生成过长的文件名
	runes := []rune(baseWithoutExt)
	if len(runes) > 50 {
		baseWithoutExt = string(runes[:50])
	}

	return baseWithoutExt + "_译文" + ext
}

// limitTextLength 限制文本长度，超长部分用省略号代替
func limitTextLength(text string, maxLen int) string {
	if len([]rune(text)) > maxLen {
		return string([]rune(text)[:maxLen-3]) + "..."
	}
	return text
}

// copyFile 将源文件复制到目标路径
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("复制文件内容失败: %w", err)
	}

	return nil
}
