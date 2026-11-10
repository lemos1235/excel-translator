package main

import "github.com/egoist/mygo/ui"

// palette 收纳不在 ui.Theme 字段里、但界面各处直接使用的颜色。
// 深色沿用旧 Qt 深色截图的中性灰阶：窗口比面板暗一档，控件比面板亮；
// 浅色按同样的层次关系给出。两套配色随系统外观切换。
type palette struct {
	background ui.Color // 窗口背景
	panel      ui.Color // 左右面板
	zone       ui.Color // 拖拽区域
	zoneEdge   ui.Color // 拖拽区域描边
	input      ui.Color // 只读文件输入框

	buttonFill         ui.Color
	buttonFillDisabled ui.Color
	buttonText         ui.Color
	buttonTextDisabled ui.Color
	buttonEdge         ui.Color

	// overlay 是悬停蒙层的基色：深色界面用白，浅色用黑。
	overlay ui.Color

	stepperSurface ui.Color
	stepperHover   ui.Color
	stepperPressed ui.Color
}

var darkPalette = palette{
	background:         ui.Hex("#323232"),
	panel:              ui.Hex("#393939"),
	zone:               ui.Hex("#3f3f3f"),
	zoneEdge:           ui.Hex("#4d4d4d"),
	input:              ui.Hex("#222222"),
	buttonFill:         ui.Hex("#5a5a5a"),
	buttonFillDisabled: ui.Hex("#4a4a4a"),
	buttonText:         ui.Hex("#ffffff"),
	buttonTextDisabled: ui.Hex("#ffffff"),
	buttonEdge:         ui.Hex("#3a3a3a"),
	overlay:            ui.Hex("#ffffff"),
	stepperSurface:     ui.Hex("#2e2e2e"),
	stepperHover:       ui.Hex("#3c3c3c"),
	stepperPressed:     ui.Hex("#242424"),
}

var lightPalette = palette{
	background:         ui.Hex("#eef0f3"),
	panel:              ui.Hex("#ffffff"),
	zone:               ui.Hex("#edeff3"),
	zoneEdge:           ui.Hex("#d8dee5"),
	input:              ui.Hex("#ffffff"),
	buttonFill:         ui.Hex("#fafbfd"),
	buttonFillDisabled: ui.Hex("#f7f8fa"),
	buttonText:         ui.Hex("#1b1f24"),
	buttonTextDisabled: ui.Hex("#9aa1aa"),
	buttonEdge:         ui.Hex("#d5dbe3"),
	overlay:            ui.Hex("#000000"),
	stepperSurface:     ui.Hex("#f2f4f7"),
	stepperHover:       ui.Hex("#e9edf1"),
	stepperPressed:     ui.Hex("#dfe4ea"),
}

// paletteFor 选出与主题明暗配套的配色。
func paletteFor(t *ui.Theme) *palette {
	if t.Dark {
		return &darkPalette
	}
	return &lightPalette
}

// wash 把悬停蒙层的基色按 alpha 混合出来，用于无边框控件的悬停与按下反馈。
func (p *palette) wash(alpha float32) ui.Color {
	return ui.RGBA(p.overlay.R, p.overlay.G, p.overlay.B, alpha)
}

// appThemeFor 返回与系统外观一致的主界面主题：深色沿用旧 Qt 配色，
// 浅色是对应的日间版本。
func appThemeFor(base *ui.Theme) *ui.Theme {
	if base.Dark {
		return darkAppTheme
	}
	return lightAppTheme
}

// settingsThemeFor 沿用主界面配色，只把强调色换成参考图里的蓝色主按钮。
func settingsThemeFor(base *ui.Theme) *ui.Theme {
	if base.Dark {
		return darkSettingsTheme
	}
	return lightSettingsTheme
}

// stepperThemeFor 让数字输入框和加减按钮融入设置窗口的输入框，
// 而不是亮灰控件。
func stepperThemeFor(base *ui.Theme) *ui.Theme {
	if base.Dark {
		return darkStepperTheme
	}
	return lightStepperTheme
}

