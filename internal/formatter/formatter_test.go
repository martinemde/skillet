package formatter

import (
	"bytes"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	f := New(Config{
		Output:          &buf,
		Verbose:         false,
		ShowUsage:       false,
		PassthroughMode: false,
	})

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
	// Test with a result message so text gets printed
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Hello, world!"}]}}
{"type":"result","result":"Hello, world!","is_error":false}`
	var output bytes.Buffer

	f := New(Config{Output: &output})
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// The text is printed when result message arrives
	result := output.String()
	if !strings.Contains(result, "Hello, world!") {
		t.Errorf("Output should contain text, got: %s", result)
	}
}

func TestFormat_ResultMessage(t *testing.T) {
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Hello, world!"}]}}
{"type":"result","result":"Hello, world!","is_error":false}`

	var output bytes.Buffer

	f := New(Config{Output: &output})
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

	f := New(Config{Output: &output})
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

	f := New(Config{Output: &output, ShowUsage: true}) // showUsage = true
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Check for usage statistics (now in table format)
	expectedStrings := []string{
		"Usage Statistics", // Table header
		"Input tokens",
		"100",
		"Output tokens",
		"50",
		"Cache read tokens",
		"25",
		"Cache creation tokens",
		"10",
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

	f := New(Config{Output: &output}) // showUsage = false
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

	f := New(Config{Output: &output})
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Should contain error marker (✗ Failed)
	if !strings.Contains(result, "Failed") {
		t.Errorf("Output should contain 'Failed', got: %s", result)
	}
}

func TestFormat_VerboseMode(t *testing.T) {
	// In verbose mode, text is streamed as it comes, not printed at the end
	input := `{"type":"system","subtype":"init"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Test message"}]}}
{"type":"result","result":"Test message","is_error":false}`

	var output bytes.Buffer

	f := New(Config{Output: &output, Verbose: true, SkillName: "test-skill"}) // verbose = true
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// In verbose mode, text is streamed as it arrives
	if !strings.Contains(result, "Test message") {
		t.Errorf("Verbose output should contain streamed text, got: %s", result)
	}
	if !strings.Contains(result, "Starting test-skill") {
		t.Errorf("Verbose output should contain starting message, got: %s", result)
	}
}

func TestFormat_InvalidJSON(t *testing.T) {
	input := `{"invalid json`

	var output bytes.Buffer

	f := New(Config{Output: &output, Verbose: true}) // verbose mode to see error
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format should not fail on invalid JSON, it should skip it: %v", err)
	}

	result := output.String()

	// Invalid JSON is skipped, debug output goes to stderr
	// Just verify the output is empty (no crashes)
	if result != "" {
		t.Errorf("Invalid JSON should be skipped, output should be empty, got: %s", result)
	}
}

func TestFormat_EmptyLines(t *testing.T) {
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Test"}]}}

{"type":"result","result":"Test","is_error":false}
`

	var output bytes.Buffer

	f := New(Config{Output: &output})
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

	// Test usage printing through the complete Format flow
	input := `{"type":"result","result":"Task complete","is_error":false,"usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":25,"cache_creation_input_tokens":10,"cache_creation":{"ephemeral_5m_input_tokens":5},"server_tool_use":{"web_search_requests":3}}}`

	f := New(Config{Output: &output, ShowUsage: true})
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Check for usage statistics (now in table format)
	expectedStrings := []string{
		"Usage Statistics", // Table header
		"Input tokens",
		"100",
		"Output tokens",
		"50",
		"Cache read tokens",
		"25",
		"Cache creation tokens",
		"10",
		"Cache creation (ephemeral_5m_input_tokens)",
		"5",
		"Server tool use (web_search_requests)",
		"3",
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

	f := New(Config{Output: &output, SkillName: "example-skill"})
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Should contain starting message with skill name
	if !strings.Contains(result, "Starting example-skill") {
		t.Errorf("Output should contain 'Starting example-skill', got: %s", result)
	}
}

func TestFormat_ToolCall(t *testing.T) {
	// Tool calls are registered but not printed until result arrives
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_123","name":"Glob","input":{"pattern":"**/*.md"}}]}}`

	var output bytes.Buffer

	f := New(Config{Output: &output})
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// No output until tool result arrives
	if result != "" {
		t.Errorf("Output should be empty until tool result, got: %s", result)
	}
}

