package gui

import (
	"errors"
	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"
	"image"
	"image/color"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileOpType 指定文件操作的类型
type FileOpType int

const (
	FileOpChoose FileOpType = iota // 选择文件
	FileOpSave                     // 保存翻译结果
)

// explorerResult 保存文件选择/创建操作的结果
type explorerResult struct {
	closer io.Closer // ReadCloser 或 WriteCloser
	err    error
	opType FileOpType // 区分选择文件和保存文件
	path   string     // 保存文件结果的路径（可能是占位符）
}

// guiState 保存GUI的状态
type guiState struct {
	theme             *material.Theme
	selectBtn         widget.Clickable
	translateBtn      widget.Clickable
	deleteBtn         widget.Clickable // 添加删除按钮
	nextBtn           widget.Clickable // 添加下一个按钮
	originalFilename  string           // 用户选择的原始文件名(用于显示)
	inputTempFile     string           // 实际输入文件的临时路径
	tempFile          string           // 临时输出文件路径
	translatedName    string           // 将生成的译文文件名
	savedFilePath     string           // 保存的文件路径
	status            string
	currentOriginal   string                                                                                   // 当前正在翻译的原文
	currentTranslated string                                                                                   // 当前翻译的结果
	processing        bool                                                                                     // 任何后台操作进行中为true
	savePending       bool                                                                                     // 标识翻译完成后需要弹出保存对话框
	translationDone   bool                                                                                     // 标识翻译是否已完成
	processFunc       func(inputFile, outputFile string, onTranslated func(original, translated string)) error // 翻译函数签名
	window            *app.Window
	explorerInst      *explorer.Explorer
	fileOpResultChan  chan explorerResult // 文件选择/创建结果的通道
	processResultChan chan error          // 翻译处理结果的通道
	initialized       bool
	translationChan   chan struct { // 翻译内容更新通道
		Original   string
		Translated string
	}
}

// CreateGUI 初始化并运行GUI
func CreateGUI(processFunc func(inputFile, outputFile string, onTranslated func(original, translated string)) error) {
	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("Excel 翻译器"),
			app.Size(unit.Dp(350), unit.Dp(250)), // 减小窗口高度使界面更紧凑
		)
		// 创建自定义主题以设置背景色
		t := material.NewTheme()
		t.Bg = color.NRGBA{R: 0xE8, G: 0xE8, B: 0xE8, A: 0xFF} // 设置背景色为 #E8E8E8

		state := guiState{
			theme:             t,
			processFunc:       processFunc,
			status:            "", // 初始不显示状态提示
			window:            w,
			fileOpResultChan:  make(chan explorerResult, 1),
			processResultChan: make(chan error, 1),
			savePending:       false,
			inputTempFile:     "",
			translationDone:   false,
			translationChan: make(chan struct {
				Original   string
				Translated string
			}, 10), // 缓冲10条翻译消息
			currentOriginal:   "",
			currentTranslated: "",
		}

		if err := run(&state); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
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

// run 实现GUI的主循环
func run(state *guiState) error {
	var ops op.Ops
	state.explorerInst = explorer.NewExplorer(state.window)

	for {
		// 处理通道中的结果
		select {
		case res := <-state.fileOpResultChan: // 处理文件选择/创建结果
			if res.err != nil {
				state.processing = false // 操作失败或取消
				if errors.Is(res.err, explorer.ErrUserDecline) {
					state.status = "已取消"
					state.tempFile = ""
					state.inputTempFile = ""
					state.originalFilename = ""
					state.translatedName = ""
					state.currentOriginal = ""
					state.currentTranslated = ""
				} else {
					state.status = "操作失败: " + res.err.Error()
				}
				// 如果是保存操作失败/取消，重置状态以允许再次尝试保存
				if res.opType == FileOpSave {
					state.savePending = true
				}
				state.window.Invalidate()
				continue
			}

			// 根据操作类型处理成功的结果
			if res.opType == FileOpChoose {
				// 尝试将ReadCloser转换为os.File类型
				reader := res.closer.(io.ReadCloser)

				// 检查是否为*os.File类型
				file, ok := reader.(*os.File)
				if ok {
					state.processing = false
					state.inputTempFile = file.Name()
					state.originalFilename = filepath.Base(file.Name())
					state.translatedName = getTranslatedFilename(state.originalFilename)
					state.status = "" // 不显示状态，因为文件名会显示在界面上
					state.window.Invalidate()
				} else {
					reader.Close() // 关闭读取器
					state.processing = false
					state.status = "不支持的文件类型"
					state.window.Invalidate()
				}
			} else if res.opType == FileOpSave {
				state.savePending = false
				state.status = "正在保存"

				// 尝试将WriteCloser转换为os.File类型
				writer := res.closer.(io.WriteCloser)
				file, ok := writer.(*os.File)

				if ok {
					outputPath := file.Name()
					state.savedFilePath = outputPath

					go func(tempFile, outputPath string) {
						// 将临时翻译结果复制到目标路径
						srcFile, err := os.Open(tempFile)
						if err != nil {
							state.processResultChan <- err
							return
						}
						defer srcFile.Close()

						_, err = io.Copy(file, srcFile)
						file.Close() // 关闭文件

						if err != nil {
							state.processResultChan <- err
							return
						}

						// 保存成功，发送信号
						state.processResultChan <- nil

						// 尝试删除临时文件和目录
						_ = os.Remove(tempFile)
						_ = os.Remove(filepath.Dir(tempFile))
					}(state.tempFile, outputPath)
				} else {
					writer.Close() // 关闭写入器
					state.processing = false
					state.savePending = true // 允许用户再次尝试保存
					state.status = "保存失败，请重试"
					state.window.Invalidate()
				}
			}
			state.window.Invalidate()

		case transMsg := <-state.translationChan: // 处理翻译内容更新
			state.currentOriginal = transMsg.Original
			state.currentTranslated = transMsg.Translated
			state.window.Invalidate()

		case processErr := <-state.processResultChan: // 处理翻译或保存结果
			if processErr != nil {
				// 处理失败
				state.status = "处理失败: " + processErr.Error()
				state.processing = false
				state.savePending = false // 重置状态
				state.translationDone = false
			} else if state.tempFile != "" && state.savePending {
				// 翻译成功，临时文件已创建，弹出保存对话框
				state.status = "翻译成功"
				// 清除文件名显示
				state.originalFilename = ""

				// 使用已计算好的译文文件名
				suggestedName := state.translatedName

				// 弹出保存对话框
				go func() {
					wc, err := state.explorerInst.CreateFile(suggestedName)
					state.fileOpResultChan <- explorerResult{
						closer: wc,
						err:    err,
						opType: FileOpSave,
						path:   "output.xlsx",
					}
				}()
			} else {
				// 保存操作完成
				log.Println("文件已成功保存")
				state.status = "保存成功" // 显示保存成功提示
				state.processing = false
				state.savePending = false
				state.translationDone = true // 标记翻译已完成
				// 清理临时文件状态
				state.tempFile = ""
				// 重置界面内容，允许用户选择新文件
				state.originalFilename = ""
				state.translatedName = ""
				state.inputTempFile = ""
				state.savedFilePath = ""
			}
			state.window.Invalidate()

		default:
			// 没有来自通道的结果，继续处理事件
		}

		// 获取Gio事件
		e := state.window.Event()

		// 调用explorer.ListenEvents
		state.explorerInst.ListenEvents(e)

		// 处理Gio事件
		switch e := e.(type) {
		case app.DestroyEvent:
			// 清理临时文件
			if state.tempFile != "" {
				_ = os.Remove(state.tempFile)
				_ = os.Remove(filepath.Dir(state.tempFile))
			}
			return e.Err

		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			// 添加一个初始化标志，确保只居中一次
			if !state.initialized {
				state.window.Perform(system.ActionCenter)
				state.initialized = true
			}

			// 仅当不在处理中时处理按钮点击
			if !state.processing {
				// 选择文件按钮
				if state.selectBtn.Clicked(gtx) {
					// 仅清理临时文件
					if state.tempFile != "" {
						_ = os.Remove(state.tempFile)
						_ = os.Remove(filepath.Dir(state.tempFile))
					}

					// 重置之前的状态
					state.tempFile = ""
					state.originalFilename = ""
					state.translatedName = ""
					state.inputTempFile = ""
					state.savePending = false
					state.translationDone = false

					state.processing = true // 等待文件选择结果
					state.status = "正在选择..."
					go func() {
						rc, err := state.explorerInst.ChooseFile(".xlsx")
						state.fileOpResultChan <- explorerResult{
							closer: rc,
							err:    err,
							opType: FileOpChoose,
						}
					}()
				}

				// 重新选择按钮 - 打开文件选择框
				if state.deleteBtn.Clicked(gtx) && state.inputTempFile != "" {
					// 清理临时文件
					if state.tempFile != "" {
						_ = os.Remove(state.tempFile)
						_ = os.Remove(filepath.Dir(state.tempFile))
					}

					// 重置状态
					state.tempFile = ""
					state.originalFilename = ""
					state.translatedName = ""
					state.inputTempFile = ""
					state.savePending = false
					state.translationDone = false

					// 打开文件选择框
					state.processing = true
					state.status = "正在选择..."
					go func() {
						rc, err := state.explorerInst.ChooseFile(".xlsx")
						state.fileOpResultChan <- explorerResult{
							closer: rc,
							err:    err,
							opType: FileOpChoose,
						}
					}()
				}

				// 下一个按钮 - 重置所有状态，相当于重新开始
				if state.nextBtn.Clicked(gtx) && state.translationDone {
					// 清理临时文件
					if state.tempFile != "" {
						_ = os.Remove(state.tempFile)
						_ = os.Remove(filepath.Dir(state.tempFile))
					}

					// 重置所有状态
					state.tempFile = ""
					state.originalFilename = ""
					state.translatedName = ""
					state.inputTempFile = ""
					state.savePending = false
					state.translationDone = false
					state.status = ""
					state.currentOriginal = ""
					state.currentTranslated = ""
					state.window.Invalidate()
				}

				// 翻译按钮
				// 确保只有在已选择文件时才允许翻译
				hasInputFile := state.inputTempFile != ""
				if state.translateBtn.Clicked(gtx) && hasInputFile && !state.translationDone {
					state.processing = true  // 标记为处理中
					state.savePending = true // 翻译完成后需要弹出保存对话框
					state.status = "正在翻译..."
					// 清除文件名显示
					state.originalFilename = ""

					// 设置临时文件路径，使用已计算好的译文文件名
					state.tempFile = createTempFilePath(state.translatedName)

					// 在goroutine中执行翻译
					go func(inputFile, tempFile string) {
						err := state.processFunc(inputFile, tempFile, func(original, translated string) {
							state.currentOriginal = original
							state.currentTranslated = translated
							log.Printf("original %s, translated %s", original, translated)
							state.window.Invalidate()
						})
						state.processResultChan <- err
						// 立即强制刷新窗口
						state.window.Invalidate()
					}(state.inputTempFile, state.tempFile)
				}
			}

			// 渲染UI
			renderUI(gtx, state)

			e.Frame(gtx.Ops)

		// 允许使用Esc键关闭窗口
		case key.Event:
			if e.State == key.Press && e.Name == key.NameEscape {
				return nil
			}
		}
	}
}

// renderUI 渲染界面
func renderUI(gtx layout.Context, state *guiState) {
	// 绘制背景色
	drawBackground(gtx, state.theme.Bg)

	// 如果正在处理中，只显示状态文本，不显示任何按钮
	if state.processing {
		layout.Stack{Alignment: layout.Center}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				return layout.Dimensions{Size: gtx.Constraints.Max}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis:      layout.Vertical,
					Spacing:   layout.SpaceStart,
					Alignment: layout.Middle,
				}.Layout(gtx,
					// 状态提示文字
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(state.theme, 15, state.status)
						lbl.Alignment = text.Middle
						lbl.Color = color.NRGBA{R: 50, G: 50, B: 50, A: 255}
						return layout.Inset{Bottom: unit.Dp(20)}.Layout(gtx, lbl.Layout)
					}),

					// 显示翻译内容（如果有）
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if state.currentOriginal != "" {
							// 限制显示长度，确保只显示一行
							origText := state.currentOriginal
							transText := state.currentTranslated

							// 获取窗口宽度，预留边距和文本前缀("原文: "和"译文: ")
							maxWidth := gtx.Constraints.Max.X
							prefixWidth := gtx.Dp(40)                               // 估计"原文: "和"译文: "的宽度
							availableWidth := maxWidth - 2*gtx.Dp(20) - prefixWidth // 减去左右边距和前缀

							// 根据可用宽度和字体大小估算可显示的字符数
							// 这是一个粗略估计，假设每个字符平均宽度为字体大小的60%
							fontSize := 14
							charWidth := fontSize * 6 / 10 // 14是字体大小，取60%
							maxChars := availableWidth / charWidth

							if maxChars > 0 {
								if len(origText) > maxChars {
									origText = origText[:maxChars-3] + "..."
								}
								if len(transText) > maxChars {
									transText = transText[:maxChars-3] + "..."
								}
							}

							return layout.Flex{
								Axis:      layout.Vertical,
								Spacing:   layout.SpaceStart,
								Alignment: layout.Middle,
							}.Layout(gtx,
								// 显示原文（居中）
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									origLabel := material.Label(state.theme, 14, "原文: "+origText)
									origLabel.Alignment = text.Middle
									origLabel.Color = color.NRGBA{R: 50, G: 50, B: 50, A: 255}
									origLabel.MaxLines = 1 // 限制为一行
									return origLabel.Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
								// 显示译文（居中）
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									transLabel := material.Label(state.theme, 14, "译文: "+transText)
									transLabel.Alignment = text.Middle
									transLabel.Color = color.NRGBA{R: 0, G: 120, B: 0, A: 255}
									transLabel.MaxLines = 1 // 限制为一行
									return transLabel.Layout(gtx)
								}),
							)
						}
						return layout.Dimensions{}
					}),
				)
			}),
		)
		return // 直接返回，跳过后续按钮渲染
	}

	// 初始状态或完成状态（只有一个按钮）时使用不同的布局方式
	if (state.inputTempFile == "" && !state.translationDone) || state.translationDone {
		// 只有一个按钮的情况：垂直居中
		layout.Stack{Alignment: layout.Center}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				return layout.Dimensions{Size: gtx.Constraints.Max}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				if state.translationDone {
					// 翻译完成状态：显示状态文字和下一个按钮
					return layout.Flex{
						Axis:      layout.Vertical,
						Spacing:   layout.SpaceStart,
						Alignment: layout.Middle,
					}.Layout(gtx,
						// 状态提示
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(state.theme, 15, state.status)
							lbl.Alignment = text.Middle
							lbl.Color = color.NRGBA{R: 50, G: 50, B: 50, A: 255}
							return layout.Inset{Bottom: unit.Dp(15)}.Layout(gtx, lbl.Layout)
						}),
						// 下一个按钮
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return buttonLayout(gtx, state.theme, &state.nextBtn, "下一个", false)
						}),
					)
				} else {
					// 初始状态显示选择文件按钮
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return buttonLayout(gtx, state.theme, &state.selectBtn, "选择文件", false)
					})
				}
			}),
		)
	} else {
		// 使用Stack布局实现整体垂直居中，同时内部使用Flex进行元素排列
		layout.Stack{Alignment: layout.Center}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				return layout.Dimensions{Size: gtx.Constraints.Max}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				// 垂直排列所有元素
				return layout.Flex{
					Axis:      layout.Vertical,
					Spacing:   layout.SpaceStart,
					Alignment: layout.Middle,
				}.Layout(gtx,
					// 文件名显示
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if state.originalFilename != "" {
							lbl := material.Label(state.theme, 15, "已选择: "+state.originalFilename)
							lbl.Alignment = text.Middle
							lbl.Color = color.NRGBA{R: 50, G: 50, B: 50, A: 255}
							return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(6)}.Layout(gtx, lbl.Layout)
						}
						return layout.Dimensions{}
					}),
					// 状态提示
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if state.status != "" {
							lbl := material.Label(state.theme, 15, state.status)
							lbl.Alignment = text.Middle
							lbl.Color = color.NRGBA{R: 50, G: 50, B: 50, A: 255}
							return layout.Inset{Bottom: unit.Dp(12)}.Layout(gtx, lbl.Layout)
						}
						return layout.Dimensions{}
					}),
					// 添加额外的空白区域
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Spacer{Height: unit.Dp(15)}.Layout(gtx)
					}),
					// 重新选择按钮
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return buttonLayout(gtx, state.theme, &state.deleteBtn, "重新选择", false)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Spacer{Height: unit.Dp(6)}.Layout(gtx)
					}),
					// 翻译按钮
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return buttonLayout(gtx, state.theme, &state.translateBtn, "翻译", false)
					}),
				)
			}),
		)
	}
}

