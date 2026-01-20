package formatter

import "time"

// EventType identifies different stream events
type EventType int

const (
	EventSystemInit EventType = iota
	EventThinking
	EventText
	EventToolComplete
	EventFinalResult
	EventUsage
)

// StreamEvent represents a parsed event from the Claude stream
type StreamEvent struct {
	Type EventType
	Data any
}

// SystemInitData represents system initialization event data
type SystemInitData struct {
	SkillName string
	SkillPath string
}

// ThinkingData represents a thinking block event
type ThinkingData struct {
	Text string
}

// TextData represents text content event
type TextData struct {
	Text string
}

// ToolCompleteData represents a completed tool operation
type ToolCompleteData struct {
	Operation ToolOperation
}

// FinalResultData represents the final result event
type FinalResultData struct {
	Result  string
	IsError bool
	Elapsed time.Duration
}

// UsageData represents token usage information
type UsageData struct {
	Usage *Usage
}