func TestFormat_ToolResult(t *testing.T) {
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_123","name":"Read","input":{"file_path":"/home/user/test.go"}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_123","content":"package main\n\nfunc main() {}"}]}}`

	var output bytes.Buffer

	f := New(Config{Output: &output})
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Should contain tool operation with target filename
	if !strings.Contains(result, "Read") {
		t.Errorf("Output should contain 'Read', got: %s", result)
	}

	if !strings.Contains(result, "test.go") {
		t.Errorf("Output should contain target filename 'test.go', got: %s", result)
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

	f := New(Config{Output: &output, SkillName: "workflow-skill"})
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Check for all expected elements in new format
	expectedStrings := []string{
		"Starting workflow-skill",
		"Glob",
		"Read",
		"I'll search for markdown files",
		"Found 3 markdown files",
		"Completed in",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result, expected) {
			t.Errorf("Output should contain '%s', got: %s", expected, result)
		}
	}
}

func TestFormat_PassthroughMode(t *testing.T) {
	// When user explicitly sets --output-format, we should passthrough without parsing
	input := `{"type":"system","subtype":"init"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Raw output"}]}}
{"type":"result","result":"Raw output","is_error":false}`

	var output bytes.Buffer

	// passthroughMode = true simulates user explicitly setting --output-format
	f := New(Config{Output: &output, PassthroughMode: true})
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// In passthrough mode, raw JSON should be output as-is
	if !strings.Contains(result, `"type":"system"`) {
		t.Errorf("Passthrough mode should output raw JSON, got: %s", result)
	}

	// Should NOT contain formatted output
	if strings.Contains(result, "Starting") {
		t.Errorf("Passthrough mode should not format output, got: %s", result)
	}
}

func TestFormat_VerboseWithoutPassthrough(t *testing.T) {
	// When user sets --verbose without --output-format, we should parse and show verbose TUI
	input := `{"type":"system","subtype":"init"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking","text":"Let me think..."}]}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Done thinking"}]}}
{"type":"result","result":"Done thinking","is_error":false}`

	var output bytes.Buffer

	// verbose = true, passthroughMode = false
	f := New(Config{Output: &output, Verbose: true, SkillName: "verbose-skill"})
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Should contain formatted output with thinking (in verbose mode)
	if !strings.Contains(result, "Starting verbose-skill") {
		t.Errorf("Verbose mode should format output, got: %s", result)
	}

	// Should contain thinking block in verbose mode
	if !strings.Contains(result, "Let me think") {
		t.Errorf("Verbose mode should show thinking blocks, got: %s", result)
	}

	// Should NOT be raw JSON
	if strings.Contains(result, `"type":"system"`) {
		t.Errorf("Verbose mode should not output raw JSON when passthroughMode is false, got: %s", result)
	}
}

func TestFormat_TodoWrite_NonVerbose(t *testing.T) {
	// TodoWrite should display as status lines even in non-verbose mode
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_123","name":"TodoWrite","input":{"todos":[{"content":"First task","status":"pending","activeForm":"Doing first task"},{"content":"Second task","status":"in_progress","activeForm":"Doing second task"},{"content":"Third task","status":"completed","activeForm":"Did third task"}]}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_123","content":"Todos have been modified successfully"}]}}`

	var output bytes.Buffer

	f := New(Config{Output: &output, Verbose: false})
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Should show pending task with empty checkbox (dimmed)
	if !strings.Contains(result, "☐ First task") {
		t.Errorf("Output should contain pending task with ☐, got: %s", result)
	}

	// Should show in-progress task with empty checkbox (prominent)
	if !strings.Contains(result, "☐ Second task") {
		t.Errorf("Output should contain in-progress task with ☐, got: %s", result)
	}

	// Should show most recently completed task with ☒
	if !strings.Contains(result, "☒ Third task") {
		t.Errorf("Output should contain completed task with ☒, got: %s", result)
	}

	// Should NOT show traditional tool format
	if strings.Contains(result, "TodoWrite") {
		t.Errorf("Output should not show TodoWrite as a tool, got: %s", result)
	}
}

