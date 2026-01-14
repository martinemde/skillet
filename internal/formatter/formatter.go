package formatter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
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
	Type      string                 `json:"type"`
	Text      string                 `json:"text,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Content   interface{}            `json:"content,omitempty"`
	ToolUseID string                 `json:"tool_use_id,omitempty"`
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
	ID      string
	Name    string
	Target  string // filename, command, or key parameter
	Status  string // "pending", "success", "error", "empty"
	Error   string
	Input   map[string]interface{}
	Result  interface{}
}

// Styles for terminal output
var (
	successIcon  = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).SetString("✓")
	errorIcon    = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).SetString("✗")
	emptyIcon    = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).SetString("○")
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	separator    = dimStyle.Render("───────────────────────────────────────────")

	// Verbose content styles
	thinkingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")). // Cyan
			Italic(true).
			MarginLeft(2)

	outputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")). // Dim
			MarginLeft(2).
			PaddingLeft(1)

	commentaryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("7")). // Light gray
			MarginLeft(2)
)

// Formatter formats stream-json output from Claude CLI
type Formatter struct {
	output          io.Writer
	verbose         bool
	showUsage       bool
	outputFormat    string
	toolCount       int
	tools           []ToolOperation
	startTime       time.Time
	toolCallMap     map[string]int // Maps tool_use_id to index in tools slice
	printedToolsHdr bool           // Track if we've printed "Tools:" header
}

// New creates a new Formatter
func New(output io.Writer, verbose, showUsage bool, outputFormat string) *Formatter {
	return &Formatter{
		output:          output,
		verbose:         verbose,
		showUsage:       showUsage,
		outputFormat:    outputFormat,
		toolCount:       0,
		tools:           make([]ToolOperation, 0),
		startTime:       time.Now(),
		toolCallMap:     make(map[string]int),
		printedToolsHdr: false,
	}
}

// Format reads stream-json input and formats it
func (f *Formatter) Format(input io.Reader) error {
	// In verbose mode with stream-json output, passthrough raw JSON directly
	if f.verbose && (f.outputFormat == "stream-json" || f.outputFormat == "json") {
		_, err := io.Copy(f.output, input)
		return err
	}

	scanner := bufio.NewScanner(input)
	var textBuilder strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			if f.verbose {
				fmt.Fprintf(os.Stderr, "DEBU Failed to parse JSON: %v\n", err)
				fmt.Fprintf(os.Stderr, "DEBU stream data=%s\n", line)
			}
			continue
		}

		// In verbose mode, print the raw JSON stream to stderr
		if f.verbose {
			fmt.Fprintf(os.Stderr, "DEBU stream data=%s\n", line)
		}

		switch msg.Type {
		case "system":
			if msg.Subtype == "init" {
				fmt.Fprintln(f.output, successIcon.String()+" Session started")
			}

		case "assistant":
			if msg.Message != nil {
				for _, content := range msg.Message.Content {
					switch content.Type {
					case "thinking":
						// Display thinking blocks in verbose mode
						if f.verbose && content.Text != "" {
							fmt.Fprintln(f.output, thinkingStyle.Render("💭 "+content.Text))
							fmt.Fprintln(f.output)
						}

					case "text":
						if content.Text != "" {
							// In verbose mode, stream commentary as it comes in
							if f.verbose {
								fmt.Fprintln(f.output, commentaryStyle.Render(content.Text))
								fmt.Fprintln(f.output)
							}
							textBuilder.WriteString(content.Text)
						}

					case "tool_use":
						f.toolCount++
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

		case "user":
			// Handle tool results
			if msg.Message != nil {
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

		case "result":
			// Print separator if we printed any tools
			if f.printedToolsHdr {
				fmt.Fprintln(f.output)
				fmt.Fprintln(f.output, separator)
				fmt.Fprintln(f.output)
			}

			// Print only the final result
			if msg.Result != "" {
				fmt.Fprintln(f.output, msg.Result)
			}

			// Print usage information if requested
			if f.showUsage && msg.Usage != nil {
				f.printUsage(msg.Usage)
			}

			// Print completion status
			elapsed := time.Since(f.startTime)
			fmt.Fprintln(f.output)
			if msg.IsError {
				fmt.Fprintln(f.output, errorIcon.String()+" Failed")
			} else {
				fmt.Fprintf(f.output, "%s Completed in %.1fs\n", successIcon.String(), elapsed.Seconds())
			}
		}
	}

	return scanner.Err()
}

// extractTarget extracts the key parameter from tool input
func (f *Formatter) extractTarget(toolName string, input map[string]interface{}) string {
	switch toolName {
	case "Read", "Write", "Edit":
		if path, ok := input["file_path"].(string); ok {
			// Get just the filename from the path
			parts := strings.Split(path, "/")
			return parts[len(parts)-1]
		}
	case "Bash":
		if cmd, ok := input["command"].(string); ok {
			// Get first word of command
			parts := strings.Fields(cmd)
			if len(parts) > 0 {
				return parts[0]
			}
		}
	case "Grep":
		if pattern, ok := input["pattern"].(string); ok {
			if len(pattern) > 20 {
				return pattern[:17] + "..."
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
func (f *Formatter) determineStatus(content interface{}) string {
	// Check if it's an error
	switch v := content.(type) {
	case string:
		if strings.Contains(v, "<tool_use_error>") || strings.Contains(v, "error") {
			return "error"
		}
		if v == "" || v == "[]" {
			return "empty"
		}
	case []interface{}:
		if len(v) == 0 {
			return "empty"
		}
		// Check array content for errors
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
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

// extractError extracts error message from tool result
func (f *Formatter) extractError(content interface{}) string {
	switch v := content.(type) {
	case string:
		// Extract text between <tool_use_error> tags
		if start := strings.Index(v, "<tool_use_error>"); start != -1 {
			if end := strings.Index(v, "</tool_use_error>"); end != -1 {
				return v[start+16 : end]
			}
		}
		// Return first 100 chars if it contains "error"
		if strings.Contains(strings.ToLower(v), "error") {
			if len(v) > 100 {
				return v[:97] + "..."
			}
			return v
		}
	case []interface{}:
		// Check array content for errors
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if text, ok := m["text"].(string); ok {
					if strings.Contains(text, "<tool_use_error>") {
						if start := strings.Index(text, "<tool_use_error>"); start != -1 {
							if end := strings.Index(text, "</tool_use_error>"); end != -1 {
								return text[start+16 : end]
							}
						}
					}
				}
			}
		}
	}
	return "unknown error"
}

// printToolOperation prints a single tool operation as it completes
func (f *Formatter) printToolOperation(tool ToolOperation) {
	// Track that we've printed at least one tool
	f.printedToolsHdr = true

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
	fmt.Fprintln(f.output, line)

	// In verbose mode, show details
	if f.verbose {
		f.printToolDetails(tool)
	}
}

// printToolDetails prints detailed tool information in verbose mode
func (f *Formatter) printToolDetails(tool ToolOperation) {
	// Show tool-specific output based on the tool type
	switch tool.Name {
	case "Read":
		f.printReadOutput(tool)
	case "Write", "Edit":
		f.printWriteOutput(tool)
	case "Bash":
		f.printBashOutput(tool)
	case "Grep", "Glob":
		f.printSearchOutput(tool)
	default:
		// For other tools, show basic input/output
		f.printGenericToolOutput(tool)
	}

	fmt.Fprintln(f.output) // Add spacing between tools in verbose mode
}

// printReadOutput shows file contents for Read operations
func (f *Formatter) printReadOutput(tool ToolOperation) {
	if tool.Result == nil {
		return
	}

	// Extract file contents from result
	resultStr := f.extractResultText(tool.Result)
	if resultStr != "" {
		// Show first 20 lines or full content if shorter
		lines := strings.Split(resultStr, "\n")
		maxLines := 20
		if len(lines) > maxLines {
			content := strings.Join(lines[:maxLines], "\n")
			fmt.Fprintln(f.output, outputStyle.Render(content))
			fmt.Fprintln(f.output, dimStyle.Render(fmt.Sprintf("  ... (%d more lines)", len(lines)-maxLines)))
		} else {
			fmt.Fprintln(f.output, outputStyle.Render(resultStr))
		}
	}
}

// printWriteOutput shows confirmation for Write/Edit operations
func (f *Formatter) printWriteOutput(tool ToolOperation) {
	if tool.Status == "success" {
		if filePath, ok := tool.Input["file_path"].(string); ok {
			fmt.Fprintln(f.output, dimStyle.Render(fmt.Sprintf("  → wrote to %s", filePath)))
		}
	}
}

// printBashOutput shows command output for Bash operations
func (f *Formatter) printBashOutput(tool ToolOperation) {
	if tool.Result == nil {
		return
	}

	resultStr := f.extractResultText(tool.Result)
	if resultStr != "" {
		// Show command output with light styling
		lines := strings.Split(resultStr, "\n")
		maxLines := 30
		if len(lines) > maxLines {
			content := strings.Join(lines[:maxLines], "\n")
			fmt.Fprintln(f.output, outputStyle.Render(content))
			fmt.Fprintln(f.output, dimStyle.Render(fmt.Sprintf("  ... (%d more lines)", len(lines)-maxLines)))
		} else {
			fmt.Fprintln(f.output, outputStyle.Render(resultStr))
		}
	}
}

// printSearchOutput shows search results for Grep/Glob operations
func (f *Formatter) printSearchOutput(tool ToolOperation) {
	if tool.Result == nil {
		return
	}

	resultStr := f.extractResultText(tool.Result)
	if resultStr != "" {
		lines := strings.Split(resultStr, "\n")
		maxLines := 15
		if len(lines) > maxLines {
			content := strings.Join(lines[:maxLines], "\n")
			fmt.Fprintln(f.output, outputStyle.Render(content))
			fmt.Fprintln(f.output, dimStyle.Render(fmt.Sprintf("  ... (%d more results)", len(lines)-maxLines)))
		} else {
			fmt.Fprintln(f.output, outputStyle.Render(resultStr))
		}
	}
}

// printGenericToolOutput shows basic input/output for other tools
func (f *Formatter) printGenericToolOutput(tool ToolOperation) {
	// Show key input parameters
	if len(tool.Input) > 0 {
		for k, v := range tool.Input {
			vStr := fmt.Sprintf("%v", v)
			if len(vStr) > 100 {
				vStr = vStr[:97] + "..."
			}
			fmt.Fprintln(f.output, dimStyle.Render(fmt.Sprintf("  → %s: %s", k, vStr)))
		}
	}

	// Show result summary
	if tool.Result != nil && tool.Status != "error" {
		resultStr := f.extractResultText(tool.Result)
		if resultStr != "" && len(resultStr) < 200 {
			fmt.Fprintln(f.output, dimStyle.Render(fmt.Sprintf("  → %s", resultStr)))
		}
	}
}

// extractResultText extracts text from a tool result
func (f *Formatter) extractResultText(result interface{}) string {
	switch v := result.(type) {
	case string:
		return v
	case []interface{}:
		// Handle array of content blocks
		var builder strings.Builder
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if text, ok := m["text"].(string); ok {
					builder.WriteString(text)
				}
			}
		}
		return builder.String()
	default:
		return fmt.Sprintf("%v", result)
	}
}

// printUsage prints token usage information
func (f *Formatter) printUsage(usage *Usage) {
	_, _ = fmt.Fprintln(f.output, "\n--- Usage Statistics ---")
	_, _ = fmt.Fprintf(f.output, "Input tokens: %d\n", usage.InputTokens)
	_, _ = fmt.Fprintf(f.output, "Output tokens: %d\n", usage.OutputTokens)

	if usage.CacheReadInputTokens > 0 {
		_, _ = fmt.Fprintf(f.output, "Cache read tokens: %d\n", usage.CacheReadInputTokens)
	}

	if usage.CacheCreationInputTokens > 0 {
		_, _ = fmt.Fprintf(f.output, "Cache creation tokens: %d\n", usage.CacheCreationInputTokens)
	}

	if usage.CacheCreation != nil {
		for k, v := range usage.CacheCreation {
			if v > 0 {
				_, _ = fmt.Fprintf(f.output, "Cache creation (%s): %d\n", k, v)
			}
		}
	}

	if usage.ServerToolUse != nil {
		for k, v := range usage.ServerToolUse {
			if v > 0 {
				_, _ = fmt.Fprintf(f.output, "Server tool use (%s): %d\n", k, v)
			}
		}
	}

	_, _ = fmt.Fprintln(f.output, "------------------------")
}
