// Package mcpserver provides an MCP server for handling permission prompts
// and user questions in Claude CLI's headless mode.
package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/term"
)

// PromptInput is the input structure for the permission prompt tool
type PromptInput struct {
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"input"`
}

// PromptOutput is the response structure for the permission prompt tool
type PromptOutput struct {
	Behavior     string         `json:"behavior"`
	UpdatedInput map[string]any `json:"updatedInput,omitempty"`
	Message      string         `json:"message,omitempty"`
}

// Run starts the MCP server in stdio mode
func Run() error {
	s := server.NewMCPServer(
		"skillet",
		"0.1.0",
		server.WithToolCapabilities(false),
	)

	// Add the prompt tool
	tool := mcp.NewTool("prompt",
		mcp.WithDescription("Handle permission prompts and user questions from Claude CLI"),
		mcp.WithString("tool_name",
			mcp.Required(),
			mcp.Description("Name of the tool being prompted for"),
		),
		mcp.WithObject("input",
			mcp.Required(),
			mcp.Description("Input parameters for the tool"),
		),
	)

	s.AddTool(tool, handlePrompt)

	return server.ServeStdio(s)
}

// handlePrompt handles incoming prompt requests
func handlePrompt(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Parse input
	var input PromptInput
	inputBytes, err := json.Marshal(req.Params.Arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}
	if err := json.Unmarshal(inputBytes, &input); err != nil {
		return nil, fmt.Errorf("failed to parse prompt input: %w", err)
	}

	var output PromptOutput

	switch input.ToolName {
	case "AskUserQuestion":
		output, err = handleAskUserQuestion(input.ToolInput)
		if err != nil {
			return nil, err
		}
	default:
		// For all other tools, auto-allow (permission-mode handles restrictions)
		output = PromptOutput{
			Behavior:     "allow",
			UpdatedInput: input.ToolInput,
		}
	}

	// Marshal output to JSON
	outputBytes, err := json.Marshal(output)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}

	return mcp.NewToolResultText(string(outputBytes)), nil
}

// handleAskUserQuestion processes AskUserQuestion tool calls
func handleAskUserQuestion(toolInput map[string]any) (PromptOutput, error) {
	// Check if we have a TTY for interactive prompts
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return PromptOutput{
			Behavior: "deny",
			Message:  "Cannot prompt user: not running in a terminal",
		}, nil
	}

	// Parse questions from input
	questions, err := parseQuestions(toolInput)
	if err != nil {
		return PromptOutput{
			Behavior: "deny",
			Message:  fmt.Sprintf("Failed to parse questions: %v", err),
		}, nil
	}

	// Prompt user for answers
	answers, err := promptUser(questions)
	if err != nil {
		return PromptOutput{
			Behavior: "deny",
			Message:  fmt.Sprintf("User prompt failed: %v", err),
		}, nil
	}

	// Return answers in updatedInput
	return PromptOutput{
		Behavior: "allow",
		UpdatedInput: map[string]any{
			"questions": toolInput["questions"],
			"answers":   answers,
		},
	}, nil
}