// progressThemeFor 给进度条单独一层主题：填充沿用主界面的强调色，
// 空槽与拖拽区同色，圆角取满高。Transparent 让包裹它的 Themed 列
// 不画出背景。
func progressThemeFor(base *ui.Theme) *ui.Theme {
	if base.Dark {
		return darkProgressTheme
	}
	return lightProgressTheme
}

// 主题只依赖系统外观的明暗，构建一次复用。
var (
	darkAppTheme       = buildAppTheme(ui.DarkTheme())
	lightAppTheme      = buildAppTheme(ui.LightTheme())
	darkSettingsTheme  = buildSettingsTheme(ui.DarkTheme())
	lightSettingsTheme = buildSettingsTheme(ui.LightTheme())
	darkStepperTheme   = buildStepperTheme(ui.DarkTheme())
	lightStepperTheme  = buildStepperTheme(ui.LightTheme())
	darkProgressTheme  = buildProgressTheme(ui.DarkTheme())
	lightProgressTheme = buildProgressTheme(ui.LightTheme())
)

func buildAppTheme(base *ui.Theme) *ui.Theme {
	t := *base
	// 主界面不使用蓝色强调色：主按钮是中性灰，与旧 Qt 界面一致。
	if base.Dark {
		t.Background = darkPalette.background
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
	} else {
		t.Background = lightPalette.background
		t.Surface = ui.Hex("#eef1f4")
		t.SurfaceHover = ui.Hex("#e6eaef")
		t.SurfacePressed = ui.Hex("#dce1e7")
		t.Border = ui.Hex("#d3d9e0")
		t.Text = ui.Hex("#1b1f24")
		t.TextMuted = ui.Hex("#59616b")
		t.Accent = ui.Hex("#3c424a")
		t.AccentHover = ui.Hex("#4a515a")
		t.AccentPressed = ui.Hex("#2f353c")
		t.AccentText = ui.Hex("#ffffff")
		t.Danger = ui.Hex("#d92d20")
		t.Warning = ui.Hex("#b54708")
		t.Success = ui.Hex("#067647")
		t.Selection = ui.RGBA(16, 24, 40, 0.1)
		t.Focus = ui.RGBA(16, 24, 40, 0.4)
		t.Inverse = ui.Hex("#1b1f24")
		t.InverseText = ui.Hex("#ffffff")
		t.Scrollbar = ui.RGBA(16, 24, 40, 0.28)
	}
	t.Radius = 3
	return &t
}

func buildSettingsTheme(base *ui.Theme) *ui.Theme {
	t := *buildAppTheme(base)
	if base.Dark {
		t.Accent = ui.Hex("#0067e1")
		t.AccentHover = ui.Hex("#1a7ae8")
		t.AccentPressed = ui.Hex("#0058c4")
	} else {
		t.Accent = ui.Hex("#0067e1")
		t.AccentHover = ui.Hex("#0058c4")
		t.AccentPressed = ui.Hex("#004aab")
	}
	return &t
}

func buildStepperTheme(base *ui.Theme) *ui.Theme {
	pal := paletteFor(base)
	t := *buildSettingsTheme(base)
	t.Background = pal.panel
	t.Surface = pal.stepperSurface
	t.SurfaceHover = pal.stepperHover
	t.SurfacePressed = pal.stepperPressed
	return &t
}

func buildProgressTheme(base *ui.Theme) *ui.Theme {
	t := *appThemeFor(base)
	t.Background = ui.Transparent
	// 空槽与上方的拖拽区域同色，进度条嵌在面板里不突兀。
	t.Border = paletteFor(base).zone
	// 进度条高 8；Progress 用 0.75×Spacing 作圆角，16/3 让两端刚好是半圆。
	t.Spacing = 16.0 / 3.0
	return &t
}
