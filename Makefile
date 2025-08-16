NAME=exceltranslator
BINDIR=bin
GOBUILD=CGO_ENABLED=1 go build --ldflags="-s -w" -v -x -a
GOFILES=main.go
APPDIR=out
APPNAME="Excel Translator.app"

PLATFORM_LIST = \
	darwin-amd64 \
	darwin-arm64 \
	linux-amd64

WINDOWS_ARCH_LIST = \
	windows-amd64

darwin-gui:
	gogio -target macos -arch arm64 -icon appicon.png --ldflags="-s -w" -o $(APPDIR)/$(APPNAME) $(GOFILES)

darwin-amd64:
	GOARCH=amd64 GOOS=darwin $(GOBUILD) -o $(BINDIR)/$(NAME)-$@ $(GOFILES)

darwin-arm64:
	GOARCH=arm64 GOOS=darwin $(GOBUILD) -o $(BINDIR)/$(NAME)-$@ $(GOFILES)

linux-amd64:
	GOARCH=amd64 GOOS=linux $(GOBUILD) -o $(BINDIR)/$(NAME)-$@ $(GOFILES)

windows-amd64:
	GOARCH=amd64 GOOS=windows $(GOBUILD) -o $(BINDIR)/$(NAME)-$@.exe $(GOFILES)

gz_releases=$(addsuffix .gz, $(PLATFORM_LIST))

zip_releases=$(addsuffix .zip, $(WINDOWS_ARCH_LIST))

$(gz_releases): %.gz : %
	chmod +x $(BINDIR)/$(NAME)-$(basename $@)
	gzip -f -S -$(VERSION).gz $(BINDIR)/$(NAME)-$(basename $@)

$(zip_releases): %.zip : %
	zip -m -j $(BINDIR)/$(NAME)-$(basename $@)-$(VERSION).zip $(BINDIR)/$(NAME)-$(basename $@).exe

releases: $(gz_releases) $(zip_releases)

clean:
	rm $(BINDIR)/*
