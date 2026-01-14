# Change: Add DarwinKit GUI

## Why
The project currently uses Qt (via `miqt`) for its GUI. The user has requested a native macOS GUI using `darwinkit`, which allows building macOS applications using native Cocoa APIs from Go. This provides a more native look and feel on macOS and serves as an alternative frontend.

## What Changes
- **New Entry Point:** A new main package (`cmd/mac-app`) will be created for the DarwinKit-based application.
- **Dependency:** The project will integrate the local `darwinkit` directory as a module dependency.
- **UI Implementation:** A new GUI implementation using `AppKit` widgets (Window, Button, TextField, etc.) that replicates the functionality of the existing Qt app.
- **Build System:** Updates to `Makefile` to build the new target.

## Impact
- **Affected Specs:** `gui-darwinkit` (New Capability).
- **Affected Code:** `go.mod`, `Makefile`, new directory `cmd/mac-app`.
- **No Impact:** Core logic in `pkg/` remains unchanged and will be reused.
