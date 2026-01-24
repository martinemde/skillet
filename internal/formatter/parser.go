package formatter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
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
	Role       string          `json:"role"`
	RawContent json.RawMessage `json:"content"`
	Content    []Content       `json:"-"` // Populated after parsing RawContent
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

// parseContent parses the RawContent field into the Content slice.
// Handles both string content (user messages) and array content (assistant messages).
func (m *MessageContent) parseContent() error {
	if m.RawContent == nil {
		return nil
	}

	// Try parsing as array first (most common for assistant messages)
	var contentArray []Content
	if err := json.Unmarshal(m.RawContent, &contentArray); err == nil {
		m.Content = contentArray
		return nil
	}

	// Try parsing as string (user messages in conversation logs)
	var contentStr string
	if err := json.Unmarshal(m.RawContent, &contentStr); err == nil {
		m.Content = []Content{{Type: "text", Text: contentStr}}
		return nil
	}

	// If neither works, leave Content empty
	return nil
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
)

// ClaudeStreamParser parses Claude JSONL stream into events
type ClaudeStreamParser struct {
	tools       []ToolOperation
	toolCallMap map[string]int
	startTime   time.Time
	skillName   string
	skillPath   string
	verbose     bool
}

// NewStreamParser creates a new stream parser
func NewStreamParser(skillName, skillPath string, verbose bool) *ClaudeStreamParser {
	return &ClaudeStreamParser{
		startTime:   time.Now(),
		toolCallMap: make(map[string]int),
		skillName:   skillName,
		skillPath:   skillPath,
		verbose:     verbose,
	}
}

// Parse reads JSONL input and emits StreamEvents
func (p *ClaudeStreamParser) Parse(input io.Reader) (<-chan StreamEvent, <-chan error) {
	events := make(chan StreamEvent, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errChan)

		scanner := bufio.NewScanner(input)
		// Increase buffer size to handle large JSONL lines
		const maxScannerBuffer = 1024 * 1024
		scanner.Buffer(make([]byte, 0, 64*1024), maxScannerBuffer)

		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "" {
				continue
			}

			var msg Message
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				// In verbose mode, show the error details to stderr
				if p.verbose {
					fmt.Fprintf(os.Stderr, "DEBUG Failed to parse JSON: %v\n", err)
					fmt.Fprintf(os.Stderr, "DEBUG stream data=%s\n", line)
				}
				// Skip invalid JSON - don't emit event
				continue
			}

			// Parse message content (handles both string and array formats)
			if msg.Message != nil {
				_ = msg.Message.parseContent()
			}

			// Handle different message types
			p.handleMessage(msg, events)
		}

		if err := scanner.Err(); err != nil {
			errChan <- err
		}
	}()

	return events, errChan
}

// handleMessage processes a message and emits appropriate events
func (p *ClaudeStreamParser) handleMessage(msg Message, events chan<- StreamEvent) {
	switch msg.Type {
	case "system":
		p.handleSystemMessage(msg, events)
	case "assistant":
		p.handleAssistantMessage(msg, events)
	case "user":
		p.handleUserMessage(msg, events)
	case "result":
		p.handleResultMessage(msg, events)
	}
}

// handleSystemMessage processes system-level messages
func (p *ClaudeStreamParser) handleSystemMessage(msg Message, events chan<- StreamEvent) {
	if msg.Subtype == "init" {
		events <- StreamEvent{
			Type: EventSystemInit,
			Data: SystemInitData{
				SkillName: p.skillName,
				SkillPath: p.skillPath,
			},
		}
	}
}

// handleAssistantMessage processes assistant response messages
func (p *ClaudeStreamParser) handleAssistantMessage(msg Message, events chan<- StreamEvent) {
	if msg.Message == nil {
		return
	}

	for _, content := range msg.Message.Content {
		switch content.Type {
		case "thinking":
			if content.Text != "" {
				events <- StreamEvent{
					Type: EventThinking,
					Data: ThinkingData{Text: content.Text},
				}
			}

		case "text":
			if content.Text != "" {
				events <- StreamEvent{
					Type: EventText,
					Data: TextData{Text: content.Text},
				}
			}

		case "tool_use":
			// Create a new tool operation
			op := ToolOperation{
				ID:     content.ID,
				Name:   content.Name,
				Target: p.extractTarget(content.Name, content.Input),
				Status: "pending",
				Input:  content.Input,
			}
			p.toolCallMap[content.ID] = len(p.tools)
			p.tools = append(p.tools, op)
		}
	}
}

// handleUserMessage processes user messages (typically tool results)
func (p *ClaudeStreamParser) handleUserMessage(msg Message, events chan<- StreamEvent) {
	if msg.Message == nil {
		return
	}

	for _, content := range msg.Message.Content {
		if content.Type == "tool_result" && content.ToolUseID != "" {
			if idx, ok := p.toolCallMap[content.ToolUseID]; ok {
				p.tools[idx].Result = content.Content
				p.tools[idx].Status = p.determineStatus(content.Content)
				if p.tools[idx].Status == "error" {
					p.tools[idx].Error = p.extractError(content.Content)
				}
				// Emit tool complete event
				events <- StreamEvent{
					Type: EventToolComplete,
					Data: ToolCompleteData{Operation: p.tools[idx]},
				}
			}
		}
	}
}

// handleResultMessage processes final result messages
func (p *ClaudeStreamParser) handleResultMessage(msg Message, events chan<- StreamEvent) {
	// Emit final result event
	elapsed := time.Since(p.startTime)
	events <- StreamEvent{
		Type: EventFinalResult,
		Data: FinalResultData{
			Result:  msg.Result,
			IsError: msg.IsError,
			Elapsed: elapsed,
		},
	}

	// Emit usage event if available
	if msg.Usage != nil {
		events <- StreamEvent{
			Type: EventUsage,
			Data: UsageData{Usage: msg.Usage},
		}
	}
}

// extractTarget extracts the key parameter from tool input
func (p *ClaudeStreamParser) extractTarget(toolName string, input map[string]any) string {
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
	case "TaskCreate":
		if subject, ok := input["subject"].(string); ok {
			return subject
		}
	case "TaskUpdate":
		if taskID, ok := input["taskId"].(string); ok {
			if status, ok := input["status"].(string); ok {
				return taskID + " → " + status
			}
			return taskID
		}
	case "TaskGet":
		if taskID, ok := input["taskId"].(string); ok {
			return taskID
		}
	case "TaskList":
		return ""
	}
	return ""
}

// determineStatus determines the status of a tool result
func (p *ClaudeStreamParser) determineStatus(content any) string {
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
func (p *ClaudeStreamParser) extractError(content any) string {
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
