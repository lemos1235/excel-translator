package main

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"ooxmltranslator/pkg/config"
)

func TestIsOfficeFile(t *testing.T) {
	cases := map[string]bool{
		"a.xlsx":     true,
		"a.DOCX":     true,
		"dir/b.pptx": true,
		"a.pdf":      false,
		"a":          false,
		"a.xlsx.txt": false,
	}
	for path, want := range cases {
		if got := isOfficeFile(path); got != want {
			t.Errorf("isOfficeFile(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestViewRendersMainControls(t *testing.T) {
	a := newApp(config.DefaultConfig())
	tt := ui.NewTester(a.view, 800, 400)

	for _, text := range []string{"开始翻译", "停止翻译", "日志", "浏览文件…"} {
		if !tt.HasText(text) {
			t.Errorf("missing %q in %q", text, tt.Texts())
		}
	}
}

func TestNarrowWindowHidesLogPanel(t *testing.T) {
	a := newApp(config.DefaultConfig())
	tt := ui.NewTester(a.view, 480, 400)

	if tt.HasText("日志") {
		t.Errorf("log panel should be hidden in a narrow window: %q", tt.Texts())
	}
	if !tt.HasText("开始翻译") {
		t.Errorf("main controls missing: %q", tt.Texts())
	}
}

func TestSettingsOpenAndCancel(t *testing.T) {
	a := newApp(config.DefaultConfig())
	opened := false
	a.presentSettings = func() { opened = true }
	originalKey := a.cfg.LLM.APIKey

	tt := ui.NewTester(a.view, 800, 600)
	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click settings: %v", err)
	}
	if !opened {
		t.Fatal("settings window was not opened")
	}

	vt := ui.NewTester(a.settingsView, 540, 540)
	if !vt.HasText("LLM 配置") || !vt.HasText("客户端配置") {
		t.Fatalf("settings view missing sections: %q", vt.Texts())
	}

	a.settingsDraft.LLM.APIKey = "changed"
	if err := vt.Click("取消"); err != nil {
		t.Fatalf("click cancel: %v", err)
	}
	if a.cfg.LLM.APIKey != originalKey {
		t.Errorf("cancel changed config: %q", a.cfg.LLM.APIKey)
	}
}

func TestSettingsSaveKeepsDraft(t *testing.T) {
	a := newApp(config.DefaultConfig())
	a.presentSettings = func() {}

	var saved *config.AppConfig
	a.saveConfigFn = func(cfg *config.AppConfig) error {
		copy := *cfg
		saved = &copy
		return nil
	}

	a.openSettings()
	a.settingsDraft.LLM.APIKey = "sk-test"
	a.settingsDraft.LLM.Model = "test-model"
	a.settingsConcurrency = 8

	tt := ui.NewTester(a.settingsView, 540, 540)
	if err := tt.Click("保存"); err != nil {
		t.Fatalf("click save: %v", err)
	}

	if saved == nil {
		t.Fatal("config was not saved")
	}
	if saved.LLM.APIKey != "sk-test" || saved.LLM.Model != "test-model" {
		t.Errorf("saved config = %+v", saved.LLM)
	}
	if saved.Translation.MaxConcurrentRequests != 8 {
		t.Errorf("max concurrent = %d, want 8", saved.Translation.MaxConcurrentRequests)
	}
	if a.cfg.LLM.APIKey != "sk-test" {
		t.Errorf("in-memory config not updated: %q", a.cfg.LLM.APIKey)
	}
}

func TestAcceptFileRejectsUnsupportedExtension(t *testing.T) {
	a := newApp(config.DefaultConfig())
	var alerts []string
	a.alert = func(title, message string) { alerts = append(alerts, title+": "+message) }

	a.acceptFile(filepath.Join(t.TempDir(), "notes.pdf"))

	if a.inputFile != "" {
		t.Errorf("unsupported file accepted: %q", a.inputFile)
	}
	if len(alerts) == 0 {
		t.Error("expected an alert for unsupported extension")
	}
}

func TestStartTranslationWithoutFile(t *testing.T) {
	a := newApp(config.DefaultConfig())
	var alerts []string
	a.alert = func(title, message string) { alerts = append(alerts, title+": "+message) }

	a.startTranslation()

	if len(alerts) == 0 {
		t.Error("expected an alert when no file is selected")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.translating {
		t.Error("translation started without a file")
	}
}

func TestStopTranslationWhenIdle(t *testing.T) {
	a := newApp(config.DefaultConfig())
	a.stopTranslation() // 不应 panic
}
