package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"exceltranslator/pkg/config"
	"exceltranslator/pkg/runner"

	"github.com/progrium/darwinkit/dispatch"
	"github.com/progrium/darwinkit/helper/action"
	"github.com/progrium/darwinkit/macos"
	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/progrium/darwinkit/macos/foundation"
	"github.com/progrium/darwinkit/macos/uti"
	"github.com/progrium/darwinkit/objc"
)

func main() {
	macos.RunApp(func(app appkit.Application, delegate *appkit.ApplicationDelegate) {
		app.SetActivationPolicy(appkit.ApplicationActivationPolicyRegular)
		app.ActivateIgnoringOtherApps(true)

		// --- State ---
		var (
			ctx    context.Context
			cancel context.CancelFunc
		)

		// --- UI Components ---
		window := appkit.NewWindowWithContentRectStyleMaskBackingDefer(
			foundation.Rect{Size: foundation.Size{Width: 600, Height: 500}},
			appkit.WindowStyleMaskClosable|appkit.WindowStyleMaskTitled|appkit.WindowStyleMaskResizable,
			appkit.BackingStoreBuffered, false)
		window.SetTitle("Excel Translator (Native)")

		// Input Section
		inputField := appkit.NewTextField()
		inputField.SetPlaceholderString("Choose an Excel/Word file...")
		inputField.SetEditable(false)

		browseBtn := appkit.NewButtonWithTitle("Browse...")

		// Settings Section
		apiKeyField := appkit.NewTextField()
		apiKeyField.SetPlaceholderString("API Key")

		apiURLField := appkit.NewTextField()
		apiURLField.SetPlaceholderString("API URL")

		modelField := appkit.NewTextField()
		modelField.SetPlaceholderString("Model")

		promptField := appkit.NewTextField()
		promptField.SetPlaceholderString("Translation Prompt")

		cjkCheckbox := appkit.NewCheckBox("Only Translate CJK")
		cjkCheckbox.SetState(appkit.ControlStateValueOn)

		// Progress
		progressIndicator := appkit.NewProgressIndicator()
		progressIndicator.SetIndeterminate(false)
		progressIndicator.SetMinValue(0)
		progressIndicator.SetMaxValue(100)
		progressIndicator.SetDoubleValue(0)

		// Log
		scrollableTextView := appkit.NewScrollableTextView()
		logView := scrollableTextView.ContentTextView()
		logView.SetEditable(false)

		logScrollView := scrollableTextView.ScrollView
		logScrollView.SetHasVerticalScroller(true)

		// Buttons
		startBtn := appkit.NewButtonWithTitle("Start Translation")
		stopBtn := appkit.NewButtonWithTitle("Stop")
		stopBtn.SetEnabled(false)

		// --- Layout (Simple Stack) ---
		contentView := appkit.NewStackView()
		contentView.SetOrientation(appkit.UserInterfaceLayoutOrientationVertical)
		contentView.SetEdgeInsets(foundation.EdgeInsets{Top: 20, Left: 20, Bottom: 20, Right: 20})
		contentView.SetSpacing(10)
		contentView.SetAlignment(appkit.LayoutAttributeCenterX)

		// Helper to add rows
		addRow := func(label string, view appkit.IView) {
			row := appkit.NewStackView()
			row.SetOrientation(appkit.UserInterfaceLayoutOrientationHorizontal)
			row.SetSpacing(10)
			if label != "" {
				lbl := appkit.NewLabel(label)
				lbl.SetAlignment(appkit.TextAlignmentRight)
				row.AddViewInGravity(lbl, appkit.StackViewGravityLeading)
			}
			row.AddViewInGravity(view, appkit.StackViewGravityTrailing)
			contentView.AddViewInGravity(row, appkit.StackViewGravityTop)
		}

		addRow("File:", inputField)
		contentView.AddViewInGravity(browseBtn, appkit.StackViewGravityTop)
		addRow("API Key:", apiKeyField)
		addRow("API URL:", apiURLField)
		addRow("Model:", modelField)
		addRow("Prompt:", promptField)
		contentView.AddViewInGravity(cjkCheckbox, appkit.StackViewGravityTop)
		contentView.AddViewInGravity(progressIndicator, appkit.StackViewGravityTop)

		btnRow := appkit.NewStackView()
		btnRow.SetOrientation(appkit.UserInterfaceLayoutOrientationHorizontal)
		btnRow.SetSpacing(10)
		btnRow.AddViewInGravity(startBtn, appkit.StackViewGravityCenter)
		btnRow.AddViewInGravity(stopBtn, appkit.StackViewGravityCenter)
		contentView.AddViewInGravity(btnRow, appkit.StackViewGravityTop)

		contentView.AddViewInGravity(logScrollView, appkit.StackViewGravityTop)

		window.SetContentView(contentView)
		window.MakeKeyAndOrderFront(nil)
		window.Center()

		// --- Logic ---

		var logs []string
		addLog := func(msg string) {
			dispatch.MainQueue().DispatchAsync(func() {
				ts := time.Now().Format("15:04:05")
				text := fmt.Sprintf("[%s] %s", ts, msg)
				logs = append(logs, text)
				fullText := strings.Join(logs, "\n")
				logView.SetString(fullText)

				// Scroll to end
				rangeEnd := foundation.Range{Location: uint64(len(fullText)), Length: 0}
				logView.ScrollRangeToVisible(rangeEnd)
			})
		}

		// Load Initial Config
		cfg, err := config.Load()
		if err == nil {
			apiKeyField.SetStringValue(cfg.LLM.APIKey)
			apiURLField.SetStringValue(cfg.LLM.BaseURL)
			modelField.SetStringValue(cfg.LLM.Model)
			promptField.SetStringValue(cfg.LLM.Prompt)
			if cfg.Extractor.CJKOnly {
				cjkCheckbox.SetState(appkit.ControlStateValueOn)
			} else {
				cjkCheckbox.SetState(appkit.ControlStateValueOff)
			}
		}

		action.Set(browseBtn, func(sender objc.Object) {
			panel := appkit.OpenPanel_OpenPanel()
			panel.SetCanChooseFiles(true)
			panel.SetCanChooseDirectories(false)
			panel.SetAllowsMultipleSelection(false)

			xlsxType := uti.Type_TypeWithFilenameExtension("xlsx")
			docxType := uti.Type_TypeWithFilenameExtension("docx")
			panel.SetAllowedContentTypes([]uti.IType{xlsxType, docxType})

			if panel.RunModal() == appkit.ModalResponseOK {
				urls := panel.URLs()
				if len(urls) > 0 {
					inputField.SetStringValue(urls[0].Path())
				}
			}
		})

		action.Set(startBtn, func(sender objc.Object) {
			inputFile := inputField.StringValue()
			if inputFile == "" {
				return
			}

			// Save config first
			newCfg := &config.AppConfig{
				LLM: config.LLMConfig{
					APIKey:  apiKeyField.StringValue(),
					BaseURL: apiURLField.StringValue(),
					Model:   modelField.StringValue(),
					Prompt:  promptField.StringValue(),
				},
				Extractor: config.ExtractorConfig{
					CJKOnly: cjkCheckbox.State() == appkit.ControlStateValueOn,
				},
			}
			config.Save(newCfg)

			// Setup output file
			base := filepath.Base(inputFile)
			ext := filepath.Ext(base)
			name := strings.TrimSuffix(base, ext)
			outputFile := filepath.Join(os.TempDir(), fmt.Sprintf("%s_translated_%d%s", name, time.Now().Unix(), ext))

			ctx, cancel = context.WithCancel(context.Background())

			// UI Updates
			startBtn.SetEnabled(false)
			stopBtn.SetEnabled(true)
			progressIndicator.SetDoubleValue(0)
			logs = nil
			logView.SetString("")

			addLog("Starting translation...")
			addLog(fmt.Sprintf("Input: %s", inputFile))

			go func() {
				err := runner.RunTranslation(ctx, inputFile, outputFile, runner.TranslationCallbacks{
					OnProgress: func(phase string, done, total int) {
						dispatch.MainQueue().DispatchAsync(func() {
							val := float64(done) / float64(total) * 100
							progressIndicator.SetDoubleValue(val)
						})
					},
					OnTranslated: func(original, translated string) {
						addLog(fmt.Sprintf("%s -> %s", original, translated))
					},
					OnError: func(stage string, err error) {
						addLog(fmt.Sprintf("Error in %s: %v", stage, err))
					},
					OnComplete: func(err error) {
						dispatch.MainQueue().DispatchAsync(func() {
							startBtn.SetEnabled(true)
							stopBtn.SetEnabled(false)

							if err != nil {
								addLog(fmt.Sprintf("Translation failed: %v", err))
								return
							}

							addLog("Translation completed successfully!")
							progressIndicator.SetDoubleValue(100)

							// Save Panel
							savePanel := appkit.SavePanel_SavePanel()
							savePanel.SetTitle("Save Translated File")
							savePanel.SetCanCreateDirectories(true)
							savePanel.SetNameFieldStringValue(name + "_translated" + ext)

							if savePanel.RunModal() == appkit.ModalResponseOK {
								dst := savePanel.URL().Path()
								// Simple file copy
								input, _ := os.ReadFile(outputFile)
								os.WriteFile(dst, input, 0644)
								addLog(fmt.Sprintf("Saved to: %s", dst))
							}
							os.Remove(outputFile)
						})
					},
				})
				if err != nil {
					fmt.Printf("Runner error: %v\n", err)
				}
			}()
		})

		action.Set(stopBtn, func(sender objc.Object) {
			if cancel != nil {
				cancel()
				addLog("Cancelling...")
			}
		})

		delegate.SetApplicationShouldTerminateAfterLastWindowClosed(func(appkit.Application) bool {
			return true
		})
	})
}
