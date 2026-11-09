package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"ooxmltranslator/pkg/config"
	"ooxmltranslator/pkg/version"
)

// appDisplayName 应用在窗口标题、菜单与关于对话框中显示的名称。
const appDisplayName = "OOXML Translator"

// displayVersion 返回展示给用户的版本号。
// 构建脚本会通过 ldflags 注入 version.Full；开发环境下回退到
// mygo.json 里的版本号，再不行就显示 dev。
func displayVersion() string {
	if version.Full != "" && version.Full != "dev" {
		return version.Full
	}
	if v := mygo.App.Version(); v != "" {
		return v
	}
	return version.Full
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("加载配置失败，使用默认配置: %v", err)
		cfg = config.DefaultConfig()
	}

	a := newApp(cfg)

	mygo.App.SetName(appDisplayName)
	mygo.App.SetVersion(displayVersion())
	a.installMenu()

	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:     appDisplayName,
			Width:     880,
			Height:    480,
			MinWidth:  400,
			MinHeight: 480,
			// 新 UI 的布局与旧 Qt 版不同，换一个 key 让首次启动使用
			// 上面的默认尺寸，而不是继承旧的窗口状态。
			StateKey: "main-v2",
			Content:  ui.View(a.view),
		})
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
