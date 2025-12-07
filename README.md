# Excel Translator

一个简单易用的 Excel 文件翻译工具。

## 主要特性

- 支持Excel文件中的文本单元格翻译
- 支持Excel形状和图表中的文本翻译
- 保留原始格式和样式
- 使用先进的AI模型进行高质量翻译
- 简洁易用的图形界面

## 配置说明

应用程序在首次运行时会在用户配置目录下创建默认配置文件：

- Windows: `%APPDATA%\Excel-Translator\config.toml`
- macOS: `~/Library/Application\ Support/Excel-Translator/config.toml`

你可以编辑此配置文件来自定义应用程序的行为：

```toml
[llm]
base_url = 'https://dashscope.aliyuncs.com/compatible-mode/v1'
api_key = 'sk-'
model = 'qwen-flash'
prompt = 'You are a professional translator. Translate it to Simplified Chinese directly. Keep all alphanu    meric characters unchanged. Ensure accuracy of technical terms. No explanations needed.'

[extractor]
# 是否仅翻译CJK文本
cjk_only = true
```

## GUI 版本

安装依赖，请参考 https://github.com/mappu/miqt

编译 & 打包:

```bash
make darwin-gui
```

截图:

![screenshot.png](demo/screenshot.png)
