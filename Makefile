.PHONY: dev build test clean darwin-arm64 windows-amd64

# 开发模式：修改 Go 代码后自动重新编译并重启
dev:
	go tool mygo dev

# 构建当前平台的生产版本
build:
	go tool mygo build

darwin-arm64:
	go tool mygo build -platform darwin/arm64

windows-amd64:
	go tool mygo build -platform windows/amd64

# 运行全部测试
test:
	go test ./...

clean:
	rm -rf build .mygo
