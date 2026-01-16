# Excel Translator

An easy-to-use tool for translating Excel files.

## Key Features

-   Supports translation of text cells within Excel files.
-   Supports translation of text within Excel shapes and charts.
-   Preserves original formatting and styles.
-   Utilizes advanced AI models for high-quality translation.
-   Provides a clean and intuitive graphical user interface (GUI).

## Configuration

Upon its first run, the application creates a default configuration file in the user's configuration directory:

-   Windows: `%APPDATA%\Excel-Translator\config.toml`
-   macOS: `~/Library/Application\ Support/Excel-Translator/config.toml`

You can edit this configuration file to customize the application's behavior:

```toml
[llm]
base_url = 'https://dashscope.aliyuncs.com/compatible-mode/v1'
api_key = 'sk-'
model = 'qwen-flash'
prompt = 'Translate to Simplified Chinese.Ignore if already Chinese. Keep all numbers and letters intact.'

[extractor]
# Translate only CJK (Chinese, Japanese, Korean) text
cjk_only = true
```

## GUI

To install dependencies, please refer to https://github.com/mappu/miqt

To compile and package:

```bash
make mac-app
```

Screenshot:

![screenshot.png](demo/screenshot.png)