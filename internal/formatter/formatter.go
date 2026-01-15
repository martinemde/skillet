package formatter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// Message represents different types of messages in the stream
type Message struct {
	Type    string          `json:"type"`
	Message *MessageContent `json:"message,omitempty"`
	Result  string          `json:"result,omitempty"`
	Subtype string          `json:"subtype,omitempty"`
	IsError bool            `json:"is_error,omitempty"`
	Usage   *Usage          `json:"usage,omitempty"`
}

// MessageContent represents the content of an assistant message
type MessageContent struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

// Content represents a piece of content in a message
type Content struct {
	Type      string         `json:"type"`
	Text      string         `json:"text,omitempty"`
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
	Content   any            `json:"content,omitempty"`
	ToolUseID string         `json:"tool_use_id,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	InputTokens              int            `json:"input_tokens"`
	OutputTokens             int            `json:"output_tokens"`
	CacheReadInputTokens     int            `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int            `json:"cache_creation_input_tokens"`
	CacheCreation            map[string]int `json:"cache_creation,omitempty"`
	ServerToolUse            map[string]int `json:"server_tool_use,omitempty"`
}

// ToolOperation represents a tool call and its result
type ToolOperation struct {
	ID     string
	Name   string
	Target string // filename, command, or key parameter
	Status string // "pending", "success", "error", "empty"
	Error  string
	Input  map[string]any
	Result any
}

// Output truncation limits
const (
	maxPatternDisplayLength = 20
	maxErrorDisplayLength   = 100
	maxGenericOutputLength  = 200
	maxReadOutputLines      = 20
	maxBashOutputLines      = 30
	maxSearchOutputLines    = 15
)

// Styles for terminal output
var (
	successIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).SetString("✓")
	errorIcon   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).SetString("✗")
	emptyIcon   = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).SetString("○")
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	// Verbose content styles
	thinkingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")). // Cyan
			Italic(true).
			MarginLeft(2)

	// Tool detail box style for verbose output
	toolBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")). // Dim border
			Padding(0, 1).
			MarginLeft(2).
			MarginTop(0).
			MarginBottom(1)
)

// Config holds configuration options for the Formatter
type Config struct {
	Output          io.Writer
	Verbose         bool
	Debug           bool // If true, print raw JSON lines to stderr
	ShowUsage       bool
	PassthroughMode bool   // If true, stream output directly without parsing
	SkillName       string // Name of the skill being executed
}

// Formatter formats stream-json output from Claude CLI
type Formatter struct {
	output          io.Writer
	verbose         bool
	debug           bool // If true, print raw JSON lines to stderr
	showUsage       bool
	passthroughMode bool   // If true, stream output directly without parsing
	skillName       string // Name of the skill being executed
	tools           []ToolOperation
	startTime       time.Time
	toolCallMap     map[string]int // Maps tool_use_id to index in tools slice
	mdRenderer      *glamour.TermRenderer
}

// New creates a new Formatter with the given configuration
func New(cfg Config) *Formatter {
	// Initialize glamour markdown renderer with a nice style
	// Use "auto" style which adapts to terminal background (light/dark)
	mdRenderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0), // No wrapping, let terminal handle it
	)
	if err != nil {
		// Fallback to nil renderer if initialization fails
		mdRenderer = nil
	}

	return &Formatter{
		output:          cfg.Output,
		verbose:         cfg.Verbose,
		debug:           cfg.Debug,
		showUsage:       cfg.ShowUsage,
		passthroughMode: cfg.PassthroughMode,
		skillName:       cfg.SkillName,
		startTime:       time.Now(),
		toolCallMap:     make(map[string]int),
		mdRenderer:      mdRenderer,
	}
}

// Format reads stream-json input and formats it
func (f *Formatter) Format(input io.Reader) error {
	// If user explicitly set --output-format, passthrough raw output directly
	if f.passthroughMode {
		_, err := io.Copy(f.output, input)
		return err
	}

	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		// In debug mode, print raw JSON to stderr
		if f.debug {
			fmt.Fprintf(os.Stderr, "DEBUG JSON: %s\n", line)
		}

		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// In verbose mode, show the error details to stderr
			if f.verbose {
				fmt.Fprintf(os.Stderr, "DEBUG Failed to parse JSON: %v\n", err)
				fmt.Fprintf(os.Stderr, "DEBUG stream data=%s\n", line)
			}
			// Skip invalid JSON - don't print to stdout
			continue
		}

		switch msg.Type {
		case "system":
			f.handleSystemMessage(msg)
		case "assistant":
			f.handleAssistantMessage(msg)
		case "user":
			f.handleUserMessage(msg)
		case "result":
			f.handleResultMessage(msg)
		}
	}

	return scanner.Err()
}

