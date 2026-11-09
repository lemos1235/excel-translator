package main

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// view 构建主界面。它始终在主线程执行，只读取状态快照，不阻塞。
func (a *app) view(c *ui.Context) {
	t := c.Theme()

	translating, progress, _, logs := a.snapshot()

	if a.toast != "" {
		c.Toast(a.toast)
		a.toast = ""
	}

	width, _ := c.Size()
	ui.Column(c).Fill().Padding(14, 16).Gap(12).Children(func() {
		// 右上角设置入口
		ui.Row(c).Justify(ui.End).Children(func() {
			if ui.Button(c, "设置").Label("设置").Tooltip("偏好设置").Clicked() {
				a.openSettings()
			}
		})

		if width >= 640 {
			ui.Row(c).Grow(1).Gap(20).AlignItems(ui.Stretch).Children(func() {
				a.leftPanel(c, t, translating, progress)
				a.logPanel(c, t, logs)
			})
		} else {
			ui.Row(c).Grow(1).Justify(ui.Center).Children(func() {
				a.leftPanel(c, t, translating, progress)
			})
		}
	})

	a.settingsModal(c, t)
	a.aboutModal(c)
}

// leftPanel 构建左侧文件选择、进度和开始/停止按钮。
func (a *app) leftPanel(c *ui.Context, t *ui.Theme, translating bool, progress float64) {
	ui.Column(c).Width(350).Padding(15).Gap(20).Radius(10).
		Border(1, t.Border).Children(func() {
		zone := ui.Column(c).MinHeight(220).Padding(15).Gap(10).Radius(10).
			Border(1, t.Border)

		zone.Children(func() {
			ui.Text(c, "拖拽文件到此区域或点击浏览文件").
				TextColor(t.TextMuted).TextAlign(ui.Center)
			ui.TextInput(c, &a.inputFile).ReadOnly(true).
				Placeholder("选择要翻译的 Excel/Word/PPT 文件…")
			if ui.Button(c, "浏览文件…").Clicked() {
				a.chooseInputFile()
			}
		})

		if zone.FileDragOver() {
			zone.Border(1, t.Accent)
		}
		if files := zone.DroppedFiles(); len(files) > 0 {
			a.acceptFile(files[0])
		}

		ui.Progress(c, progress).Height(8)

		ui.Row(c).Justify(ui.Center).Gap(20).Children(func() {
			if ui.PrimaryButton(c, "开始翻译").Width(88).
				Disabled(translating).Clicked() {
				a.startTranslation()
			}
			if ui.Button(c, "停止翻译").Width(88).
				Disabled(!translating).Clicked() {
				a.stopTranslation()
			}
		})

		// 与旧版一致：内容靠上，剩余空间留在底部。
		ui.Spacer(c)
	})
}

// logPanel 构建右侧日志区，用户停留在底部时自动跟随新日志。
func (a *app) logPanel(c *ui.Context, t *ui.Theme, logs []string) {
	ui.Column(c).Grow(1).Gap(6).Children(func() {
		ui.Text(c, "日志").Bold()

		ui.Column(c).Grow(1).Padding(8).Radius(10).
			Border(1, t.Border).Children(func() {
			if a.logScroll.Y >= a.logScroll.MaxY {
				a.logScroll.Y = math.MaxFloat32
			}

			ui.Scroll(c).TrackScroll(&a.logScroll).Grow(1).
				KeepScrollbars(true).Children(func() {
				for _, line := range logs {
					ui.Text(c, line).FontSize(12).Font("monospace").Wrap()
				}
			})
		})
	})
}
