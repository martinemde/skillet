package formatter

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/martinemde/skillet/internal/color"
)

// Icon and style definitions for terminal output
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

// applyColorToIcon applies or removes color from an icon style based on color mode
func applyColorToIcon(icon lipgloss.Style, colorMode string) lipgloss.Style {
	if !color.ShouldUseColors(colorMode) {
		// Return a plain style without color
		return lipgloss.NewStyle().SetString(icon.Value())
	}
	return icon
}

// applyColorToStyle applies or removes color from a style based on color mode
func applyColorToStyle(style lipgloss.Style, colorMode string) lipgloss.Style {
	if !color.ShouldUseColors(colorMode) {
		// Return a plain style without color, but preserve other properties like margin
		return lipgloss.NewStyle()
	}
	return style
}
