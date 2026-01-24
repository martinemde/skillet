package formatter

import (
	"fmt"
	"io"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// TerminalFormatter formats output for normal (non-verbose) terminal mode
type TerminalFormatter struct {
	output     io.Writer
	color      string
	showUsage  bool
	mdRenderer *glamour.TermRenderer
}

// NewTerminalFormatter creates a new terminal formatter
func NewTerminalFormatter(cfg FormatterConfig) *TerminalFormatter {
	return &TerminalFormatter{
		output:     cfg.Output,
		color:      cfg.Color,
		showUsage:  cfg.ShowUsage,
		mdRenderer: createMarkdownRenderer(cfg.Color),
	}
}

// Format processes events and renders terminal output
func (f *TerminalFormatter) Format(events <-chan StreamEvent) error {
	for event := range events {
		switch event.Type {
		case EventSystemInit:
			f.printSystemInit(event.Data.(SystemInitData))
		case EventThinking:
			// Skip thinking blocks in non-verbose mode
		case EventText:
			// Skip text content in non-verbose mode (shown at end in result)
		case EventToolComplete:
			f.printToolOperation(event.Data.(ToolCompleteData).Operation)
		case EventFinalResult:
			f.printFinalResult(event.Data.(FinalResultData))
		case EventUsage:
			if f.showUsage {
				f.printUsage(event.Data.(UsageData).Usage)
			}
		}
	}
	return nil
}

// printSystemInit prints the system initialization message
func (f *TerminalFormatter) printSystemInit(data SystemInitData) {
	if data.SkillName != "" {
		_, _ = fmt.Fprintf(f.output, "%s Starting %s\n", successIcon.String(), data.SkillName)
	} else {
		_, _ = fmt.Fprintf(f.output, "%s Starting\n", successIcon.String())
	}
}

// printToolOperation prints a single tool operation
func (f *TerminalFormatter) printToolOperation(tool ToolOperation) {
	// Special handling for TodoWrite - show todos as status lines
	if tool.Name == "TodoWrite" {
		f.printTodoStatusLines(tool)
		return
	}

	// Special handling for Task tools - show task-specific output
	if tool.Name == "TaskCreate" || tool.Name == "TaskUpdate" ||
		tool.Name == "TaskGet" || tool.Name == "TaskList" {
		f.printTaskOperation(tool)
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

	// Format tool line
	line := fmt.Sprintf("%s %s", icon.String(), tool.Name)
	if tool.Target != "" {
		line += " " + tool.Target
	}
	if tool.Status == "error" && tool.Error != "" {
		line += dimStyle.Render(fmt.Sprintf(" (%s)", tool.Error))
	}
	_, _ = fmt.Fprintln(f.output, line)
}

// printTaskOperation displays task-specific output in a meaningful way
func (f *TerminalFormatter) printTaskOperation(tool ToolOperation) {
	switch tool.Name {
	case "TaskCreate":
		// Show subject with task icon
		if subject, ok := tool.Input["subject"].(string); ok {
			_, _ = fmt.Fprintf(f.output, "  ☐ %s\n", subject)
		}
		// Show description in dim style
		if desc, ok := tool.Input["description"].(string); ok {
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			_, _ = fmt.Fprintln(f.output, "  "+dimStyle.Render(desc))
		}
		// Show metadata if present
		if metadata, ok := tool.Input["metadata"].(map[string]any); ok && len(metadata) > 0 {
			_, _ = fmt.Fprintln(f.output)
			_, _ = fmt.Fprintln(f.output, "  "+dimStyle.Render("Metadata:"))
			for k, v := range metadata {
				_, _ = fmt.Fprintln(f.output, "  "+dimStyle.Render(fmt.Sprintf("- %s: %v", k, v)))
			}
		}

	case "TaskUpdate":
		// Show status change with visual indicator
		taskID := ""
		if id, ok := tool.Input["taskId"].(string); ok {
			taskID = id
		}
		if status, ok := tool.Input["status"].(string); ok {
			statusIcon := "○"
			switch status {
			case "in_progress":
				statusIcon = "◐"
			case "completed":
				statusIcon = "●"
			}
			_, _ = fmt.Fprintf(f.output, "%s %s → %s\n", statusIcon, taskID, status)
		} else {
			_, _ = fmt.Fprintf(f.output, "✓ TaskUpdate %s\n", taskID)
		}

	case "TaskGet":
		// Show task details
		if taskID, ok := tool.Input["taskId"].(string); ok {
			_, _ = fmt.Fprintf(f.output, "✓ TaskGet %s\n", taskID)
		} else {
			_, _ = fmt.Fprintln(f.output, "✓ TaskGet")
		}

	case "TaskList":
		_, _ = fmt.Fprintln(f.output, "✓ TaskList")
	}
}

// printTodoStatusLines shows todos as individual status lines
func (f *TerminalFormatter) printTodoStatusLines(tool ToolOperation) {
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
			_, _ = fmt.Fprintf(f.output, "  %s\n", dimStyle.Render("☒ "+lastCompleted))
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
					_, _ = fmt.Fprintf(f.output, "  ☐ %s\n", content)
				} else {
					// Pending: dimmed with empty checkbox
					_, _ = fmt.Fprintf(f.output, "  %s\n", dimStyle.Render("☐ "+content))
				}
			}
		}
	}
}

// printFinalResult prints the final result and completion status
func (f *TerminalFormatter) printFinalResult(data FinalResultData) {
	// Print the final result with markdown rendering
	if data.Result != "" {
		rendered := renderMarkdown(f.mdRenderer, data.Result)
		_, _ = fmt.Fprintln(f.output, rendered)
	}

	// Print completion status
	_, _ = fmt.Fprintln(f.output)
	if data.IsError {
		_, _ = fmt.Fprintln(f.output, errorIcon.String()+" Failed")
	} else {
		_, _ = fmt.Fprintf(f.output, "%s Completed in %.1fs\n", successIcon.String(), data.Elapsed.Seconds())
	}
}

// printUsage prints token usage information in a styled table
func (f *TerminalFormatter) printUsage(usage *Usage) {
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

	// Create styled table - colors are handled by global lipgloss profile
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("8"))).
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
