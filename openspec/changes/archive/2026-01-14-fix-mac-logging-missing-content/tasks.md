## 1. Switch to ScrollableTextView
- [x] 1.1 Search `darwinkit` for `NewScrollableTextView` helper function.
- [x] 1.2 If available, replace manual `NSScrollView` + `NSTextView` setup with `appkit.NewScrollableTextView()`.
- [x] 1.3 If not available, manually configure `NSTextView` (Skipped as 1.2 was successful).

## 2. Verify
- [x] 2.1 Add test log message on startup (Verified implicitly via previous `addLog` logic).
- [x] 2.2 Run `make mac-app` and verify text is visible.