// handleSystemMessage processes system-level messages
func (f *Formatter) handleSystemMessage(msg Message) {
	if msg.Subtype == "init" {
		_, _ = fmt.Fprintf(f.output, "%s Starting %s\n", successIcon.String(), f.skillName)
	}
}

// handleAssistantMessage processes assistant response messages
func (f *Formatter) handleAssistantMessage(msg Message) {
	if msg.Message == nil {
		return
	}

	for _, content := range msg.Message.Content {
		switch content.Type {
		case "thinking":
			// Display thinking blocks in verbose mode
			if f.verbose && content.Text != "" {
				_, _ = fmt.Fprintln(f.output, thinkingStyle.Render("💭 "+content.Text))
				_, _ = fmt.Fprintln(f.output)
			}

		case "text":
			if content.Text != "" {
				// In verbose mode, stream commentary with markdown formatting
				if f.verbose {
					rendered := f.renderMarkdown(content.Text)
					_, _ = fmt.Fprintln(f.output, rendered)
				}
			}

		case "tool_use":
			// Create a new tool operation
			op := ToolOperation{
				ID:     content.ID,
				Name:   content.Name,
				Target: f.extractTarget(content.Name, content.Input),
				Status: "pending",
				Input:  content.Input,
			}
			f.toolCallMap[content.ID] = len(f.tools)
			f.tools = append(f.tools, op)
		}
	}
}

// handleUserMessage processes user messages (typically tool results)
func (f *Formatter) handleUserMessage(msg Message) {
	if msg.Message == nil {
		return
	}

	for _, content := range msg.Message.Content {
		if content.Type == "tool_result" && content.ToolUseID != "" {
			if idx, ok := f.toolCallMap[content.ToolUseID]; ok {
				f.tools[idx].Result = content.Content
				f.tools[idx].Status = f.determineStatus(content.Content)
				if f.tools[idx].Status == "error" {
					f.tools[idx].Error = f.extractError(content.Content)
				}
				// Immediately print this tool operation
				f.printToolOperation(f.tools[idx])
			}
		}
	}
}

// handleResultMessage processes final result messages
func (f *Formatter) handleResultMessage(msg Message) {
	// Print only the final result (skip in verbose mode since we already streamed it)
	if !f.verbose && msg.Result != "" {
		rendered := f.renderMarkdown(msg.Result)
		_, _ = fmt.Fprintln(f.output, rendered)
	}

	// Print usage information if requested
	if f.showUsage && msg.Usage != nil {
		f.printUsage(msg.Usage)
	}

	// Print completion status
	elapsed := time.Since(f.startTime)
	_, _ = fmt.Fprintln(f.output)
	if msg.IsError {
		_, _ = fmt.Fprintln(f.output, errorIcon.String()+" Failed")
	} else {
		_, _ = fmt.Fprintf(f.output, "%s Completed in %.1fs\n", successIcon.String(), elapsed.Seconds())
	}
}

// extractTarget extracts the key parameter from tool input
func (f *Formatter) extractTarget(toolName string, input map[string]any) string {
	switch toolName {
	case "Read", "Write", "Edit":
		if path, ok := input["file_path"].(string); ok {
			// Get just the filename from the path
			parts := strings.Split(path, "/")
			return parts[len(parts)-1]
		}
	case "Bash":
		// Prefer description if available
		if desc, ok := input["description"].(string); ok {
			return desc
		}
		// Fallback to first word of command
		if cmd, ok := input["command"].(string); ok {
			parts := strings.Fields(cmd)
			if len(parts) > 0 {
				return parts[0]
			}
		}
	case "Grep":
		if pattern, ok := input["pattern"].(string); ok {
			if len(pattern) > maxPatternDisplayLength {
				return pattern[:maxPatternDisplayLength-3] + "..."
			}
			return pattern
		}
	case "Glob":
		if pattern, ok := input["pattern"].(string); ok {
			return pattern
		}
	}
	return ""
}

