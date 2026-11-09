#!/bin/bash
# =============================================================================
# 版本号工具（供各构建脚本 source 使用）
#
# 版本号维护在仓库根目录的 VERSION 文件中，格式为 <语义化版本>+<发布号>，
# 例如 1.0.0+2。构建时通过 ldflags 注入到 exceltranslator/pkg/version.Full。
# =============================================================================

# 兜底 $0：脚本被非 bash 的 shell（如 zsh）source 时 BASH_SOURCE 未定义
VERSION_FILE="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)/VERSION"

# read_version 读取当前版本号并解析到 FULL_VERSION/SHORT_VERSION/BUILD_NUMBER
read_version() {
    if [ ! -f "$VERSION_FILE" ]; then
        echo "1.0.0+1" > "$VERSION_FILE"
    fi
    FULL_VERSION=$(tr -d '[:space:]' < "$VERSION_FILE")
    SHORT_VERSION=${FULL_VERSION%%+*}
    case "$FULL_VERSION" in
        *+*) BUILD_NUMBER=${FULL_VERSION##*+} ;;
        *)   BUILD_NUMBER=1 ;;
    esac
}

# bump_version 递增末位发布号（1.0.0+1 -> 1.0.0+2）并写回 VERSION 文件
bump_version() {
    read_version
    BUILD_NUMBER=$((BUILD_NUMBER + 1))
    FULL_VERSION="$SHORT_VERSION+$BUILD_NUMBER"
    printf '%s\n' "$FULL_VERSION" > "$VERSION_FILE"
    echo ">>> 构建版本: $FULL_VERSION"
}

# version_ldflags 输出注入版本号所需的 ldflags 片段
version_ldflags() {
    printf -- '-X exceltranslator/pkg/version.Full=%s' "$FULL_VERSION"
}

# sync_mygo_json 把 SHORT_VERSION 同步到 mygo.json 的 version 字段，
# 供 mygo build 生成安装包、磁盘映像和文件属性时使用。
sync_mygo_json() {
    read_version
    local root file tmp
    root="$(dirname "$VERSION_FILE")"
    file="$root/mygo.json"
    [ -f "$file" ] || return 0

    tmp="$(mktemp "${file}.XXXXXX")"
    sed "s/\"version\"[[:space:]]*:[[:space:]]*\"[^\"]*\"/\"version\": \"$SHORT_VERSION\"/" "$file" > "$tmp"
    mv "$tmp" "$file"
    echo ">>> mygo.json 版本: $SHORT_VERSION"
}
