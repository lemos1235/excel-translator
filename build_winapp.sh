#!/bin/bash
set -e

cd "$(dirname "$0")"

# 递增版本号并同步到 mygo.json
source ./version.sh
bump_version
sync_mygo_json

# 通过 MYGO_LDFLAGS 把完整版本号注入 pkg/version.Full
export MYGO_LDFLAGS="$(version_ldflags)"

echo "=== 构建 Windows 应用 ==="
args=(-platform windows/amd64)
if ! command -v makensis >/dev/null 2>&1; then
    echo ">>> 未找到 makensis，跳过 Windows 安装器（仍会生成 .exe）"
    args+=(-skip-installer)
fi
go tool mygo build "${args[@]}"

echo "=== 构建完成 ==="
echo "输出目录: build/windows-amd64/"
ls -lh "build/windows-amd64/" 2>/dev/null || true
