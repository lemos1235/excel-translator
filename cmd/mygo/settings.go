package main

import (
	"github.com/egoist/mygo/ui"

	"exceltranslator/pkg/config"
)

// openSettings 从当前配置拷贝一份草稿，取消时直接丢弃。
func (a *app) openSettings() {
	a.settingsDraft = *a.cfg
	a.settingsConcurrency = float64(a.settingsDraft.Translation.MaxConcurrentRequests)
	if a.settingsConcurrency < 1 {
		a.settingsConcurrency = 1
	}
	a.settingsError = ""
	a.settingsOpen = true
	a.invalidate()
}

func (a *app) settingsModal(c *ui.Context, t *ui.Theme) {
	ui.Modal(c, &a.settingsOpen, func() {
		ui.Column(c).Width(520).Gap(14).Padding(20).Children(func() {
			ui.Text(c, "偏好设置").FontSize(18).Bold()

			ui.Form(c, func() {
				ui.Field(c, "API Key", func() {
					ui.TextInput(c, &a.settingsDraft.LLM.APIKey).Password()
				})
				ui.Field(c, "API URL", func() {
					ui.TextInput(c, &a.settingsDraft.LLM.BaseURL)
				})
				ui.Field(c, "模型", func() {
					ui.TextInput(c, &a.settingsDraft.LLM.Model)
				})
				ui.Field(c, "翻译提示词", func() {
					ui.TextArea(c, &a.settingsDraft.LLM.Prompt).Lines(4, 8)
				})
				ui.Field(c, "最大并发请求数", func() {
					ui.NumberInput(c, &a.settingsConcurrency, 1, 20, 1)
				})
				ui.Field(c, "仅翻译 CJK 文本", func() {
					ui.Checkbox(c, &a.settingsDraft.Translation.CJKOnly, "")
				})
			})

			if a.settingsError != "" {
				ui.Text(c, a.settingsError).FontSize(12).TextColor(t.Danger).Wrap()
			}

			ui.Row(c).Justify(ui.End).Gap(8).Children(func() {
				if ui.Button(c, "取消").Clicked() {
					a.settingsOpen = false
					a.settingsError = ""
				}
				if ui.PrimaryButton(c, "保存").Clicked() {
					a.saveSettings()
				}
			})
		})
	})
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
		a.invalidate()
		return
	}

	a.settingsOpen = false
	a.settingsError = ""
	a.toast = "配置已保存"
	a.invalidate()
}
