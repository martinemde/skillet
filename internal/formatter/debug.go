package formatter

import "io"

// DebugFormatter is a no-op formatter for debug mode
// Debug output (raw JSONL) is printed to stderr before parsing,
// so this formatter just drains the event channel
type DebugFormatter struct {
	output io.Writer
}

// NewDebugFormatter creates a new debug formatter
func NewDebugFormatter(output io.Writer) *DebugFormatter {
	return &DebugFormatter{
		output: output,
	}
}

// Format drains the event channel (events are already logged to stderr)
func (f *DebugFormatter) Format(events <-chan StreamEvent) error {
	// Drain channel - debug printing happens before parsing in main flow
	for range events {
		// No-op
	}
	return nil
}
