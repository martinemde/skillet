package formatter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/log"
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

// Formatter formats stream-json output from Claude CLI
type Formatter struct {
	output    io.Writer
	verbose   bool
	showUsage bool
	logger    *log.Logger
	toolCount int
}

// New creates a new Formatter
func New(output io.Writer, verbose, showUsage bool) *Formatter {
	logger := log.New(output)
	logger.SetReportCaller(false)
	logger.SetReportTimestamp(false)

	// Set log level based on verbose flag
	if verbose {
		logger.SetLevel(log.DebugLevel)
	} else {
		logger.SetLevel(log.InfoLevel)
	}

	return &Formatter{
		output:    output,
		verbose:   verbose,
		showUsage: showUsage,
		logger:    logger,
		toolCount: 0,
	}
}

// Format reads stream-json input and formats it
func (f *Formatter) Format(input io.Reader) error {
	scanner := bufio.NewScanner(input)
	var textBuilder strings.Builder
	toolCallMap := make(map[string]string) // Maps tool_use_id to tool name

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			f.logger.Debug("Failed to parse JSON", "error", err)
			continue
		}

		// In verbose mode, print the raw JSON
		f.logger.Debug("stream", "data", line)

		switch msg.Type {
		case "system":
			if msg.Subtype == "init" {
				f.logger.Info("Session initialized")
			}

		case "assistant":
			if msg.Message != nil {
				for _, content := range msg.Message.Content {
					switch content.Type {
					case "text":
						if content.Text != "" {
							textBuilder.WriteString(content.Text)
						}

					case "tool_use":
						f.toolCount++
						// Store the tool name for later reference
						toolCallMap[content.ID] = content.Name

						// Format the input for display
						inputStr := f.formatToolInput(content.Input)
						f.logger.Info("Tool call",
							"tool", content.Name,
							"id", fmt.Sprintf("#%d", f.toolCount),
							"input", inputStr)
					}
				}
			}

		case "user":
			// Handle tool results
			if msg.Message != nil {
				for _, content := range msg.Message.Content {
					if content.Type == "tool_result" && content.ToolUseID != "" {
						toolName := toolCallMap[content.ToolUseID]
						resultStr := f.formatToolResult(content.Content)

						f.logger.Info("Tool result",
							"tool", toolName,
							"result", resultStr)
					}
				}
			}

		case "result":
			// Print any accumulated text
			if textBuilder.Len() > 0 {
				text := textBuilder.String()
				// Print the assistant's text response
				if text != "" {
					_, _ = fmt.Fprintln(f.output, text)
				}
				textBuilder.Reset()
			}

			// Print result if it's different from accumulated text
			if msg.Result != "" && msg.Result != textBuilder.String() {
				_, _ = fmt.Fprintln(f.output, msg.Result)
			}

			// Print usage information if requested
			if f.showUsage && msg.Usage != nil {
				f.printUsage(msg.Usage)
			}

			// Print error information if present
			if msg.IsError {
				f.logger.Error("Execution failed", "subtype", msg.Subtype)
			} else {
				f.logger.Info("Execution completed")
			}
		}
	}

	// Print any remaining text
	if textBuilder.Len() > 0 {
		_, _ = fmt.Fprint(f.output, textBuilder.String())
	}

	return scanner.Err()
}

// formatToolInput formats tool input for display
func (f *Formatter) formatToolInput(input map[string]interface{}) string {
	if len(input) == 0 {
		return ""
	}

	// Create a compact representation
	parts := make([]string, 0, len(input))
	for k, v := range input {
		vStr := fmt.Sprintf("%v", v)
		// Truncate long values
		if len(vStr) > 60 {
			vStr = vStr[:57] + "..."
		}
		parts = append(parts, fmt.Sprintf("%s=%s", k, vStr))
	}
	return strings.Join(parts, ", ")
}

// formatToolResult formats tool result for display
func (f *Formatter) formatToolResult(content interface{}) string {
	switch v := content.(type) {
	case string:
		if len(v) > 100 {
			return v[:97] + "..."
		}
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
		result := builder.String()
		if len(result) > 100 {
			return result[:97] + "..."
		}
		return result
	default:
		str := fmt.Sprintf("%v", content)
		if len(str) > 100 {
			return str[:97] + "..."
		}
		return str
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
