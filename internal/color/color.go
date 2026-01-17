package color

import "os"

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