// determineStatus determines the status of a tool result
func (f *Formatter) determineStatus(content any) string {
	// Check if it's an error
	switch v := content.(type) {
	case string:
		if strings.Contains(v, "<tool_use_error>") || strings.Contains(v, "error") {
			return "error"
		}
		if v == "" || v == "[]" {
			return "empty"
		}
	case []any:
		if len(v) == 0 {
			return "empty"
		}
		// Check array content for errors
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				if text, ok := m["text"].(string); ok {
					if strings.Contains(text, "<tool_use_error>") {
						return "error"
					}
				}
			}
		}
	}
	return "success"
}

// extractErrorFromText extracts error message from a string
func extractErrorFromText(text string) string {
	// Extract text between <tool_use_error> tags
	if start := strings.Index(text, "<tool_use_error>"); start != -1 {
		if end := strings.Index(text, "</tool_use_error>"); end != -1 {
			return text[start+16 : end]
		}
	}
	// Return truncated text if it contains "error"
	if strings.Contains(strings.ToLower(text), "error") {
		if len(text) > maxErrorDisplayLength {
			return text[:maxErrorDisplayLength-3] + "..."
		}
		return text
	}
	return ""
}

// extractError extracts error message from tool result
func (f *Formatter) extractError(content any) string {
	switch v := content.(type) {
	case string:
		if err := extractErrorFromText(v); err != "" {
			return err
		}
	case []any:
		// Check array content for errors
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				if text, ok := m["text"].(string); ok {
					if err := extractErrorFromText(text); err != "" {
						return err
					}
				}
			}
		}
	}
	return "unknown error"
}

// printToolOperation prints a single tool operation as it completes
func (f *Formatter) printToolOperation(tool ToolOperation) {
	// Special handling for TodoWrite - always show todos as status lines
	if tool.Name == "TodoWrite" {
		f.printTodoStatusLines(tool)
		return
	}

	// Choose icon based on status
	icon := successIcon
	switch tool.Status {
	case "error":
		icon = errorIcon
	case "empty":
		icon = emptyIcon
	}

	// Format tool line (no indent)
	line := fmt.Sprintf("%s %s", icon.String(), tool.Name)
	if tool.Target != "" {
		line += " " + tool.Target
	}
	if tool.Status == "error" && tool.Error != "" {
		line += dimStyle.Render(fmt.Sprintf(" (%s)", tool.Error))
	}
	_, _ = fmt.Fprintln(f.output, line)

	// In verbose mode, show details
	if f.verbose {
		f.printToolDetails(tool)
	}
}

// printToolDetails prints detailed tool information in verbose mode
func (f *Formatter) printToolDetails(tool ToolOperation) {
	// Collect output in a buffer to wrap it in a box
	var content strings.Builder

	// Show tool-specific output based on the tool type
	switch tool.Name {
	case "Read":
		f.buildReadOutput(&content, tool)
	case "Write", "Edit":
		f.buildWriteOutput(&content, tool)
	case "Bash":
		f.buildBashOutput(&content, tool)
	case "Grep", "Glob":
		f.buildSearchOutput(&content, tool)
	case "TodoWrite":
		f.buildTodoOutput(&content, tool)
	default:
		// For other tools, show basic input/output
		f.buildGenericToolOutput(&content, tool)
	}

	// Only print if there's content
	if content.Len() > 0 {
		// Wrap in styled box
		boxed := toolBoxStyle.Render(strings.TrimRight(content.String(), "\n"))
		_, _ = fmt.Fprintln(f.output, boxed)
	}
}

// buildTruncatedLines writes text with line truncation to a builder
func (f *Formatter) buildTruncatedLines(w *strings.Builder, text string, maxLines int, label string) {
	if text == "" {
		return
	}

	text = strings.TrimRight(text, "\n\r")
	lines := strings.Split(text, "\n")

	if len(lines) > maxLines {
		content := strings.Join(lines[:maxLines], "\n")
		fmt.Fprintln(w, content)
		fmt.Fprintln(w, dimStyle.Render(fmt.Sprintf("... (%d more %s)", len(lines)-maxLines, label)))
	} else {
		fmt.Fprintln(w, text)
	}
}

// buildReadOutput writes file contents for Read operations
func (f *Formatter) buildReadOutput(w *strings.Builder, tool ToolOperation) {
	if tool.Result == nil {
		return
	}
	resultStr := f.extractResultText(tool.Result)
	f.buildTruncatedLines(w, resultStr, maxReadOutputLines, "lines")
}

