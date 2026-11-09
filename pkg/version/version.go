// Package version 提供应用的版本信息。
//
// 版本号维护在仓库根目录的 VERSION 文件中，格式为 <语义化版本>+<发布号>，
// 例如 1.0.0+2。构建脚本会读取该文件并通过
// -ldflags "-X exceltranslator/pkg/version.Full=1.0.0+2" 注入到二进制中。
package version

// Full 完整版本号，形如 1.0.0+2；未经构建脚本注入时为 dev。
var Full = "dev"
