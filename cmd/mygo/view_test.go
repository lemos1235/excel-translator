package main

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"exceltranslator/pkg/config"
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

func TestSettingsModalOpenAndCancel(t *testing.T) {
	a := newApp(config.DefaultConfig())
	originalKey := a.cfg.LLM.APIKey

	tt := ui.NewTester(a.view, 800, 400)
	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click settings: %v", err)
	}
	if !tt.HasText("偏好设置") {
		t.Fatalf("settings modal did not open: %q", tt.Texts())
	}

	a.settingsDraft.LLM.APIKey = "changed"
	if err := tt.Click("取消"); err != nil {
		t.Fatalf("click cancel: %v", err)
	}
	if tt.HasText("偏好设置") {
		t.Error("settings modal still open after cancel")
	}
	if a.cfg.LLM.APIKey != originalKey {
		t.Errorf("cancel changed config: %q", a.cfg.LLM.APIKey)
	}
}

func TestSettingsSaveKeepsDraft(t *testing.T) {
	a := newApp(config.DefaultConfig())

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

	tt := ui.NewTester(a.view, 800, 400)
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
	if a.settingsOpen {
		t.Error("settings modal still open after save")
	}
	if a.cfg.LLM.APIKey != "sk-test" {
		t.Errorf("in-memory config not updated: %q", a.cfg.LLM.APIKey)
	}
}

func TestAcceptFileRejectsUnsupportedExtension(t *testing.T) {
	a := newApp(config.DefaultConfig())
	a.acceptFile(filepath.Join(t.TempDir(), "notes.pdf"))

	if a.inputFile != "" {
		t.Errorf("unsupported file accepted: %q", a.inputFile)
	}
	if a.toast == "" {
		t.Error("expected an error toast for unsupported extension")
	}
}

func TestStartTranslationWithoutFile(t *testing.T) {
	a := newApp(config.DefaultConfig())
	a.startTranslation()

	if a.toast == "" {
		t.Error("expected a toast when no file is selected")
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