func TestFormat_TodoWrite_Verbose(t *testing.T) {
	// TodoWrite should display as status lines in verbose mode too
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_123","name":"TodoWrite","input":{"todos":[{"content":"First task","status":"pending","activeForm":"Doing first task"},{"content":"Second task","status":"in_progress","activeForm":"Doing second task"},{"content":"Third task","status":"completed","activeForm":"Did third task"}]}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_123","content":"Todos have been modified successfully"}]}}`

	var output bytes.Buffer

	f := New(Config{Output: &output, Verbose: true})
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Should show pending task with empty checkbox (dimmed)
	if !strings.Contains(result, "☐ First task") {
		t.Errorf("Output should contain pending task with ☐, got: %s", result)
	}

	// Should show in-progress task with empty checkbox (prominent)
	if !strings.Contains(result, "☐ Second task") {
		t.Errorf("Output should contain in-progress task with ☐, got: %s", result)
	}

	// Should show most recently completed task with ☒
	if !strings.Contains(result, "☒ Third task") {
		t.Errorf("Output should contain completed task with ☒, got: %s", result)
	}

	// Should NOT show traditional tool format
	if strings.Contains(result, "TodoWrite") {
		t.Errorf("Output should not show TodoWrite as a tool, got: %s", result)
	}
}

func TestFormat_TodoWrite_AllCompleted(t *testing.T) {
	// When all todos are completed, only the last completed one should be shown
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_123","name":"TodoWrite","input":{"todos":[{"content":"First task","status":"completed","activeForm":"Did first task"},{"content":"Second task","status":"completed","activeForm":"Did second task"}]}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_123","content":"Todos have been modified successfully"}]}}`

	var output bytes.Buffer

	f := New(Config{Output: &output, Verbose: false})
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Should show only the last completed task with ☒
	if !strings.Contains(result, "☒ Second task") {
		t.Errorf("Output should contain last completed task with ☒, got: %s", result)
	}

	// Should NOT show first completed task
	if strings.Contains(result, "First task") {
		t.Errorf("Output should not contain first completed task, got: %s", result)
	}
}

