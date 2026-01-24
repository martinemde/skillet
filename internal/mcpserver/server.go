// Package mcpserver provides an MCP server for handling permission prompts
// and user questions in Claude CLI's headless mode.
package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/martinemde/skillet/internal/promptserver"
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
	// Get client to connect to parent prompt server
	client := promptserver.NewClient()
	if client == nil {
		return PromptOutput{
			Behavior: "deny",
			Message:  "Cannot prompt user: SKILLET_PROMPT_SOCK not set",
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

	// Convert to promptserver.Question format
	psQuestions := make([]promptserver.Question, len(questions))
	for i, q := range questions {
		psQuestions[i] = promptserver.Question{
			Question:    q.Question,
			Header:      q.Header,
			MultiSelect: q.MultiSelect,
			Options:     make([]promptserver.Option, len(q.Options)),
		}
		for j, opt := range q.Options {
			psQuestions[i].Options[j] = promptserver.Option{
				Label:       opt.Label,
				Description: opt.Description,
			}
		}
	}

	// Send to parent for prompting
	answers, err := client.AskUserQuestion(psQuestions)
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
