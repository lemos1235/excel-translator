MAIN_GO_FILES="./main.go"
MAC_APP_NAME="Excel Translator.app"
APP_OUT="./out"

mac-app:
	gogio -target macos -arch arm64 -icon appicon.png --ldflags="-s -w" -o $(APP_OUT)/$(MAC_APP_NAME) $(MAIN_GO_FILES)
