package formatter

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/martinemde/skillet/internal/color"
)

// Output truncation limits for verbose mode
const (
	maxGenericOutputLength = 200
	maxReadOutputLines     = 20
	maxBashOutputLines     = 30
	maxSearchOutputLines   = 15
)

// VerboseTerminalFormatter formats output for verbose terminal mode
type VerboseTerminalFormatter struct {
	output     io.Writer
	color      string
	showUsage  bool
	mdRenderer *glamour.TermRenderer
}

// NewVerboseTerminalFormatter creates a new verbose terminal formatter
func NewVerboseTerminalFormatter(cfg FormatterConfig) *VerboseTerminalFormatter {
	return &VerboseTerminalFormatter{
		output:     cfg.Output,
		color:      cfg.Color,
		showUsage:  cfg.ShowUsage,
		mdRenderer: createMarkdownRenderer(cfg.Color),
	}
}

// Format processes events and renders verbose terminal output
func (f *VerboseTerminalFormatter) Format(events <-chan StreamEvent) error {
	for event := range events {
		switch event.Type {
		case EventSystemInit:
			f.printSystemInit(event.Data.(SystemInitData))
		case EventThinking:
			f.printThinking(event.Data.(ThinkingData))
		case EventText:
			f.printText(event.Data.(TextData))
		case EventToolComplete:
			f.printToolOperationWithDetails(event.Data.(ToolCompleteData).Operation)
		case EventFinalResult:
			// Skip result text (already streamed), just print completion
			f.printCompletion(event.Data.(FinalResultData))
		case EventUsage:
			if f.showUsage {
				f.printUsage(event.Data.(UsageData).Usage)
			}
		}
	}
	return nil
}

// printSystemInit prints the system initialization message with path
func (f *VerboseTerminalFormatter) printSystemInit(data SystemInitData) {
	icon := applyColorToIcon(successIcon, f.color)
	if data.SkillName != "" {
		// In verbose mode, append the path in dim style
		if data.SkillPath != "" {
			pathStyle := applyColorToStyle(dimStyle, f.color)
			_, _ = fmt.Fprintf(f.output, "%s Starting %s %s\n", icon.String(), data.SkillName, pathStyle.Render(data.SkillPath))
		} else {
			_, _ = fmt.Fprintf(f.output, "%s Starting %s\n", icon.String(), data.SkillName)
		}
	} else {
		_, _ = fmt.Fprintf(f.output, "%s Starting\n", icon.String())
	}
}

// printThinking prints a thinking block
func (f *VerboseTerminalFormatter) printThinking(data ThinkingData) {
	if data.Text == "" {
		return
	}
	style := applyColorToStyle(thinkingStyle, f.color)
	_, _ = fmt.Fprintln(f.output, style.Render("💭 "+data.Text))
	_, _ = fmt.Fprintln(f.output)
}

// printText prints text content with markdown rendering
func (f *VerboseTerminalFormatter) printText(data TextData) {
	if data.Text == "" {
		return
	}
	rendered := renderMarkdown(f.mdRenderer, data.Text)
	_, _ = fmt.Fprintln(f.output, rendered)
}

// printToolOperationWithDetails prints a tool operation with detailed output
func (f *VerboseTerminalFormatter) printToolOperationWithDetails(tool ToolOperation) {
	// Special handling for TodoWrite - show todos as status lines
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
	icon = applyColorToIcon(icon, f.color)

	// Format tool line
	line := fmt.Sprintf("%s %s", icon.String(), tool.Name)
	if tool.Target != "" {
		line += " " + tool.Target
	}
	if tool.Status == "error" && tool.Error != "" {
		style := applyColorToStyle(dimStyle, f.color)
		line += style.Render(fmt.Sprintf(" (%s)", tool.Error))
	}
	_, _ = fmt.Fprintln(f.output, line)

	// Show tool details in a box
	f.printToolDetails(tool)
}

