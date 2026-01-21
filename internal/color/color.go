package color

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// ShouldUseColors determines if colors should be used based on the color setting
func ShouldUseColors(colorMode string) bool {
	switch colorMode {
	case "always":
		return true
	case "never":
		return false
	case "auto":
		// Check if output is a terminal
		if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
			// It's a terminal, check for NO_COLOR environment variable
			if os.Getenv("NO_COLOR") != "" {
				return false
			}
			return true
		}
		return false
	default:
		return true // Default to colors
	}
}

// ConfigureColorProfile sets the global lipgloss color profile based on the color mode.
// This must be called early before any lipgloss/glamour rendering to ensure colors
// are properly enabled or disabled when output is piped.
//
// For "always": Forces TrueColor profile to enable full color support regardless of
// TTY status. This allows colors to work in piped output (e.g., fzf preview).
//
// For "never": Forces Ascii profile which disables all colors.
//
// For "auto": Does nothing, letting lipgloss use its default TTY-based detection.
func ConfigureColorProfile(colorMode string) {
	switch colorMode {
	case "always":
		// Force TrueColor when user explicitly requests colors
		// This bypasses TTY detection entirely
		lipgloss.SetColorProfile(termenv.TrueColor)
	case "never":
		lipgloss.SetColorProfile(termenv.Ascii)
		// "auto" - let lipgloss use its default TTY-based detection
	}
}
