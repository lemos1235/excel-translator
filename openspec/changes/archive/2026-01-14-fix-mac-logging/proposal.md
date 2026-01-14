# Change: Fix Mac Logging

## Why
The console log section in the native Mac app (DarwinKit) is not displaying any content. This is likely because the `NSScrollView` containing the log view has no intrinsic height and collapses to zero within the `NSStackView`.

## What Changes
- **Constraints:** Explicitly set a minimum height constraint for the log scroll view.
- **Scroll Logic:** Ensure the text view updates and scrolls correctly.

## Impact
- **Affected Specs:** `gui-darwinkit` (MODIFIED).
- **Affected Code:** `cmd/mac-app/main.go`.
