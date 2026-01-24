// Package promptserver provides IPC between the parent skillet process
// (which has TTY access) and the MCP server child process.
package promptserver

// SocketEnvVar is the environment variable name for the socket path
const SocketEnvVar = "SKILLET_PROMPT_SOCK"

// Request represents a prompt request from the MCP server to the parent
type Request struct {
	Type      string     `json:"type"`      // "ask_user_question"
	Questions []Question `json:"questions"` // questions to ask
}

// Response represents the parent's response to a prompt request
type Response struct {
	Success bool              `json:"success"`
	Answers map[string]string `json:"answers,omitempty"` // question -> answer
	Error   string            `json:"error,omitempty"`
}

// Question represents a single question from AskUserQuestion
type Question struct {
	Question    string   `json:"question"`
	Header      string   `json:"header"`
	Options     []Option `json:"options"`
	MultiSelect bool     `json:"multiSelect"`
}

// Option represents a selectable option for a question
type Option struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}
