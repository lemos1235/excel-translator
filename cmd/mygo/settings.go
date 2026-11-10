package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"ooxmltranslator/pkg/config"
)

// openSettings 从当前配置拷贝一份草稿并打开偏好设置窗口，取消时直接丢弃。
func (a *app) openSettings() {
	a.settingsDraft = *a.cfg
	a.settingsConcurrency = float64(a.settingsDraft.Translation.MaxConcurrentRequests)
	if a.settingsConcurrency < 1 {
		a.settingsConcurrency = 1
	}
	a.settingsError = ""

	if a.presentSettings != nil {
		a.presentSettings()
		return
	}
	a.showSettingsWindow()
}

// showSettingsWindow 打开偏好设置窗口，已在显示时只聚焦。
func (a *app) showSettingsWindow() {
	if a.settingsWin != nil && !a.settingsWin.IsDestroyed() {
		a.settingsWin.Show()
		a.settingsWin.Focus()
		return
	}

	win := mygo.NewWindow(mygo.WindowOptions{
		Title:     "偏好设置",
		Width:     540,
		Height:    520,
		MinWidth:  480,
		MinHeight: 500,
		Content:   ui.View(a.settingsView),
	})
	a.settingsWin = win
	win.OnClosed(func() {
		if a.settingsWin == win {
			a.settingsWin = nil
		}
		a.settingsError = ""
	})
}

// settingsView 构建偏好设置窗口的内容：两个分组框，底部是保存与取消。
func (a *app) settingsView(c *ui.Context) {
	t := settingsThemeFor(c.Theme())
	c.SetTheme(t)
	pal := paletteFor(t)

	ui.Column(c).Fill().Padding(20).Gap(14).Children(func() {
		ui.Form(c, func() {
			a.settingsSection(c, "LLM 配置", func() {
				ui.Field(c, "API Key:", func() {
					ui.TextInput(c, &a.settingsDraft.LLM.APIKey).Password().
						Background(pal.input)
				})
				ui.Field(c, "API URL:", func() {
					ui.TextInput(c, &a.settingsDraft.LLM.BaseURL).
						Background(pal.input)
				})
				ui.Field(c, "模型:", func() {
					ui.TextInput(c, &a.settingsDraft.LLM.Model).
						Background(pal.input)
				})
			})

			a.settingsSection(c, "客户端配置", func() {
				ui.Field(c, "仅翻译CJK文本:", func() {
					ui.Checkbox(c, &a.settingsDraft.Translation.CJKOnly, "")
				})
				ui.Field(c, "翻译提示词:", func() {
					ui.TextArea(c, &a.settingsDraft.LLM.Prompt).Lines(4, 8).
						Background(pal.input)
				})
				ui.Field(c, "最大并发请求数:", func() {
					ui.Themed(c, stepperThemeFor(t)).Children(func() {
						ui.NumberInput(c, &a.settingsConcurrency, 1, 20, 1)
					})
				})
			})
		}).Gap(20)

		ui.Spacer(c)

		if a.settingsError != "" {
			ui.Text(c, a.settingsError).FontSize(12).TextColor(t.Danger).Wrap()
		}

		ui.Row(c).Justify(ui.Center).Gap(20).Children(func() {
			if ui.PrimaryButton(c, "保存").Width(80).Padding(2.5, 10).Radius(6).Clicked() {
				a.saveSettings()
			}
			if ui.Button(c, "取消").Width(80).Padding(2.5, 10).Radius(6).Clicked() {
				a.cancelSettings()
			}
		})
	})
}

// settingsSection 画出参考图里的分组框：标题压在框线之上，字段由外层 Form 统一对齐。
func (a *app) settingsSection(c *ui.Context, legend string, fn func()) {
	t := c.Theme()
	pal := paletteFor(t)
	ui.Column(c).Gap(0).Children(func() {
		ui.Text(c, legend).FontSize(13).Bold().
			Background(t.Background).Padding(0, 0, 4)
		ui.Column(c).Padding(14).Gap(8).Radius(6).
			Border(1, t.Border).Background(pal.panel).Children(fn)
	})
}

func (a *app) cancelSettings() {
	a.settingsError = ""
	if a.settingsWin != nil && !a.settingsWin.IsDestroyed() {
		a.settingsWin.Close()
	}
}

func (a *app) saveSettings() {
	a.settingsDraft.Translation.MaxConcurrentRequests = int(a.settingsConcurrency)
	if a.settingsDraft.Translation.MaxConcurrentRequests < 1 {
		a.settingsDraft.Translation.MaxConcurrentRequests = 1
	}

	previous := *a.cfg
	*a.cfg = a.settingsDraft

	save := a.saveConfigFn
	if save == nil {
		save = config.Save
	}
	if err := save(a.cfg); err != nil {
		*a.cfg = previous
		a.settingsError = "保存配置失败: " + err.Error()
		a.invalidateSettings()
		return
	}

	a.settingsError = ""
	if a.settingsWin != nil && !a.settingsWin.IsDestroyed() {
		a.settingsWin.Close()
	}
}

// invalidateSettings 刷新偏好设置窗口。
func (a *app) invalidateSettings() {
	if a.settingsWin != nil && !a.settingsWin.IsDestroyed() {
		a.settingsWin.Invalidate()
	}
}
