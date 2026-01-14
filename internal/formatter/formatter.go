package formatter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
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
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
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
}

// New creates a new Formatter
func New(output io.Writer, verbose, showUsage bool) *Formatter {
	return &Formatter{
		output:    output,
		verbose:   verbose,
		showUsage: showUsage,
	}
}

// Format reads stream-json input and formats it
func (f *Formatter) Format(input io.Reader) error {
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
				_, _ = fmt.Fprintf(f.output, "[ERROR] Failed to parse JSON: %v\n", err)
			}
			continue
		}

		switch msg.Type {
		case "assistant":
			if msg.Message != nil {
				for _, content := range msg.Message.Content {
					if content.Type == "text" && content.Text != "" {
						textBuilder.WriteString(content.Text)
					}
				}
			}

		case "result":
			// Print any accumulated text
			if textBuilder.Len() > 0 {
				_, _ = fmt.Fprint(f.output, textBuilder.String())
				textBuilder.Reset()
			}

			// Print result if it's different from accumulated text
			if msg.Result != "" {
				_, _ = fmt.Fprintln(f.output, msg.Result)
			}

			// Print usage information if requested
			if f.showUsage && msg.Usage != nil {
				f.printUsage(msg.Usage)
			}

			// Print error information if present
			if msg.IsError {
				_, _ = fmt.Fprintf(f.output, "\n[ERROR] Execution failed (subtype: %s)\n", msg.Subtype)
			}
		}

		// In verbose mode, print the raw JSON
		if f.verbose {
			_, _ = fmt.Fprintf(f.output, "[DEBUG] %s\n", line)
		}
	}

	// Print any remaining text
	if textBuilder.Len() > 0 {
		_, _ = fmt.Fprint(f.output, textBuilder.String())
	}

	return scanner.Err()
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
