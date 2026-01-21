package formatter

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// FormatterConfig holds configuration options for formatters
type FormatterConfig struct {
	Output    io.Writer
	ShowUsage bool
	Color     string // Color mode: "auto", "always", or "never"
}

// Formatter is the interface all formatters must implement
type Formatter interface {
	Format(events <-chan StreamEvent) error
}

// Config holds configuration options for the legacy Format function
// Deprecated: Use NewParser and specific formatter types instead
type Config struct {
	Output          io.Writer
	Verbose         bool
	Debug           bool // If true, print raw JSON lines to stderr
	ShowUsage       bool
	PassthroughMode bool   // If true, stream output directly without parsing
	SkillName       string // Name of the skill being executed
	SkillPath       string // Path to the skill/command file being executed
	Color           string // Color mode: "auto", "always", or "never"
}

// Formatter struct for backward compatibility
// Deprecated: Use specific formatter types (TerminalFormatter, VerboseTerminalFormatter) instead
type legacyFormatter struct {
	output          io.Writer
	verbose         bool
	debug           bool
	showUsage       bool
	passthroughMode bool
	skillName       string
	skillPath       string
	color           string
}

// New creates a formatter with the legacy API
// Deprecated: Use NewStreamParser with specific formatter types instead
func New(cfg Config) *legacyFormatter {
	return &legacyFormatter{
		output:          cfg.Output,
		verbose:         cfg.Verbose,
		debug:           cfg.Debug,
		showUsage:       cfg.ShowUsage,
		passthroughMode: cfg.PassthroughMode,
		skillName:       cfg.SkillName,
		skillPath:       cfg.SkillPath,
		color:           cfg.Color,
	}
}

// Format implements the legacy formatting API
func (f *legacyFormatter) Format(input io.Reader) error {
	// If user explicitly set --output-format, passthrough raw output directly
	if f.passthroughMode {
		_, err := io.Copy(f.output, input)
		return err
	}

	// Create a pipe for the scanner to read from and formatters to consume
	pr, pw := io.Pipe()

	// Start scanner goroutine that reads input and writes to pipe
	scanErr := make(chan error, 1)
	go func() {
		defer func() { _ = pw.Close() }()

		scanner := bufio.NewScanner(input)
		const maxScannerBuffer = 1024 * 1024
		scanner.Buffer(make([]byte, 0, 64*1024), maxScannerBuffer)

		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "" {
				continue
			}

			// In debug mode, print raw JSON to stderr
			if f.debug {
				fmt.Fprintf(os.Stderr, "%s\n", line)
			}

			// Write line to pipe for parser
			_, _ = fmt.Fprintln(pw, line)
		}

		if err := scanner.Err(); err != nil {
			scanErr <- err
		} else {
			scanErr <- nil
		}
	}()

	// Create parser
	parser := NewStreamParser(f.skillName, f.skillPath, f.verbose)
	events, parserErr := parser.Parse(pr)

	// Create appropriate formatter based on verbose flag
	var formatter Formatter
	if f.verbose {
		formatter = NewVerboseTerminalFormatter(FormatterConfig{
			Output:    f.output,
			ShowUsage: f.showUsage,
			Color:     f.color,
		})
	} else {
		formatter = NewTerminalFormatter(FormatterConfig{
			Output:    f.output,
			ShowUsage: f.showUsage,
			Color:     f.color,
		})
	}

	// Format events (blocks until all events are processed)
	formatErr := formatter.Format(events)

	// Wait for parser error channel to close
	parseErr := <-parserErr

	// Wait for scanner to complete
	scannerErr := <-scanErr

	// Return first error encountered
	if scannerErr != nil {
		return fmt.Errorf("scanning failed: %w", scannerErr)
	}
	if parseErr != nil {
		return fmt.Errorf("parsing failed: %w", parseErr)
	}
	if formatErr != nil {
		return fmt.Errorf("formatting failed: %w", formatErr)
	}

	return nil
}
