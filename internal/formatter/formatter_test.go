package formatter

import (
	"bytes"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, false, false)

	if f.output != &buf {
		t.Error("Formatter should store the output writer")
	}

	if f.verbose {
		t.Error("Verbose should be false")
	}

	if f.showUsage {
		t.Error("ShowUsage should be false")
	}
}

func TestFormat_AssistantMessage(t *testing.T) {
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Hello, world!"}]}}`
	var output bytes.Buffer

	f := New(&output, false, false)
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// The text is buffered internally and printed when result arrives
	// Without a result message, text remains in the buffer but gets printed at end
	result := output.String()
	if !strings.Contains(result, "Hello, world!") {
		t.Errorf("Output should contain buffered text, got: %s", result)
	}
}

func TestFormat_ResultMessage(t *testing.T) {
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Hello, world!"}]}}
{"type":"result","result":"Hello, world!","is_error":false}`

	var output bytes.Buffer

	f := New(&output, false, false)
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()
	if !strings.Contains(result, "Hello, world!") {
		t.Errorf("Output should contain 'Hello, world!', got: %s", result)
	}
}

func TestFormat_MultipleMessages(t *testing.T) {
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"First message"}]}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":" Second message"}]}}
{"type":"result","result":"First message Second message","is_error":false}`

	var output bytes.Buffer

	f := New(&output, false, false)
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()
	if !strings.Contains(result, "First message Second message") {
		t.Errorf("Output should contain concatenated messages, got: %s", result)
	}
}

func TestFormat_WithUsage(t *testing.T) {
	input := `{"type":"result","result":"Task complete","is_error":false,"usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":25,"cache_creation_input_tokens":10}}`

	var output bytes.Buffer

	f := New(&output, false, true) // showUsage = true
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Check for usage statistics
	expectedStrings := []string{
		"Usage Statistics",
		"Input tokens: 100",
		"Output tokens: 50",
		"Cache read tokens: 25",
		"Cache creation tokens: 10",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result, expected) {
			t.Errorf("Output should contain '%s', got: %s", expected, result)
		}
	}
}

func TestFormat_WithoutUsage(t *testing.T) {
	input := `{"type":"result","result":"Task complete","is_error":false,"usage":{"input_tokens":100,"output_tokens":50}}`

	var output bytes.Buffer

	f := New(&output, false, false) // showUsage = false
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Should NOT contain usage statistics
	if strings.Contains(result, "Usage Statistics") {
		t.Errorf("Output should not contain usage statistics when showUsage is false, got: %s", result)
	}

	// Should still contain the result
	if !strings.Contains(result, "Task complete") {
		t.Errorf("Output should contain the result, got: %s", result)
	}
}

func TestFormat_ErrorResult(t *testing.T) {
	input := `{"type":"result","result":"","is_error":true,"subtype":"permission_denied"}`

	var output bytes.Buffer

	f := New(&output, false, false)
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Should contain error information
	if !strings.Contains(result, "[ERROR]") {
		t.Errorf("Output should contain error marker, got: %s", result)
	}

	if !strings.Contains(result, "permission_denied") {
		t.Errorf("Output should contain error subtype, got: %s", result)
	}
}

func TestFormat_VerboseMode(t *testing.T) {
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Test"}]}}`

	var output bytes.Buffer

	f := New(&output, true, false) // verbose = true
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Should contain debug output with raw JSON
	if !strings.Contains(result, "[DEBUG]") {
		t.Errorf("Verbose output should contain [DEBUG] marker, got: %s", result)
	}

	if !strings.Contains(result, `"type":"assistant"`) {
		t.Errorf("Verbose output should contain raw JSON, got: %s", result)
	}
}

func TestFormat_InvalidJSON(t *testing.T) {
	input := `{"invalid json`

	var output bytes.Buffer

	f := New(&output, true, false) // verbose mode to see error
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format should not fail on invalid JSON, it should skip it: %v", err)
	}

	result := output.String()

	// In verbose mode, should see error message
	if !strings.Contains(result, "[ERROR]") {
		t.Errorf("Verbose output should contain error about invalid JSON, got: %s", result)
	}
}

func TestFormat_EmptyLines(t *testing.T) {
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Test"}]}}

{"type":"result","result":"Test","is_error":false}
`

	var output bytes.Buffer

	f := New(&output, false, false)
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Should handle empty lines gracefully
	result := output.String()
	if !strings.Contains(result, "Test") {
		t.Errorf("Output should contain the text, got: %s", result)
	}
}

func TestPrintUsage(t *testing.T) {
	var output bytes.Buffer

	f := New(&output, false, true)

	usage := &Usage{
		InputTokens:              100,
		OutputTokens:             50,
		CacheReadInputTokens:     25,
		CacheCreationInputTokens: 10,
		CacheCreation: map[string]int{
			"ephemeral_5m_input_tokens": 5,
		},
		ServerToolUse: map[string]int{
			"web_search_requests": 3,
		},
	}

	f.printUsage(usage)

	result := output.String()

	expectedStrings := []string{
		"Usage Statistics",
		"Input tokens: 100",
		"Output tokens: 50",
		"Cache read tokens: 25",
		"Cache creation tokens: 10",
		"Cache creation (ephemeral_5m_input_tokens): 5",
		"Server tool use (web_search_requests): 3",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result, expected) {
			t.Errorf("Usage output should contain '%s', got: %s", expected, result)
		}
	}
}
