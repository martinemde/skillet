package promptserver

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

// Client connects to the parent prompt server via Unix socket
type Client struct {
	socketPath string
}

// NewClient creates a client that connects to the prompt server
// Returns nil if SKILLET_PROMPT_SOCK is not set
func NewClient() *Client {
	socketPath := os.Getenv(SocketEnvVar)
	if socketPath == "" {
		return nil
	}
	return &Client{socketPath: socketPath}
}

// NewClientWithPath creates a client with an explicit socket path
func NewClientWithPath(socketPath string) *Client {
	return &Client{socketPath: socketPath}
}

// AskUserQuestion sends questions to the parent and returns answers
func (c *Client) AskUserQuestion(questions []Question) (map[string]string, error) {
	// Connect to socket with timeout
	conn, err := net.DialTimeout("unix", c.socketPath, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to prompt server: %w", err)
	}
	defer func() { _ = conn.Close() }()

	// Set deadline for the entire operation (60 seconds to match MCP timeout)
	if err := conn.SetDeadline(time.Now().Add(60 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set deadline: %w", err)
	}

	// Send request
	req := Request{
		Type:      "ask_user_question",
		Questions: questions,
	}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	decoder := json.NewDecoder(conn)
	var resp Response
	if err := decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("prompt failed: %s", resp.Error)
	}

	return resp.Answers, nil
}