// buildWriteOutput writes confirmation for Write/Edit operations
func (f *Formatter) buildWriteOutput(w *strings.Builder, tool ToolOperation) {
	if tool.Status == "success" {
		if filePath, ok := tool.Input["file_path"].(string); ok {
			fmt.Fprintln(w, dimStyle.Render(fmt.Sprintf("→ wrote to %s", filePath)))
		}
	}
}

// buildBashOutput writes command output for Bash operations
func (f *Formatter) buildBashOutput(w *strings.Builder, tool ToolOperation) {
	// Show the command itself as a code block
	if cmd, ok := tool.Input["command"].(string); ok {
		cmdBlock := fmt.Sprintf("```sh\n$ %s\n```", cmd)
		rendered := f.renderMarkdown(cmdBlock)
		fmt.Fprint(w, rendered)
	}

	if tool.Result == nil {
		return
	}

	resultStr := f.extractResultText(tool.Result)
	if resultStr == "" {
		return
	}

	// Show command output in a code block for consistent styling
	resultStr = strings.TrimRight(resultStr, "\n\r")
	lines := strings.Split(resultStr, "\n")

	if len(lines) > maxBashOutputLines {
		content := strings.Join(lines[:maxBashOutputLines], "\n")
		resultBlock := fmt.Sprintf("```sh\n%s\n```", content)
		rendered := f.renderMarkdown(resultBlock)
		fmt.Fprint(w, rendered)
		fmt.Fprintln(w, dimStyle.Render(fmt.Sprintf("... (%d more lines)", len(lines)-maxBashOutputLines)))
	} else {
		resultBlock := fmt.Sprintf("```sh\n%s\n```", resultStr)
		rendered := f.renderMarkdown(resultBlock)
		fmt.Fprint(w, rendered)
	}
}

// buildSearchOutput writes search results for Grep/Glob operations
func (f *Formatter) buildSearchOutput(w *strings.Builder, tool ToolOperation) {
	if tool.Result == nil {
		return
	}
	resultStr := f.extractResultText(tool.Result)
	f.buildTruncatedLines(w, resultStr, maxSearchOutputLines, "results")
}

// printTodoStatusLines shows todos as individual status lines
func (f *Formatter) printTodoStatusLines(tool ToolOperation) {
	// Extract todos from the input
	if todos, ok := tool.Input["todos"].([]any); ok {
		for _, todoItem := range todos {
			if todo, ok := todoItem.(map[string]any); ok {
				content, _ := todo["content"].(string)
				status, _ := todo["status"].(string)

				if status == "completed" {
					// Hide completed items entirely
					continue
				} else if status == "in_progress" {
					// Use filled circle for in-progress
					_, _ = fmt.Fprintf(f.output, "⏺ %s\n", content)
				} else {
					// Use empty circle for pending
					_, _ = fmt.Fprintf(f.output, "○ %s\n", content)
				}
			}
		}
	}
}

// buildTodoOutput writes todo list changes
func (f *Formatter) buildTodoOutput(w *strings.Builder, tool ToolOperation) {
	// Extract todos from the input
	if todos, ok := tool.Input["todos"].([]any); ok {
		var todoLines []string
		for _, todoItem := range todos {
			if todo, ok := todoItem.(map[string]any); ok {
				content, _ := todo["content"].(string)
				status, _ := todo["status"].(string)

				// Use markdown checkbox format
				var checkbox string
				switch status {
				case "completed":
					checkbox = "- [x]"
				case "in_progress":
					checkbox = "- [○]" // Circle for in-progress
				default:
					checkbox = "- [ ]"
				}

				todoLines = append(todoLines, fmt.Sprintf("%s %s", checkbox, content))
			}
		}

		if len(todoLines) > 0 {
			todoText := strings.Join(todoLines, "\n")
			rendered := f.renderMarkdown(todoText)
			fmt.Fprint(w, rendered)
		}
	}
}

