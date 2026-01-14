# Change: Fix Mac Logging Missing Content

## Why
The `NSTextView` used for logging in the Mac app is not displaying text, nor is it editable when `SetEditable(true)` is called. This persists even after setting a fixed height on its container `NSScrollView`. This suggests an issue with the `NSTextView` initialization, sizing (frame), or its integration with `NSScrollView`.

## What Changes
- **Initialization:** Use `NewScrollableTextView()` helper if available (seen in `darwinkit` source previously) or manually configure the `NSTextView` with `SetVerticallyResizable(true)`, `SetAutoresizingMask(WidthSizable | HeightSizable)`, and a non-zero initial frame.
- **Scroll View Integration:** Ensure the `NSScrollView` is correctly configured to host the text view.

## Impact
- **Affected Specs:** `gui-darwinkit` (MODIFIED).
- **Affected Code:** `cmd/mac-app/main.go`.
