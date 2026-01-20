package formatter

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/martinemde/skillet/internal/color"
)

// createMarkdownRenderer initializes a glamour markdown renderer
func createMarkdownRenderer(colorMode string) *glamour.TermRenderer {
	useColors := color.ShouldUseColors(colorMode)

	if !useColors {
		return nil
	}

	// Use "auto" style which adapts to terminal background (light/dark)
	mdRenderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0), // No wrapping, let terminal handle it
	)
	if err != nil {
		// Fallback to nil renderer if initialization fails
		return nil
	}

	return mdRenderer
}

// renderMarkdown renders markdown text using glamour, or returns plain text if unavailable
func renderMarkdown(mdRenderer *glamour.TermRenderer, text string) string {
	// If no renderer available, return plain text
	if mdRenderer == nil {
		return text
	}

	// Render the markdown
	rendered, err := mdRenderer.Render(text)
	if err != nil {
		// Fallback to plain text if rendering fails
		return text
	}

	// glamour adds leading/trailing newlines for formatting, trim them
	return strings.TrimSpace(rendered)
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

// extractResultText extracts text from a tool result
func extractResultText(result any) string {
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
		return ""
	}
}
