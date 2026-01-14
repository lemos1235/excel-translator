## 1. Setup & Refactoring
- [ ] 1.1 Update `go.mod` to require and replace `github.com/progrium/darwinkit` with `./darwinkit`.
- [ ] 1.2 Run `go mod tidy`.
- [ ] 1.3 Create new package `pkg/runner`.
- [ ] 1.4 Move `RunTranslation` and `TranslationCallbacks` from `main.go` to `pkg/runner/runner.go`.
- [ ] 1.5 Update `main.go` to import and use `pkg/runner`.
- [ ] 1.6 Verify the Qt app still builds and runs (`go build`).

## 2. DarwinKit App Structure
- [ ] 2.1 Create `cmd/mac-app` directory and `main.go`.
- [ ] 2.2 Implement the main application skeleton using `macos.RunApp`.
- [ ] 2.2 Create the main `NSWindow` and set basic properties (title, size).
- [ ] 2.3 Implement the layout (using `NSStackView` or absolute positioning if necessary) for:
    - File input section (TextField + Browse Button).
    - Progress Bar.
    - Start/Stop Buttons.
    - Log TextView (inside ScrollView).
    - Settings inputs (API Key, URL, Model, Prompt, CJK Checkbox).

## 3. Logic Integration
- [ ] 3.1 Implement the "Browse" button logic using `NSOpenPanel`.
- [ ] 3.2 Implement "Start Translation" handler:
    - Read values from UI inputs.
    - Construct `pkg/config` or direct structs.
    - Call `RunTranslation` in a goroutine.
- [ ] 3.3 Implement Callbacks for `RunTranslation`:
    - `OnProgress`: Update `NSProgressIndicator` (dispatch to main thread).
    - `OnLog/Translated`: Append to `NSTextView` (dispatch to main thread).
    - `OnComplete`: Reset UI state.
- [ ] 3.4 Implement "Stop Translation" handler (cancel context).
- [ ] 3.5 Implement `SaveConfig` functionality (on start or explicitly).

## 4. Build & Verify
- [ ] 4.1 Create a `make mac-app` target in `Makefile`.
- [ ] 4.2 Verify compilation and runtime on macOS.