func TestFormat_TodoWrite_OnlyPending(t *testing.T) {
	// Only pending todos should show with empty checkboxes
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_123","name":"TodoWrite","input":{"todos":[{"content":"Task one","status":"pending","activeForm":"Doing task one"},{"content":"Task two","status":"pending","activeForm":"Doing task two"}]}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_123","content":"Todos have been modified successfully"}]}}`

	var output bytes.Buffer

	f := New(Config{Output: &output, Verbose: false})
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Should show both pending tasks with empty checkboxes (dimmed)
	if !strings.Contains(result, "☐ Task one") {
		t.Errorf("Output should contain first pending task, got: %s", result)
	}

	if !strings.Contains(result, "☐ Task two") {
		t.Errorf("Output should contain second pending task, got: %s", result)
	}

	// Should not have completed checkboxes
	if strings.Contains(result, "☒") {
		t.Errorf("Output should not contain completed marker for pending tasks, got: %s", result)
	}
}

func TestStripSystemReminders_Basic(t *testing.T) {
	input := "Some text <system-reminder>This is a reminder</system-reminder> more text"
	expected := "Some text \n more text"
	result := stripSystemReminders(input)

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestStripSystemReminders_Multiple(t *testing.T) {
	input := "First <system-reminder>Reminder 1</system-reminder> middle <system-reminder>Reminder 2</system-reminder> last"
	expected := "First \n middle \n last"
	result := stripSystemReminders(input)

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestStripSystemReminders_UnclosedTag(t *testing.T) {
	input := "Some text <system-reminder>This reminder has no closing tag"
	expected := "Some text "
	result := stripSystemReminders(input)

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestStripSystemReminders_NestedContent(t *testing.T) {
	// Test with nested-looking content (not actual nested tags, but content that looks like it)
	input := "Text <system-reminder>Reminder with <tags> inside</system-reminder> end"
	expected := "Text \n end"
	result := stripSystemReminders(input)

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestStripSystemReminders_WithNewlines(t *testing.T) {
	input := "First line\n<system-reminder>\nMultiline reminder\nwith content\n</system-reminder>\nLast line"
	expected := "First line\nLast line"
	result := stripSystemReminders(input)

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestStripSystemReminders_NoReminders(t *testing.T) {
	input := "Just some plain text without any reminders"
	expected := input
	result := stripSystemReminders(input)

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestStripSystemReminders_EmptyString(t *testing.T) {
	input := ""
	expected := ""
	result := stripSystemReminders(input)

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestStripSystemReminders_OnlyReminder(t *testing.T) {
	input := "<system-reminder>Only a reminder</system-reminder>"
	expected := ""
	result := stripSystemReminders(input)

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFormat_TaskCreate_NonVerbose(t *testing.T) {
	// TaskCreate should display task subject and description
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_123","name":"TaskCreate","input":{"subject":"Implement authentication","description":"Add login/logout with JWT","activeForm":"Implementing auth","metadata":{"type":"feature","priority":"high"}}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_123","content":"Task created with id: 1"}]}}`

	var output bytes.Buffer

	f := New(Config{Output: &output, Verbose: false})
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Should show subject with checkbox
	if !strings.Contains(result, "☐ Implement authentication") {
		t.Errorf("Output should contain task subject with checkbox, got: %s", result)
	}

	// Should show description
	if !strings.Contains(result, "Add login/logout with JWT") {
		t.Errorf("Output should contain task description, got: %s", result)
	}

	// Should show metadata
	if !strings.Contains(result, "Metadata:") {
		t.Errorf("Output should contain metadata header, got: %s", result)
	}

	// Should NOT show traditional tool format
	if strings.Contains(result, "✓ TaskCreate") {
		t.Errorf("Output should not show TaskCreate as a traditional tool, got: %s", result)
	}
}

func TestFormat_TaskCreate_Verbose(t *testing.T) {
	// TaskCreate in verbose mode should show task details in a box
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_123","name":"TaskCreate","input":{"subject":"Fix memory leak","description":"The LRU cache is not evicting old entries properly","metadata":{"type":"bug","priority":"critical"}}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_123","content":"Task created with id: 2"}]}}`

	var output bytes.Buffer

	f := New(Config{Output: &output, Verbose: true})
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Should show subject with emoji
	if !strings.Contains(result, "📋 Fix memory leak") {
		t.Errorf("Output should contain task subject with 📋 emoji, got: %s", result)
	}

	// Should show description
	if !strings.Contains(result, "The LRU cache is not evicting old entries properly") {
		t.Errorf("Output should contain task description, got: %s", result)
	}

	// Should show metadata
	if !strings.Contains(result, "Metadata:") {
		t.Errorf("Output should contain metadata header, got: %s", result)
	}
}

func TestFormat_TaskUpdate_StatusChange(t *testing.T) {
	// TaskUpdate should show status change with visual indicator
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_456","name":"TaskUpdate","input":{"taskId":"1","status":"in_progress"}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_456","content":"Task updated"}]}}`

	var output bytes.Buffer

	f := New(Config{Output: &output, Verbose: false})
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Should show status change with half-filled indicator for in_progress
	if !strings.Contains(result, "◐") {
		t.Errorf("Output should contain half-filled indicator for in_progress, got: %s", result)
	}

	if !strings.Contains(result, "in_progress") {
		t.Errorf("Output should contain status 'in_progress', got: %s", result)
	}

	// Should show task ID
	if !strings.Contains(result, "1") {
		t.Errorf("Output should contain task ID '1', got: %s", result)
	}
}

