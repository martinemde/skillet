package formatter

import "github.com/charmbracelet/lipgloss"

// Icon and style definitions for terminal output.
// Colors are automatically handled by the global lipgloss color profile
// which is configured by color.ConfigureColorProfile() based on --color flag.
var (
	successIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).SetString("✓")
	errorIcon   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).SetString("✗")
	emptyIcon   = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).SetString("☐")
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
