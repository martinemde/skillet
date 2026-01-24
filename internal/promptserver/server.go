package promptserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/charmbracelet/huh"
)

// Server handles prompt requests from MCP server children via Unix socket
type Server struct {
	socketPath string
	listener   net.Listener
	mu         sync.Mutex
	done       chan struct{}
}

// New creates a new prompt server with a unique socket path
func New() (*Server, error) {
	// Create socket in temp directory with unique name
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("skillet-%d.sock", os.Getpid()))

	return &Server{
		socketPath: socketPath,
		done:       make(chan struct{}),
	}, nil
}

// SocketPath returns the path to the Unix socket
func (s *Server) SocketPath() string {
	return s.socketPath
}

// Start begins listening for connections
func (s *Server) Start(ctx context.Context) error {
	// Remove any existing socket file
	_ = os.Remove(s.socketPath)

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on socket: %w", err)
	}
	s.listener = listener

	go s.acceptLoop(ctx)

	return nil
}

// Stop closes the server and removes the socket
func (s *Server) Stop() {
	close(s.done)
	if s.listener != nil {
		_ = s.listener.Close()
	}
	_ = os.Remove(s.socketPath)
}

// acceptLoop accepts incoming connections
func (s *Server) acceptLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				continue
			}
		}

		go s.handleConnection(conn)
	}
}

// handleConnection processes a single connection
func (s *Server) handleConnection(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	// Serialize prompt handling to avoid multiple prompts at once
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read request
	decoder := json.NewDecoder(conn)
	var req Request
	if err := decoder.Decode(&req); err != nil {
		s.sendError(conn, fmt.Sprintf("failed to decode request: %v", err))
		return
	}

	// Handle request
	var resp Response
	switch req.Type {
	case "ask_user_question":
		answers, err := s.promptUser(req.Questions)
		if err != nil {
			resp = Response{Success: false, Error: err.Error()}
		} else {
			resp = Response{Success: true, Answers: answers}
		}
	default:
		resp = Response{Success: false, Error: fmt.Sprintf("unknown request type: %s", req.Type)}
	}

	// Send response
	encoder := json.NewEncoder(conn)
	_ = encoder.Encode(resp)
}

// sendError sends an error response
func (s *Server) sendError(conn net.Conn, msg string) {
	encoder := json.NewEncoder(conn)
	_ = encoder.Encode(Response{Success: false, Error: msg})
}

// promptUser displays questions using huh and collects answers
func (s *Server) promptUser(questions []Question) (map[string]string, error) {
	answers := make(map[string]string)

	for _, q := range questions {
		var answer string

		if q.MultiSelect {
			answer = s.promptMultiSelect(q)
		} else {
			answer = s.promptSelect(q)
		}

		answers[q.Question] = answer
	}

	return answers, nil
}

// promptSelect handles single-select questions
func (s *Server) promptSelect(q Question) string {
	options := make([]huh.Option[string], 0, len(q.Options)+1)
	for _, opt := range q.Options {
		options = append(options, huh.NewOption(opt.Label, opt.Label))
	}
	// Add "Other" option for free text input
	options = append(options, huh.NewOption("Other...", "__other__"))

	var answer string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(q.Header).
				Description(q.Question).
				Options(options...).
				Value(&answer),
		),
	)

	if err := form.Run(); err != nil {
		return ""
	}

	if answer == "__other__" {
		return s.promptFreeText(q.Header)
	}

	return answer
}

// promptMultiSelect handles multi-select questions
func (s *Server) promptMultiSelect(q Question) string {
	options := make([]huh.Option[string], 0, len(q.Options)+1)
	for _, opt := range q.Options {
		options = append(options, huh.NewOption(opt.Label, opt.Label))
	}
	// Add "Other" option for free text input
	options = append(options, huh.NewOption("Other...", "__other__"))

	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(q.Header).
				Description(q.Question).
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return ""
	}

	// Check if "Other" was selected
	hasOther := false
	filtered := make([]string, 0, len(selected))
	for _, sel := range selected {
		if sel == "__other__" {
			hasOther = true
		} else {
			filtered = append(filtered, sel)
		}
	}

	if hasOther {
		other := s.promptFreeText(q.Header)
		if other != "" {
			filtered = append(filtered, other)
		}
	}

	return strings.Join(filtered, ", ")
}

// promptFreeText prompts for free text input
func (s *Server) promptFreeText(title string) string {
	var text string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				Description("Enter your answer").
				Value(&text),
		),
	)

	if err := form.Run(); err != nil {
		return ""
	}

	return text
}
