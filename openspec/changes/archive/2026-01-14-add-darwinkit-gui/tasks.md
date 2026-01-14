## 1. Setup & Refactoring
- [x] 1.1 Update `go.mod` to require and replace `github.com/progrium/darwinkit` with `./darwinkit`.
- [x] 1.2 Run `go mod tidy`.
- [x] 1.3 Create new package `pkg/runner`.
- [x] 1.4 Move `RunTranslation` and `TranslationCallbacks` from `main.go` to `pkg/runner/runner.go`.
- [x] 1.5 Update `main.go` to import and use `pkg/runner`.
- [x] 1.6 Verify the Qt app still builds and runs (`go build`).

## 2. DarwinKit App Structure
- [x] 2.1 Create `cmd/mac-app` directory and `main.go`.
- [x] 2.2 Implement the main application skeleton using `macos.RunApp`.
- [x] 2.2 Create the main `NSWindow` and set basic properties (title, size).
- [x] 2.3 Implement the layout (using `NSStackView` or absolute positioning if necessary) for:
    - File input section (TextField + Browse Button).
    - Progress Bar.
    - Start/Stop Buttons.
    - Log TextView (inside ScrollView).
    - Settings inputs (API Key, URL, Model, Prompt, CJK Checkbox).

## 3. Logic Integration
- [x] 3.1 Implement the "Browse" button logic using `NSOpenPanel`.
- [x] 3.2 Implement "Start Translation" handler:
    - Read values from UI inputs.
    - Construct `pkg/config` or direct structs.
    - Call `RunTranslation` in a goroutine.
- [x] 3.3 Implement Callbacks for `RunTranslation`:
    - `OnProgress`: Update `NSProgressIndicator` (dispatch to main thread).
    - `OnLog/Translated`: Append to `NSTextView` (dispatch to main thread).
    - `OnComplete`: Reset UI state.
- [x] 3.4 Implement "Stop Translation" handler (cancel context).
- [x] 3.5 Implement `SaveConfig` functionality (on start or explicitly).

## 4. Build & Verify
- [x] 4.1 Create a `make mac-app` target in `Makefile`.
- [x] 4.2 Verify compilation and runtime on macOS.