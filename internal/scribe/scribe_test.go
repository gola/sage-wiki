package scribe

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCompressSession_StringContent(t *testing.T) {
	input := []byte(`{"role":"user","content":"How does WAL mode work in SQLite?"}
{"role":"assistant","content":"<thinking>Let me explain WAL mode.</thinking>WAL (Write-Ahead Logging) allows concurrent reads during writes."}
{"role":"user","type":"tool_result","content":"file contents here"}
{"role":"assistant","type":"thinking","content":"deep thinking here"}
{"role":"user","content":"Thanks, that helps."}
`)

	compressed := compressSession(input)

	if !strings.Contains(compressed, "How does WAL mode work") {
		t.Error("should keep user message")
	}
	if !strings.Contains(compressed, "Thanks, that helps") {
		t.Error("should keep second user message")
	}
	if strings.Contains(compressed, "Let me explain WAL mode") {
		t.Error("should strip <thinking> tags from assistant content")
	}
	if !strings.Contains(compressed, "WAL (Write-Ahead Logging)") {
		t.Error("should keep assistant content outside thinking tags")
	}
	if strings.Contains(compressed, "file contents here") {
		t.Error("should skip tool_result messages")
	}
	if strings.Contains(compressed, "deep thinking here") {
		t.Error("should skip thinking type messages")
	}
}

func TestCompressSession_ArrayContent(t *testing.T) {
	// Real Claude session format: assistant content is an array of blocks
	input := []byte(`{"role":"assistant","content":[{"type":"text","text":"Here is the explanation of WAL mode."},{"type":"tool_use","id":"tool_1","name":"read_file"}]}
{"role":"assistant","content":[{"type":"text","text":"The second response."},{"type":"text","text":"With two text blocks."}]}
`)

	compressed := compressSession(input)

	if !strings.Contains(compressed, "Here is the explanation of WAL mode") {
		t.Error("should extract text from content blocks array")
	}
	if strings.Contains(compressed, "tool_use") {
		t.Error("should not include tool_use block content")
	}
	if !strings.Contains(compressed, "The second response") {
		t.Error("should extract first text block from second message")
	}
	if !strings.Contains(compressed, "With two text blocks") {
		t.Error("should extract second text block from second message")
	}
}

func TestCompressSession_Empty(t *testing.T) {
	if result := compressSession([]byte("")); result != "" {
		t.Errorf("empty input should produce empty output, got %q", result)
	}

	if result := compressSession([]byte("not json\ngarbage\n")); result != "" {
		t.Errorf("non-JSON input should produce empty output, got %q", result)
	}
}

func TestExtractTextContent(t *testing.T) {
	// Plain string
	result := extractTextContent([]byte(`"hello world"`))
	if result != "hello world" {
		t.Errorf("string content = %q, want 'hello world'", result)
	}

	// Array of blocks
	result = extractTextContent([]byte(`[{"type":"text","text":"first"},{"type":"tool_use","text":"skip"},{"type":"text","text":"second"}]`))
	if result != "first\nsecond" {
		t.Errorf("array content = %q, want 'first\\nsecond'", result)
	}

	// Empty
	result = extractTextContent([]byte(``))
	if result != "" {
		t.Errorf("empty = %q, want empty", result)
	}

	// Null
	result = extractTextContent([]byte(`null`))
	if result != "" {
		t.Errorf("null = %q, want empty", result)
	}
}

func TestIsKebabCase(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"flash-attention", true},
		{"wal-mode-sqlite", true},
		{"concept", true},
		{"ab", true},
		{"a", false},
		{"", false},
		{"Flash-Attention", false},
		{"flash_attention", false},
		{"123-abc", false},
		{"flash--attention", false},
	}

	for _, tt := range tests {
		got := isKebabCase(tt.input)
		if got != tt.want {
			t.Errorf("isKebabCase(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestStripThinkingTags(t *testing.T) {
	input := "Before <thinking>some thought</thinking> after"
	result := stripThinkingTags(input)
	if result != "Before  after" {
		t.Errorf("stripThinkingTags = %q, want 'Before  after'", result)
	}

	if stripThinkingTags("no tags") != "no tags" {
		t.Error("no tags should be unchanged")
	}

	input2 := "<thinking>a</thinking>middle<thinking>b</thinking>end"
	result2 := stripThinkingTags(input2)
	if result2 != "middleend" {
		t.Errorf("multiple blocks = %q, want 'middleend'", result2)
	}
}

func TestTruncateUTF8(t *testing.T) {
	// ASCII only — should truncate cleanly
	ascii := "hello world"
	if got := truncateUTF8(ascii, 5); got != "hello" {
		t.Errorf("ascii truncate = %q, want 'hello'", got)
	}

	// Below limit — returns unchanged
	if got := truncateUTF8(ascii, 100); got != ascii {
		t.Errorf("below limit = %q, want unchanged", got)
	}

	// Multi-byte: CJK characters are 3 bytes each (e.g., U+4E16 "世" = 0xE4 0xB8 0x96)
	// Build a string of CJK chars where the byte limit falls mid-rune.
	cjk := strings.Repeat("世", 100) // 300 bytes
	// Truncate at 10 bytes: 3 full CJK chars = 9 bytes, next starts at byte 9
	got := truncateUTF8(cjk, 10)
	if !utf8.ValidString(got) {
		t.Errorf("truncated CJK is not valid UTF-8: %q", got)
	}
	if len(got) != 9 { // 3 chars × 3 bytes
		t.Errorf("truncated CJK len = %d, want 9 (3 chars)", len(got))
	}

	// 4-byte characters (emoji): U+1F600 "😀" = 4 bytes
	emoji := strings.Repeat("😀", 50) // 200 bytes
	got = truncateUTF8(emoji, 6) // 1 full emoji = 4 bytes, 2nd starts at byte 4
	if !utf8.ValidString(got) {
		t.Errorf("truncated emoji is not valid UTF-8: %q", got)
	}
	if len(got) != 4 { // 1 emoji × 4 bytes
		t.Errorf("truncated emoji len = %d, want 4 (1 emoji)", len(got))
	}

	// Empty string
	if got := truncateUTF8("", 10); got != "" {
		t.Errorf("empty string = %q, want empty", got)
	}
}
