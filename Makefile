QT_GO_FILES="./main.go"
QT_APP_NAME="Excel Translator.app"

MAC_GO_FILES="./cmd/mac-app"
MAC_APP_NAME="Excel Translator (Mac Native).app"

APP_OUT=out

qt-app:
	gogio -target macos -arch arm64 -icon appicon.png --ldflags="-s -w" -o $(APP_OUT)/$(QT_APP_NAME) $(QT_GO_FILES)

mac-app:
	gogio -target macos -arch arm64 -icon appicon.png --ldflags="-s -w"  -o $(APP_OUT)/$(MAC_APP_NAME) $(MAC_GO_FILES)
