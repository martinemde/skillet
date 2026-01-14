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
	if !strings.Contains(result, "ERRO") {
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
	if !strings.Contains(result, "DEBU") {
		t.Errorf("Verbose output should contain DEBU marker, got: %s", result)
	}

	// The JSON is logged in the data field (may be escaped)
	if !strings.Contains(result, "assistant") || !strings.Contains(result, "type") {
		t.Errorf("Verbose output should contain raw JSON data, got: %s", result)
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
	if !strings.Contains(result, "DEBU") {
		t.Errorf("Verbose output should contain debug info about invalid JSON, got: %s", result)
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

func TestFormat_SystemInit(t *testing.T) {
	input := `{"type":"system","subtype":"init"}`

	var output bytes.Buffer

	f := New(&output, false, false)
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Should contain session initialized message
	if !strings.Contains(result, "Session initialized") {
		t.Errorf("Output should contain 'Session initialized', got: %s", result)
	}
}

func TestFormat_ToolCall(t *testing.T) {
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_123","name":"Glob","input":{"pattern":"**/*.md"}}]}}`

	var output bytes.Buffer

	f := New(&output, false, false)
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Should contain tool call information
	if !strings.Contains(result, "Tool call") {
		t.Errorf("Output should contain 'Tool call', got: %s", result)
	}

	if !strings.Contains(result, "Glob") {
		t.Errorf("Output should contain tool name 'Glob', got: %s", result)
	}

	if !strings.Contains(result, "pattern=**/*.md") {
		t.Errorf("Output should contain tool input, got: %s", result)
	}
}

func TestFormat_ToolResult(t *testing.T) {
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_123","name":"Read","input":{"file_path":"/home/user/test.go"}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_123","content":"package main\n\nfunc main() {}"}]}}`

	var output bytes.Buffer

	f := New(&output, false, false)
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Should contain tool call and result
	if !strings.Contains(result, "Tool call") {
		t.Errorf("Output should contain 'Tool call', got: %s", result)
	}

	if !strings.Contains(result, "Tool result") {
		t.Errorf("Output should contain 'Tool result', got: %s", result)
	}

	if !strings.Contains(result, "Read") {
		t.Errorf("Output should contain tool name, got: %s", result)
	}
}

func TestFormat_CompleteWorkflow(t *testing.T) {
	// Simulate a complete workflow with system init, tool calls, and results
	input := `{"type":"system","subtype":"init"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I'll search for markdown files."}]}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_001","name":"Glob","input":{"pattern":"**/*.md"}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_001","content":"README.md\nSKILL.md\nDOCS.md"}]}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Found 3 markdown files. Let me read one."}]}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_002","name":"Read","input":{"file_path":"SKILL.md"}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_002","content":"---\nname: test-skill\ndescription: A test skill\n---\n\n# Instructions"}]}}
{"type":"result","result":"I'll search for markdown files.\nFound 3 markdown files. Let me read one.","is_error":false}`

	var output bytes.Buffer

	f := New(&output, false, false)
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Check for all expected elements
	expectedStrings := []string{
		"Session initialized",
		"Tool call",
		"Glob",
		"Read",
		"Tool result",
		"I'll search for markdown files",
		"Found 3 markdown files",
		"Execution completed",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result, expected) {
			t.Errorf("Output should contain '%s', got: %s", expected, result)
		}
	}
}

func TestFormatToolInput_LongValue(t *testing.T) {
	f := New(&bytes.Buffer{}, false, false)

	input := map[string]interface{}{
		"pattern": "this is a very long pattern that should be truncated to avoid cluttering the output with too much detail",
	}

	result := f.formatToolInput(input)

	// Should truncate long values
	if len(result) > 100 {
		t.Errorf("formatToolInput should truncate long values, got length: %d", len(result))
	}

	if !strings.Contains(result, "...") {
		t.Errorf("formatToolInput should show truncation with '...', got: %s", result)
	}
}

func TestFormatToolResult_Truncation(t *testing.T) {
	f := New(&bytes.Buffer{}, false, false)

	longResult := strings.Repeat("a", 200)
	result := f.formatToolResult(longResult)

	// Should truncate long results
	if len(result) > 100 {
		t.Errorf("formatToolResult should truncate long results, got length: %d", len(result))
	}

	if !strings.Contains(result, "...") {
		t.Errorf("formatToolResult should show truncation with '...', got: %s", result)
	}
}
