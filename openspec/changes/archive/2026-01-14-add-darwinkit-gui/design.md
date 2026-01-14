## Context
The goal is to provide a native macOS interface using the `darwinkit` library which is present locally in the project root. This library wraps macOS Cocoa APIs (AppKit, Foundation, etc.).

## Goals
- Create a functional GUI that mirrors the features of the Qt version:
  - File selection (drag & drop, browse).
  - Configuration settings (API Key, Model, etc.).
  - Progress tracking and logging.
  - Translation execution.
- Reuse existing core business logic (`pkg/fileprocessor`, `pkg/translator`, `pkg/config`).
- Ensure the UI remains responsive during long-running translation tasks.

## Decisions

### 1. Separate Entry Point
- **Decision:** Create `cmd/mac-app/main.go` instead of modifying the root `main.go`.
- **Why:** To cleanly separate the Qt build (which has specific CGO/library requirements) from the DarwinKit build. This avoids complex build tags and conditional logic in a single file.

### 2. Refactor Shared Logic
- **Decision:** Extract `RunTranslation` and `TranslationCallbacks` from `main.go` to a new package `pkg/runner`.
- **Why:** The translation orchestration logic is currently embedded in the Qt `main.go`. To allow `cmd/mac-app` to reuse the same logic without duplicating code or importing `main` (which isn't allowed), this must be moved to a shared library package.

### 3. Local Module Replacement
- **Decision:** Use `go mod edit -replace github.com/progrium/darwinkit=./darwinkit` in the root `go.mod`.
- **Why:** The `darwinkit` library is vendored/cloned locally. We need to tell the Go toolchain to use this local version instead of fetching from GitHub.

### 4. UI Component Mapping
- **Qt** -> **AppKit** mapping:
  - `QMainWindow` -> `NSWindow`
  - `QLineEdit` -> `NSTextField`
  - `QPushButton` -> `NSButton`
  - `QProgressBar` -> `NSProgressIndicator`
  - `QTextEdit` -> `NSTextView` (wrapped in `NSScrollView`)
  - `QCheckBox` -> `NSButton` (with `SwitchButton` or `Checkbox` type)
  - `QFileDialog` -> `NSOpenPanel` / `NSSavePanel`

### 5. Concurrency Model
- **Decision:** Use `darwinkit`'s main thread dispatch mechanism (likely exposed via `dispatch` package or implicit in the run loop) to update UI components from the translation callbacks.
- The `RunTranslation` function in `pkg` accepts callbacks. These callbacks (OnProgress, OnLog, etc.) will wrap their UI update logic in `dispatch.MainQueue(func() { ... })` to ensure thread safety.

## Open Questions
- Does the local `darwinkit` version support all necessary widgets (e.g., Drag & Drop on `NSTextField` or `NSView`)?
  - *Assumption:* Yes, or we can fall back to standard button-based interactions if Drag & Drop is too complex to implement in the first pass.

## Risks
- **CGO Complexity:** `darwinkit` relies on CGO and specific macOS frameworks. Build errors might occur if the environment isn't set up correctly, but the user is on `darwin`.