func TestFormat_TaskUpdate_Completed(t *testing.T) {
	// TaskUpdate to completed should show filled indicator
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_789","name":"TaskUpdate","input":{"taskId":"2","status":"completed"}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_789","content":"Task updated"}]}}`

	var output bytes.Buffer

	f := New(Config{Output: &output, Verbose: false})
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Should show filled indicator for completed
	if !strings.Contains(result, "●") {
		t.Errorf("Output should contain filled indicator for completed, got: %s", result)
	}

	if !strings.Contains(result, "completed") {
		t.Errorf("Output should contain status 'completed', got: %s", result)
	}
}

func TestFormat_TaskList(t *testing.T) {
	// TaskList should show list of tasks
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_abc","name":"TaskList","input":{}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_abc","content":"1. Implement auth (in_progress)\n2. Fix memory leak (pending)\n3. Update docs (completed)"}]}}`

	var output bytes.Buffer

	f := New(Config{Output: &output, Verbose: false})
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Should show TaskList confirmation
	if !strings.Contains(result, "TaskList") {
		t.Errorf("Output should contain 'TaskList', got: %s", result)
	}
}

func TestFormat_TaskGet(t *testing.T) {
	// TaskGet should show task details
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_def","name":"TaskGet","input":{"taskId":"1"}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_def","content":"Task 1: Implement authentication\nStatus: in_progress\nDescription: Add login/logout with JWT"}]}}`

	var output bytes.Buffer

	f := New(Config{Output: &output, Verbose: false})
	err := f.Format(strings.NewReader(input))

	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := output.String()

	// Should show TaskGet with task ID
	if !strings.Contains(result, "TaskGet") {
		t.Errorf("Output should contain 'TaskGet', got: %s", result)
	}

	if !strings.Contains(result, "1") {
		t.Errorf("Output should contain task ID '1', got: %s", result)
	}
}

func TestParser_ExtractTarget_TaskCreate(t *testing.T) {
	parser := NewStreamParser("test", "", false)

	target := parser.extractTarget("TaskCreate", map[string]any{
		"subject":     "My task subject",
		"description": "Task description",
	})

	if target != "My task subject" {
		t.Errorf("Expected 'My task subject', got '%s'", target)
	}
}

func TestParser_ExtractTarget_TaskUpdate(t *testing.T) {
	parser := NewStreamParser("test", "", false)

	// With status
	target := parser.extractTarget("TaskUpdate", map[string]any{
		"taskId": "42",
		"status": "completed",
	})

	if target != "42 → completed" {
		t.Errorf("Expected '42 → completed', got '%s'", target)
	}

	// Without status
	target = parser.extractTarget("TaskUpdate", map[string]any{
		"taskId": "42",
	})

	if target != "42" {
		t.Errorf("Expected '42', got '%s'", target)
	}
}

func TestParser_ExtractTarget_TaskGet(t *testing.T) {
	parser := NewStreamParser("test", "", false)

	target := parser.extractTarget("TaskGet", map[string]any{
		"taskId": "123",
	})

	if target != "123" {
		t.Errorf("Expected '123', got '%s'", target)
	}
}

func TestParser_ExtractTarget_TaskList(t *testing.T) {
	parser := NewStreamParser("test", "", false)

	target := parser.extractTarget("TaskList", map[string]any{})

	if target != "" {
		t.Errorf("Expected empty string, got '%s'", target)
	}
}