// printToolDetails prints detailed tool information in a bordered box
func (f *VerboseTerminalFormatter) printToolDetails(tool ToolOperation) {
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
		// Handled by status line
	default:
		// For other tools, show basic input/output
		f.buildGenericToolOutput(&content, tool)
	}

	// Only print if there's content
	if content.Len() > 0 {
		// Apply color to box style
		boxStyle := toolBoxStyle
		if !color.ShouldUseColors(f.color) {
			boxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				Padding(0, 1).
				MarginLeft(2).
				MarginTop(0).
				MarginBottom(1)
		}
		// Wrap in styled box
		boxed := boxStyle.Render(strings.TrimRight(content.String(), "\n"))
		_, _ = fmt.Fprintln(f.output, boxed)
	}
}

// buildTruncatedLines writes text with line truncation to a builder
func (f *VerboseTerminalFormatter) buildTruncatedLines(w *strings.Builder, text string, maxLines int, label string) {
	if text == "" {
		return
	}

	text = strings.TrimRight(text, "\n\r")
	lines := strings.Split(text, "\n")

	if len(lines) > maxLines {
		content := strings.Join(lines[:maxLines], "\n")
		fmt.Fprintln(w, content)
		style := applyColorToStyle(dimStyle, f.color)
		fmt.Fprintln(w, style.Render(fmt.Sprintf("... (%d more %s)", len(lines)-maxLines, label)))
	} else {
		fmt.Fprintln(w, text)
	}
}

// buildReadOutput writes file contents for Read operations
func (f *VerboseTerminalFormatter) buildReadOutput(w *strings.Builder, tool ToolOperation) {
	if tool.Result == nil {
		return
	}
	resultStr := extractResultText(tool.Result)
	f.buildTruncatedLines(w, resultStr, maxReadOutputLines, "lines")
}

// buildWriteOutput writes confirmation for Write/Edit operations
func (f *VerboseTerminalFormatter) buildWriteOutput(w *strings.Builder, tool ToolOperation) {
	if tool.Status == "success" {
		if filePath, ok := tool.Input["file_path"].(string); ok {
			style := applyColorToStyle(dimStyle, f.color)
			fmt.Fprintln(w, style.Render(fmt.Sprintf("→ wrote to %s", filePath)))
		}
	}
}

// buildBashOutput writes command output for Bash operations
func (f *VerboseTerminalFormatter) buildBashOutput(w *strings.Builder, tool ToolOperation) {
	// Show the command itself as a code block
	if cmd, ok := tool.Input["command"].(string); ok {
		cmdBlock := fmt.Sprintf("```sh\n$ %s\n```", cmd)
		rendered := renderMarkdown(f.mdRenderer, cmdBlock)
		fmt.Fprint(w, rendered)
	}

	if tool.Result == nil {
		return
	}

	resultStr := extractResultText(tool.Result)
	if resultStr == "" {
		return
	}

	// Show command output in a code block for consistent styling
	resultStr = strings.TrimRight(resultStr, "\n\r")
	lines := strings.Split(resultStr, "\n")

	if len(lines) > maxBashOutputLines {
		content := strings.Join(lines[:maxBashOutputLines], "\n")
		resultBlock := fmt.Sprintf("```sh\n%s\n```", content)
		rendered := renderMarkdown(f.mdRenderer, resultBlock)
		fmt.Fprint(w, rendered)
		style := applyColorToStyle(dimStyle, f.color)
		fmt.Fprintln(w, style.Render(fmt.Sprintf("... (%d more lines)", len(lines)-maxBashOutputLines)))
	} else {
		resultBlock := fmt.Sprintf("```sh\n%s\n```", resultStr)
		rendered := renderMarkdown(f.mdRenderer, resultBlock)
		fmt.Fprint(w, rendered)
	}
}

// buildSearchOutput writes search results for Grep/Glob operations
func (f *VerboseTerminalFormatter) buildSearchOutput(w *strings.Builder, tool ToolOperation) {
	if tool.Result == nil {
		return
	}
	resultStr := extractResultText(tool.Result)
	f.buildTruncatedLines(w, resultStr, maxSearchOutputLines, "results")
}

