# Excel Translator

一个简单易用的Excel文件翻译工具，可以将Excel文件中的日文内容翻译成中文。

## 主要特性

- 支持Excel文件中的文本单元格翻译
- 支持Excel形状和图表中的文本翻译
- 保留原始格式和样式
- 使用先进的AI模型进行高质量翻译
- 简洁易用的图形界面

## 配置说明

应用程序在首次运行时会在用户配置目录下创建默认配置文件：

- Windows: `%APPDATA%\Exceltranslator\config.toml`
- macOS: `~/Library/Application\ Support/Exceltranslator/config.toml`

你可以编辑此配置文件来自定义应用程序的行为：

```toml
# 客户端设置
[client]
# 最大并发请求数
max_concurrent_requests = 5
# 是否自动检测中日韩文字（仅翻译包含中日韩文字的文本）
auto_detect_cjk = true
# 翻译提示词
prompt = 'You are a professional translator. Translate Japanese to Simplified Chinese directly. Keep all alphanumeric characters unchanged. Ensure accuracy of technical terms. No explanations needed.'

# 大语言模型设置
[llm]
# 使用的模型名称
model = "qwen-plus-latest"
# API 密钥
api_key = "sk-your-api-key-here"
# API 服务器地址
api_url = "https://dashscope.aliyuncs.com/compatible-mode/v1"
```

## 命令行版本

```bash
# 安装依赖
go mod download

# 构建
go build -o bin/exceltranslator cli/main.go

# 运行
./bin/exceltranslator
``` 

## GUI 版本

<img src="demo/screenshot2.png" alt="Excel Translator 演示" width="500" />
