package main

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// view 构建主界面。它始终在主线程执行，只读取状态快照，不阻塞。
func (a *app) view(c *ui.Context) {
	// 先取框架按系统外观给出的默认主题，再换成同明暗的自定义配色。
	t := appThemeFor(c.Theme())
	c.SetTheme(t)
	pal := paletteFor(t)

	translating, progress, _, logs := a.snapshot()

	width, _ := c.Size()
	ui.Column(c).Fill().Padding(6, 16).Gap(8).Children(func() {
		// 右上角设置入口：无边框的齿轮图标，悬停时淡淡高亮。
		ui.Row(c).Justify(ui.End).Children(func() {
			settings := ui.ButtonBase(c).Size(20, 20).Radius(t.Radius).
				Label("设置").Tooltip("偏好设置").
				Children(func() {
					ui.Icon(c, gearIcon).Size(12, 12).TextColor(t.TextMuted)
				})
			switch {
			case settings.Pressed():
				settings.Background(pal.wash(0.14))
			case settings.Hovered():
				settings.Background(pal.wash(0.08))
			}
			if settings.Clicked() {
				a.openSettings()
			}
		})

		if width >= 640 {
			ui.Row(c).Grow(1).Gap(20).AlignItems(ui.Stretch).
				Margin(0, 0, 8, 0).Children(func() {
				a.leftPanel(c, t, translating, progress, false)
				a.logPanel(c, t, logs)
			})
		} else {
			ui.Row(c).Grow(1).AlignItems(ui.Stretch).
				Margin(0, 0, 8, 0).Children(func() {
				a.leftPanel(c, t, translating, progress, true)
			})
		}
	})

	a.aboutModal(c)
}

// leftPanel 构建左侧文件选择、进度和开始/停止按钮。grow 为真时
// 面板填满窄窗口的可用宽度，避免两侧留下大片空白。
func (a *app) leftPanel(c *ui.Context, t *ui.Theme, translating bool, progress float64, grow bool) {
	pal := paletteFor(t)
	panel := ui.Column(c).Padding(15).Gap(20).Radius(10).
		Border(1, t.Border).Background(pal.panel)
	if grow {
		panel.Grow(1)
	} else {
		panel.Width(350)
	}
	panel.Children(func() {
		zone := ui.Column(c).MinHeight(220).Padding(15).Gap(10).Radius(10).
			Border(1, pal.zoneEdge).Background(pal.zone)

		zone.Children(func() {
			ui.Text(c, "拖拽文件到此区域或点击浏览文件").
				TextColor(t.TextMuted).TextAlign(ui.Center)
			ui.TextInput(c, &a.inputFile).ReadOnly(true).
				Background(pal.input).
				Placeholder("选择要翻译的 Excel/Word/PPT 文件…")
			if appButton(c, "浏览文件…", false).Clicked() {
				a.chooseInputFile()
			}
		})

		if zone.FileDragOver() {
			zone.Border(1, t.Accent)
		}
		if files := zone.DroppedFiles(); len(files) > 0 {
			a.acceptFile(files[0])
		}

		// 进度条用蓝色强调色填充，主界面其余控件保持灰色。
		ui.Themed(c, progressThemeFor(t)).Children(func() {
			ui.Progress(c, progress).Height(8)
		})

		ui.Row(c).Justify(ui.Center).Gap(20).Children(func() {
			if appPrimaryButton(c, "开始翻译", translating).Width(80).Clicked() {
				a.startTranslation()
			}
			if appButton(c, "停止翻译", !translating).Width(80).Clicked() {
				a.stopTranslation()
			}
		})

		// 与旧版一致：内容靠上，剩余空间留在底部。
		ui.Spacer(c)
	})
}

// logPanel 构建右侧日志区，用户停留在底部时自动跟随新日志。
func (a *app) logPanel(c *ui.Context, t *ui.Theme, logs []string) {
	pal := paletteFor(t)
	// 右侧内边距比其他三边小，让滚动条贴近面板边缘；滚动内容自身
	// 仍保留左右留白，避免文字与滚动条重叠。
	ui.Column(c).Grow(1).Gap(6).Padding(10, 4, 10, 10).Radius(10).
		Border(1, t.Border).Background(pal.panel).Children(func() {
		ui.Text(c, "日志").Bold()

		if a.logScroll.Y >= a.logScroll.MaxY {
			a.logScroll.Y = math.MaxFloat32
		}

		ui.Scroll(c).TrackScroll(&a.logScroll).Grow(1).Padding(2, 10, 2, 0).
			KeepScrollbars(true).Children(func() {
			for _, line := range logs {
				ui.Text(c, line).FontSize(12).Font("monospace").Wrap()
			}
		})
	})
}
