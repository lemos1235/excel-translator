#!/bin/bash
set -e

cd "$(dirname "$0")"

# 递增版本号并同步到 mygo.json
source ./version.sh
bump_version
sync_mygo_json

# 通过 MYGO_LDFLAGS 把完整版本号注入 pkg/version.Full
export MYGO_LDFLAGS="$(version_ldflags)"

echo "=== 构建 macOS 应用 ==="
go tool mygo build -platform darwin/arm64

echo "=== 构建完成 ==="
echo "输出目录: build/darwin-arm64/"
ls -lh "build/darwin-arm64/" 2>/dev/null || true
