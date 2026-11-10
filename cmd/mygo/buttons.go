package main

import "github.com/egoist/mygo/ui"

// appButton 画出参考图里的普通按钮，填充取当前主题的灰阶。
func appButton(c *ui.Context, label string, disabled bool) ui.Element {
	pal := paletteFor(c.Theme())
	return styleAppButton(ui.Button(c, label), pal, pal.buttonFill, pal.buttonEdge, pal.buttonText, disabled)
}

// appPrimaryButton 画出参考图里的主按钮，填充取当前主题的强调色。
func appPrimaryButton(c *ui.Context, label string, disabled bool) ui.Element {
	t := c.Theme()
	return styleAppButton(ui.PrimaryButton(c, label), paletteFor(t), t.Accent, t.Accent, t.AccentText, disabled)
}

// styleAppButton 给按钮套上当前主题的填充、描边与文字颜色。
func styleAppButton(b ui.Element, pal *palette, fill, edge, text ui.Color, disabled bool) ui.Element {
	if disabled {
		fill = pal.buttonFillDisabled
		text = pal.buttonTextDisabled
	}
	return b.
		Disabled(disabled).
		Background(fill).
		TextColor(text).
		Padding(2.5, 10).
		Radius(6).
		Border(1, edge)
}
