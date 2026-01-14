# Project Context

## Purpose
Excel Translator is a desktop GUI application designed to translate the content of Excel (.xlsx) and Word (.docx) documents while preserving their original formatting and styles. It leverages advanced LLMs (Large Language Models) to provide high-quality translations, with specific support for filtering and translating CJK (Chinese, Japanese, Korean) text.

## Tech Stack
- **Language:** Go 1.24
- **GUI Framework:** Qt 6 (via [miqt](https://github.com/mappu/miqt) bindings)
- **LLM Integration:** `github.com/openai/openai-go/v3` (Compatible with OpenAI and Aliyun DashScope)
- **Configuration:** TOML (`github.com/pelletier/go-toml/v2`)
- **Document Processing:** Native Go `archive/zip` and `regexp` for direct XML manipulation (OOXML)
- **Build System:** Makefile

## Project Conventions

### Code Style
- Follows standard Go formatting and idioms (`gofmt`).
- Error handling uses explicit error return values and wrapping (`fmt.Errorf("...: %w", err)`).
- Comments explain "why" rather than "what", particularly for complex regex or XML logic.

### Architecture Patterns
- **Layered Architecture:**
    - **UI Layer (`main.go`):** Handles Qt window management, event loops, and user interaction. Uses `mainthread` helper for safe UI updates from background goroutines.
    - **Application Layer (`pkg/fileprocessor`):** Orchestrates the file processing pipeline (unzip -> extract -> translate -> replace -> zip).
    - **Domain Layer (`pkg/textextractor`, `pkg/translator`):** Contains core logic for identifying translatable text and managing the translation workflow.
    - **Infrastructure Layer (`pkg/llmservice`, `pkg/config`):** Handles external API communication and local configuration persistence.
- **Concurrency:** Uses `context.Context` for cancellation and goroutines for non-blocking UI operations during file processing.

### Document Processing Strategy
- **Direct XML Manipulation:** Instead of using heavy object models (like `excelize`), the project treats .xlsx/.docx files as ZIP archives and uses efficient Regex-based parsing to locate and replace text nodes (e.g., `<w:t>`, `<t>`) within the internal XML structure.
- **Streaming/Batching:** Text is extracted and sent to the LLM in batches to optimize API usage.

### Testing Strategy
- **Current State:** Minimal automated testing.
- **Goal:** Implement unit tests for core logic in `pkg/textextractor` and `pkg/translator`.

### Git Workflow
- Standard feature-branch workflow.
- Commit messages should be clear and descriptive.

## Domain Context
- **OOXML Structure:** Understanding the internal XML structure of Office documents (Word/Excel) is crucial.
    - Word: `word/document.xml`, `word/header*.xml`, `word/footer*.xml`
    - Excel: `xl/sharedStrings.xml`, `xl/drawings/drawing*.xml`, `xl/workbook.xml`
- **CJK Filtering:** Logic exists to specifically target or filter Chinese, Japanese, and Korean characters using unicode ranges.
- **Excel Constraints:** Sheet names are strictly limited to 31 characters.

## Important Constraints
- **GUI Dependencies:** Requires Qt 6 libraries to be available on the system or bundled.
- **API dependency:** Relies on a valid API key and internet connection for the configured LLM provider.
- **Performance:** Large documents are processed via streaming (zip) but memory usage should be monitored when holding large shared string tables.

## External Dependencies
- **LLM Provider:** OpenAI API compatible service (default configuration examples use Aliyun DashScope).
- **Qt 6:** Runtime requirement for the GUI.