// drawBackground 绘制背景色
func drawBackground(gtx layout.Context, color color.NRGBA) {
	// 获取整个区域尺寸
	size := gtx.Constraints.Max
	// 使用完整区域大小绘制背景
	dr := image.Rectangle{Max: image.Point{X: size.X, Y: size.Y}}

	// 创建一个矩形区域
	paint.FillShape(gtx.Ops, color, clip.Rect(dr).Op())
}

// buttonLayout 创建按钮布局
func buttonLayout(gtx layout.Context, theme *material.Theme, button *widget.Clickable, label string, disabled bool) layout.Dimensions {
	// 减小按钮外边距
	margins := layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(10), Right: unit.Dp(10)}

	return margins.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// 限制按钮最小和最大宽度
		gtx.Constraints.Min.X = gtx.Dp(80)

		// 创建按钮
		btn := material.Button(theme, button, label)

		// 设置圆角
		btn.CornerRadius = unit.Dp(4)

		// 减小内边距使按钮更紧凑
		btn.Inset = layout.Inset{
			Top:    unit.Dp(4),
			Bottom: unit.Dp(4),
			Left:   unit.Dp(8),
			Right:  unit.Dp(8),
		}

		// 文字大小
		btn.TextSize = unit.Sp(14)

		// 根据禁用状态设置颜色
		if disabled {
			gtx = gtx.Disabled()
			btn.Background = color.NRGBA{R: 200, G: 200, B: 200, A: 255} // 浅灰色背景
			btn.Color = color.NRGBA{R: 150, G: 150, B: 150, A: 255}      // 灰色文字
		} else {
			btn.Background = color.NRGBA{R: 0x13, G: 0x7A, B: 0x50, A: 255} // 绿色背景 #137A50
			btn.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}         // 白色文字
		}

		return btn.Layout(gtx)
	})
}
