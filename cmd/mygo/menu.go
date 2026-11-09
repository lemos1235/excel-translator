package main

import (
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// installMenu 设置应用菜单。macOS 使用应用菜单，其他平台使用“文件”菜单。
func (a *app) installMenu() {
	openSettings := func(*mygo.MenuItem, *mygo.Window) {
		a.openSettings()
	}
	showAbout := func(*mygo.MenuItem, *mygo.Window) {
		a.aboutOpen = true
		a.invalidate()
	}

	var items []*mygo.MenuItem
	if runtime.GOOS == "darwin" {
		items = []*mygo.MenuItem{
			{Label: appDisplayName, Submenu: []*mygo.MenuItem{
				{Label: "关于 " + appDisplayName, Click: showAbout},
				mygo.Separator(),
				{Label: "设置…", Accelerator: "CmdOrCtrl+,", Click: openSettings},
				mygo.Separator(),
				{Role: mygo.RoleServices},
				mygo.Separator(),
				{Role: mygo.RoleHide},
				{Role: mygo.RoleHideOthers},
				{Role: mygo.RoleUnhide},
				mygo.Separator(),
				{Role: mygo.RoleQuit},
			}},
			{Role: mygo.RoleEditMenu},
			{Role: mygo.RoleWindowMenu},
		}
	} else {
		items = []*mygo.MenuItem{
			{Label: "文件", Submenu: []*mygo.MenuItem{
				{Label: "设置…", Accelerator: "CmdOrCtrl+,", Click: openSettings},
				mygo.Separator(),
				{Label: "关于 " + appDisplayName, Click: showAbout},
				mygo.Separator(),
				{Role: mygo.RoleQuit},
			}},
			{Role: mygo.RoleEditMenu},
		}
	}

	mygo.App.SetMenu(mygo.NewMenu(items))
}

func (a *app) aboutModal(c *ui.Context) {
	t := c.Theme()
	ui.Modal(c, &a.aboutOpen, func() {
		ui.Column(c).Width(320).Gap(10).Padding(20).Children(func() {
			ui.Text(c, appDisplayName).FontSize(20).Bold()
			ui.Text(c, "版本 "+displayVersion()).TextColor(t.TextMuted)
			ui.Text(c, "使用 MyGo 构建").FontSize(12).TextColor(t.TextMuted)
			ui.Row(c).Justify(ui.End).Children(func() {
				if ui.PrimaryButton(c, "好").Clicked() {
					a.aboutOpen = false
				}
			})
		})
	})
}
