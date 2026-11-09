package main

import "github.com/egoist/mygo/ui"

// 参考旧 Qt 深色界面的中性灰阶配色。窗口比面板暗一档，控件比面板亮，
// 强调色沿用截图里的浅灰主按钮，不再使用蓝色。
var (
	colorBackground = ui.Hex("#323232") // 窗口背景
	colorPanel      = ui.Hex("#393939") // 左右面板
	colorZone       = ui.Hex("#3f3f3f") // 拖拽区域
	colorInput      = ui.Hex("#222222") // 只读文件输入框
)

// appTheme 是主界面与对话框共用的主题，构建一次复用。
var appTheme = func() *ui.Theme {
	t := ui.DarkTheme()
	t.Background = colorBackground
	t.Surface = ui.Hex("#4a4a4a")
	t.SurfaceHover = ui.Hex("#575757")
	t.SurfacePressed = ui.Hex("#3a3a3a")
	t.Border = ui.Hex("#4d4d4d")
	t.Text = ui.Hex("#e8e8e8")
	t.TextMuted = ui.Hex("#a3a3a3")
	t.Accent = ui.Hex("#6a6a6a")
	t.AccentHover = ui.Hex("#7a7a7a")
	t.AccentPressed = ui.Hex("#5a5a5a")
	t.AccentText = ui.Hex("#ffffff")
	t.Danger = ui.Hex("#e57373")
	t.Warning = ui.Hex("#e0a94a")
	t.Success = ui.Hex("#7bc47f")
	t.Selection = ui.RGBA(255, 255, 255, 0.25)
	t.Focus = ui.RGBA(255, 255, 255, 0.45)
	t.Inverse = ui.Hex("#e8e8e8")
	t.InverseText = ui.Hex("#222222")
	t.Scrollbar = ui.RGBA(255, 255, 255, 0.3)
	t.Radius = 3
	return t
}()

// settingsTheme 沿用主界面配色，只把强调色换成参考图里的蓝色主按钮。
var settingsTheme = func() *ui.Theme {
	t := *appTheme
	t.Accent = ui.Hex("#0067e1")
	t.AccentHover = ui.Hex("#1a7ae8")
	t.AccentPressed = ui.Hex("#0058c4")
	return &t
}()

// stepperTheme 让数字输入框和加减按钮融入设置窗口的深色输入框，而不是亮灰控件。
var stepperTheme = func() *ui.Theme {
	t := *settingsTheme
	t.Background = colorPanel
	t.Surface = ui.Hex("#2e2e2e")
	t.SurfaceHover = ui.Hex("#3c3c3c")
	t.SurfacePressed = ui.Hex("#242424")
	return &t
}()
