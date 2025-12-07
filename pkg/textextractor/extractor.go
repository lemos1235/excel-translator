package textextractor

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode"
)

var (
	phoneticRunRegex      = regexp.MustCompile(`(?s)<rPh\b[^>]*?>.*?</rPh>`)
	phoneticPropertyRegex = regexp.MustCompile(`(?s)<phoneticPr\b[^>]*?/?>`)
)

// FileType represents the type of file being processed
type FileType string

const (
	FileTypeDocx FileType = "docx"
	FileTypeXlsx FileType = "xlsx"
)

// ExtractorConfig holds configuration for the extraction process
type ExtractorConfig struct {
	CJKOnly bool // If true, only translate text containing CJK characters
}

// Extractor handles text extraction and replacement
type Extractor struct {
	config ExtractorConfig
}

// NewExtractor creates a new Extractor instance
func NewExtractor(config ExtractorConfig) *Extractor {
	return &Extractor{
		config: config,
	}
}

// ContainsCJK checks if the string contains any CJK characters
func ContainsCJK(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) || // Chinese
			(r >= 0x3040 && r <= 0x309F) || // Hiragana
			(r >= 0x30A0 && r <= 0x30FF) || // Katakana
			(r >= 0xAC00 && r <= 0xD7AF) { // Hangul
			return true
		}
	}
	return false
}

// IsValidTextContent checks if the text is valid for translation.
// It returns false for empty strings, pure numbers, or text consisting only of symbols/punctuation.
func IsValidTextContent(s string) bool {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return false
	}

	// Check if it's just numbers and punctuation
	isMeaningful := false
	for _, r := range trimmed {
		if !unicode.IsNumber(r) && !unicode.IsPunct(r) && !unicode.IsSymbol(r) && !unicode.IsSpace(r) {
			isMeaningful = true
			break
		}
	}
	return isMeaningful
}

// ReplaceText finds text nodes in the content and replaces them using the replacer function.
func (e *Extractor) ReplaceText(content string, xmlType string, replacer func(string) (string, error)) (string, error) {
	var re *regexp.Regexp

	// DOCX - word/document.xml, word/header*.xml, word/footer*.xml
	if strings.Contains(xmlType, "word/document.xml") || strings.Contains(xmlType, "word/header") || strings.Contains(xmlType, "word/footer") {
		re = regexp.MustCompile(`<w:t[^>]*?>(.*?)</w:t>`)
	} else if strings.Contains(xmlType, "xl/sharedStrings.xml") {
		// Clean up phonetic annotations (furigana/ruby) which should not be translated
		content = removePhoneticAnnotations(content)
		// XLSX Shared Strings
		re = regexp.MustCompile(`<t>(.*?)</t>`)
	} else if strings.Contains(xmlType, "drawings/drawing") {
		// XLSX Drawings (Shapes)
		re = regexp.MustCompile(`<a:t>(.*?)</a:t>`)
	} else if strings.Contains(xmlType, "xl/workbook.xml") {
		// XLSX Workbook - sheet names
		re = regexp.MustCompile(`<sheet name="([^"]+)"[^>]*?>`)
	} else {
		return content, nil // No translation needed, return original content
	}

	// Find all matches
	matches := re.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content, nil
	}

	type Replacement struct {
		Start int
		End   int
		Text  string
	}
	var replacements []Replacement

	// Extract and translate
	for _, match := range matches {
		// match[0], match[1]: indices of the full match (e.g. <w:t>text</w:t>)
		// match[2], match[3]: indices of the capture group (e.g. text)

		if len(match) < 4 {
			continue
		}

		originalText := content[match[2]:match[3]]

		// Unescape XML entities before processing
		unescaped := html.UnescapeString(originalText)

		// 1. Filter: Check if text is meaningful (not just numbers/symbols)
		if !IsValidTextContent(unescaped) {
			continue
		}

		// 2. Filter: CJK Only check
		if e.config.CJKOnly && !ContainsCJK(unescaped) {
			continue
		}

		// Translate
		translated, err := replacer(unescaped)
		if err != nil {
			return content, fmt.Errorf("failed to translate text '%s': %w", unescaped, err)
		}

		// For sheet names, Excel has a 31-character limit
		if strings.Contains(xmlType, "xl/workbook.xml") {
			translated = truncateSheetName(translated)
		}

		// Escape XML entities after translation
		escapedTranslated := html.EscapeString(translated)

		// Reconstruct the full tag/attribute with the translated content
		// We keep the prefix (before the capture group) and suffix (after the capture group)
		// prefix: content[match[0]:match[2]]
		// suffix: content[match[3]:match[1]]
		replacementStr := content[match[0]:match[2]] + escapedTranslated + content[match[3]:match[1]]

		replacements = append(replacements, Replacement{
			Start: match[0],
			End:   match[1],
			Text:  replacementStr,
		})
	}

	// Apply replacements
	var sb strings.Builder
	lastIndex := 0
	for _, r := range replacements {
		sb.WriteString(content[lastIndex:r.Start])
		sb.WriteString(r.Text)
		lastIndex = r.End
	}
	sb.WriteString(content[lastIndex:])

	return sb.String(), nil
}

// removePhoneticAnnotations strips Excel phonetic (ruby) markup that should not be preserved.
func removePhoneticAnnotations(content string) string {
	content = phoneticRunRegex.ReplaceAllString(content, "")
	content = phoneticPropertyRegex.ReplaceAllString(content, "")
	return content
}

// truncateSheetName enforces Excel's 31-character sheet name limit using rune count.
func truncateSheetName(name string) string {
	const maxRunes = 31
	runes := []rune(name)
	if len(runes) <= maxRunes {
		return name
	}
	return string(runes[:maxRunes])
}