// buildGenericToolOutput writes basic input/output for other tools
func (f *VerboseTerminalFormatter) buildGenericToolOutput(w *strings.Builder, tool ToolOperation) {
	style := applyColorToStyle(dimStyle, f.color)

	// Show key input parameters
	if len(tool.Input) > 0 {
		for k, v := range tool.Input {
			vStr := fmt.Sprintf("%v", v)
			if len(vStr) > maxErrorDisplayLength {
				vStr = vStr[:maxErrorDisplayLength-3] + "..."
			}
			fmt.Fprintln(w, style.Render(fmt.Sprintf("→ %s: %s", k, vStr)))
		}
	}

	// Show result summary
	if tool.Result != nil && tool.Status != "error" {
		resultStr := extractResultText(tool.Result)
		if resultStr != "" && len(resultStr) < maxGenericOutputLength {
			fmt.Fprintln(w, style.Render(fmt.Sprintf("→ %s", resultStr)))
		}
	}
}

// printTodoStatusLines shows todos as individual status lines
func (f *VerboseTerminalFormatter) printTodoStatusLines(tool ToolOperation) {
	// Extract todos from the input
	if todos, ok := tool.Input["todos"].([]any); ok {
		// Find the most recently completed task
		var lastCompleted string
		for _, todoItem := range todos {
			if todo, ok := todoItem.(map[string]any); ok {
				if status, _ := todo["status"].(string); status == "completed" {
					lastCompleted, _ = todo["content"].(string)
				}
			}
		}

		// Show the most recently completed task first (dimmed with ☒)
		if lastCompleted != "" {
			style := applyColorToStyle(dimStyle, f.color)
			_, _ = fmt.Fprintf(f.output, "%s\n", style.Render("☒ "+lastCompleted))
		}

		// Show remaining tasks
		for _, todoItem := range todos {
			if todo, ok := todoItem.(map[string]any); ok {
				content, _ := todo["content"].(string)
				status, _ := todo["status"].(string)

				if status == "completed" {
					// Skip completed items (we showed the last one above)
					continue
				} else if status == "in_progress" {
					// In-progress: prominent with empty checkbox
					_, _ = fmt.Fprintf(f.output, "☐ %s\n", content)
				} else {
					// Pending: dimmed with empty checkbox
					style := applyColorToStyle(dimStyle, f.color)
					_, _ = fmt.Fprintf(f.output, "%s\n", style.Render("☐ "+content))
				}
			}
		}
	}
}

// printCompletion prints only the completion status (result text already streamed)
func (f *VerboseTerminalFormatter) printCompletion(data FinalResultData) {
	// Don't print result text - it was already streamed as EventText
	// Just print completion status
	_, _ = fmt.Fprintln(f.output)
	if data.IsError {
		icon := applyColorToIcon(errorIcon, f.color)
		_, _ = fmt.Fprintln(f.output, icon.String()+" Failed")
	} else {
		icon := applyColorToIcon(successIcon, f.color)
		_, _ = fmt.Fprintf(f.output, "%s Completed in %.1fs\n", icon.String(), data.Elapsed.Seconds())
	}
}

// printUsage prints token usage information in a styled table
func (f *VerboseTerminalFormatter) printUsage(usage *Usage) {
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

	// Create styled table with color support
	borderStyle := lipgloss.NewStyle()
	if color.ShouldUseColors(f.color) {
		borderStyle = borderStyle.Foreground(lipgloss.Color("8"))
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(borderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			if col == 0 {
				// Metric name column - dim style
				return applyColorToStyle(dimStyle, f.color)
			}
			// Value column - normal style
			return lipgloss.NewStyle()
		}).
		Headers("Usage Statistics", "Count").
		Rows(rows...)

	_, _ = fmt.Fprintln(f.output)
	_, _ = fmt.Fprintln(f.output, t)
}