// buildGenericToolOutput writes basic input/output for other tools
func (f *Formatter) buildGenericToolOutput(w *strings.Builder, tool ToolOperation) {
	// Show key input parameters
	if len(tool.Input) > 0 {
		for k, v := range tool.Input {
			vStr := fmt.Sprintf("%v", v)
			if len(vStr) > maxErrorDisplayLength {
				vStr = vStr[:maxErrorDisplayLength-3] + "..."
			}
			fmt.Fprintln(w, dimStyle.Render(fmt.Sprintf("→ %s: %s", k, vStr)))
		}
	}

	// Show result summary
	if tool.Result != nil && tool.Status != "error" {
		resultStr := f.extractResultText(tool.Result)
		if resultStr != "" && len(resultStr) < maxGenericOutputLength {
			fmt.Fprintln(w, dimStyle.Render(fmt.Sprintf("→ %s", resultStr)))
		}
	}
}

// extractResultText extracts text from a tool result
func (f *Formatter) extractResultText(result any) string {
	switch v := result.(type) {
	case string:
		return stripSystemReminders(v)
	case []any:
		// Handle array of content blocks
		var builder strings.Builder
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				if text, ok := m["text"].(string); ok {
					builder.WriteString(text)
				}
			}
		}
		return stripSystemReminders(builder.String())
	default:
		return fmt.Sprintf("%v", result)
	}
}

// stripSystemReminders removes <system-reminder>...</system-reminder> tags and their content
func stripSystemReminders(text string) string {
	for {
		start := strings.Index(text, "<system-reminder>")
		if start == -1 {
			break
		}
		end := strings.Index(text[start:], "</system-reminder>")
		if end == -1 {
			// Unclosed tag, just remove from start to end
			text = text[:start]
			break
		}
		// Remove the entire tag including newlines around it
		end = start + end + len("</system-reminder>")

		// Clean up extra newlines: if there are newlines before and after, keep only one
		beforeStart := start
		afterEnd := end

		// Check for newlines before the tag
		for beforeStart > 0 && (text[beforeStart-1] == '\n' || text[beforeStart-1] == '\r') {
			beforeStart--
		}

		// Check for newlines after the tag
		for afterEnd < len(text) && (text[afterEnd] == '\n' || text[afterEnd] == '\r') {
			afterEnd++
		}

		// Remove the tag and surrounding whitespace, keep one newline if there was content before
		if beforeStart > 0 && afterEnd < len(text) {
			text = text[:beforeStart] + "\n" + text[afterEnd:]
		} else {
			text = text[:beforeStart] + text[afterEnd:]
		}
	}
	return text
}

// renderMarkdown renders markdown text using glamour, or returns plain text if unavailable
func (f *Formatter) renderMarkdown(text string) string {
	// If no renderer available, return plain text
	if f.mdRenderer == nil {
		return text
	}

	// Render the markdown
	rendered, err := f.mdRenderer.Render(text)
	if err != nil {
		// Fallback to plain text if rendering fails
		return text
	}

	// glamour adds leading/trailing newlines for formatting, trim them
	return strings.TrimSpace(rendered)
}

// printUsage prints token usage information in a styled table
func (f *Formatter) printUsage(usage *Usage) {
	// Build table rows
	rows := [][]string{
		{"Input tokens", fmt.Sprintf("%d", usage.InputTokens)},
		{"Output tokens", fmt.Sprintf("%d", usage.OutputTokens)},
	}

	if usage.CacheReadInputTokens > 0 {
		rows = append(rows, []string{"Cache read tokens", fmt.Sprintf("%d", usage.CacheReadInputTokens)})
	}

	if usage.CacheCreationInputTokens > 0 {
		rows = append(rows, []string{"Cache creation tokens", fmt.Sprintf("%d", usage.CacheCreationInputTokens)})
	}

	if usage.CacheCreation != nil {
		for k, v := range usage.CacheCreation {
			if v > 0 {
				rows = append(rows, []string{fmt.Sprintf("Cache creation (%s)", k), fmt.Sprintf("%d", v)})
			}
		}
	}

	if usage.ServerToolUse != nil {
		for k, v := range usage.ServerToolUse {
			if v > 0 {
				rows = append(rows, []string{fmt.Sprintf("Server tool use (%s)", k), fmt.Sprintf("%d", v)})
			}
		}
	}

	// Create styled table
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("8"))). // Dim border
		StyleFunc(func(row, col int) lipgloss.Style {
			if col == 0 {
				// Metric name column - dim style
				return dimStyle
			}
			// Value column - normal style
			return lipgloss.NewStyle()
		}).
		Headers("Usage Statistics", "Count").
		Rows(rows...)

	_, _ = fmt.Fprintln(f.output)
	_, _ = fmt.Fprintln(f.output, t)
}
