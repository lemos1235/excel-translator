gogio -target macos -arch arm64 \
  --ldflags="-s -w" \
  -icon appicon.png  \
  -o ./out/Excel\ Translator.app \
  cmd/qt/main.go
