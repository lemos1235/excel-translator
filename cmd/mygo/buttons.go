package main

import "github.com/egoist/mygo/ui"

// 按钮的颜色：灰色填充配纯白文字，深灰细边给边缘一个清晰轮廓。
var (
	buttonFill         = ui.Hex("#5a5a5a")
	buttonFillDisabled = ui.Hex("#4a4a4a")
	buttonText         = ui.Hex("#ffffff")
	buttonEdge         = ui.Hex("#3a3a3a")
)

// appButton 画出参考图里的普通按钮。
func appButton(c *ui.Context, label string, disabled bool) ui.Element {
	return styleAppButton(ui.Button(c, label), buttonFill, disabled)
}

// appPrimaryButton 画出参考图里的主按钮，填充取当前主题的强调色。
func appPrimaryButton(c *ui.Context, label string, disabled bool) ui.Element {
	return styleAppButton(ui.PrimaryButton(c, label), c.Theme().Accent, disabled)
}

// styleAppButton 给按钮套上灰色填充。
func styleAppButton(b ui.Element, fill ui.Color, disabled bool) ui.Element {
	if disabled {
		fill = buttonFillDisabled
	}
	return b.
		Disabled(disabled).
		Background(fill).
		TextColor(buttonText).
		Padding(2.5, 10).
		Radius(6).
		Border(1, buttonEdge)
}
