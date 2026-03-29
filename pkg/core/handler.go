package core

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// Handler defines the interface for message handlers
type Handler interface {
	// CanHandle checks if this handler can process the message
	CanHandle(msg *Message) bool

	// Handle processes the message and returns a result
	Handle(ctx context.Context, msg *Message) (*HandlerResult, error)
}

// HandlerResult represents the result of handling a message
type HandlerResult struct {
	Success    bool   // Whether handling was successful
	Message    string // Result message
	ReplyTo    string // Optional: address to reply to
	ReplyBody  string // Optional: reply content
	ReplySubj  string // Optional: reply subject
	Action     string // Action to take on original message (seen/unseen/delete/flag)
	Output     string // Handler output (stdout/stderr)
	StatusCode int    // Exit status code
}

// ExecHandler executes an external command to handle messages
type ExecHandler struct {
	Command  string            // Command to execute
	Args     []string          // Command arguments
	Env      []string          // Environment variables
	Timeout  time.Duration     // Execution timeout
	WorkDir  string            // Working directory
	Input    func(*Message) (string, error) // Function to generate input from message
}

// NewExecHandler creates a new exec handler
func NewExecHandler(command string, args ...string) *ExecHandler {
	return &ExecHandler{
		Command: command,
		Args:    args,
		Timeout: 60 * time.Second,
	}
}

// CanHandle always returns true for exec handlers
func (h *ExecHandler) CanHandle(msg *Message) bool {
	return true
}

// Handle executes the command with the message
func (h *ExecHandler) Handle(ctx context.Context, msg *Message) (*HandlerResult, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, h.Timeout)
	defer cancel()

	// Prepare command
	cmd := exec.CommandContext(ctx, h.Command, h.Args...)

	// Set environment
	if len(h.Env) > 0 {
		cmd.Env = append(os.Environ(), h.Env...)
	}

	// Set working directory
	if h.WorkDir != "" {
		cmd.Dir = h.WorkDir
	}

	// Prepare input
	var input string
	if h.Input != nil {
		var err error
		input, err = h.Input(msg)
		if err != nil {
			return nil, fmt.Errorf("failed to generate input: %w", err)
		}
	} else {
		// Default: pass message as JSON
		msgJSON, err := json.Marshal(msg)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal message: %w", err)
		}
		input = string(msgJSON)
	}

	// Setup stdin/stdout/stderr
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()

	result := &HandlerResult{
		Output: stdout.String(),
	}

	// Check exit code
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				result.StatusCode = status.ExitStatus()
			}
		}
		result.Success = false
		result.Message = fmt.Sprintf("Command failed: %v\nStderr: %s", err, stderr.String())
		return result, nil
	}

	result.Success = true
	result.StatusCode = 0
	result.Message = "Command executed successfully"
	result.Output = stdout.String()

	return result, nil
}

// WithTimeout sets the execution timeout
func (h *ExecHandler) WithTimeout(timeout time.Duration) *ExecHandler {
	h.Timeout = timeout
	return h
}

// WithEnv sets environment variables
func (h *ExecHandler) WithEnv(env ...string) *ExecHandler {
	h.Env = env
	return h
}

// WithWorkDir sets the working directory
func (h *ExecHandler) WithWorkDir(dir string) *ExecHandler {
	h.WorkDir = dir
	return h
}

// WithInput sets a custom input function
func (h *ExecHandler) WithInput(fn func(*Message) (string, error)) *ExecHandler {
	h.Input = fn
	return h
}

// ScriptHandler executes a script file
type ScriptHandler struct {
	ScriptPath string
	Args       []string
	Timeout    time.Duration
}

// NewScriptHandler creates a new script handler
func NewScriptHandler(scriptPath string, args ...string) *ScriptHandler {
	return &ScriptHandler{
		ScriptPath: scriptPath,
		Args:       args,
		Timeout:    60 * time.Second,
	}
}

// CanHandle checks if the script file exists and is executable
func (h *ScriptHandler) CanHandle(msg *Message) bool {
	info, err := os.Stat(h.ScriptPath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// Handle executes the script
func (h *ScriptHandler) Handle(ctx context.Context, msg *Message) (*HandlerResult, error) {
	// Detect script type from extension
	var command string
	var args []string

	if strings.HasSuffix(h.ScriptPath, ".sh") {
		command = "bash"
		args = append([]string{h.ScriptPath}, h.Args...)
	} else if strings.HasSuffix(h.ScriptPath, ".py") {
		command = "python"
		args = append([]string{h.ScriptPath}, h.Args...)
	} else if strings.HasSuffix(h.ScriptPath, ".js") {
		command = "node"
		args = append([]string{h.ScriptPath}, h.Args...)
	} else {
		// Try to execute directly
		command = h.ScriptPath
		args = h.Args
	}

	execHandler := NewExecHandler(command, args...).WithTimeout(h.Timeout)
	return execHandler.Handle(ctx, msg)
}

// WithTimeout sets the execution timeout
func (h *ScriptHandler) WithTimeout(timeout time.Duration) *ScriptHandler {
	h.Timeout = timeout
	return h
}

// HTTPHandler sends messages to an HTTP endpoint (future implementation)
type HTTPHandler struct {
	URL     string
	Method  string
	Headers map[string]string
	Timeout time.Duration
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(url string) *HTTPHandler {
	return &HTTPHandler{
		URL:     url,
		Method:  "POST",
		Timeout: 30 * time.Second,
		Headers: make(map[string]string),
	}
}

// CanHandle always returns true for HTTP handlers
func (h *HTTPHandler) CanHandle(msg *Message) bool {
	return h.URL != ""
}

// Handle sends the message to the HTTP endpoint
func (h *HTTPHandler) Handle(ctx context.Context, msg *Message) (*HandlerResult, error) {
	// TODO: Implement HTTP request
	return &HandlerResult{
		Success: false,
		Message: "HTTP handler not yet implemented",
	}, nil
}

// HandlerEngine manages multiple handlers
type HandlerEngine struct {
	handlers []Handler
	Timeout  time.Duration
}

// NewHandlerEngine creates a new handler engine
func NewHandlerEngine() *HandlerEngine {
	return &HandlerEngine{
		handlers: make([]Handler, 0),
		Timeout:  60 * time.Second,
	}
}

// AddHandler adds a handler to the engine
func (e *HandlerEngine) AddHandler(handler Handler) {
	e.handlers = append(e.handlers, handler)
}

// Handle processes a message through the first matching handler
func (e *HandlerEngine) Handle(ctx context.Context, msg *Message) (*HandlerResult, error) {
	for _, handler := range e.handlers {
		if handler.CanHandle(msg) {
			// Add timeout if not already in context
			if _, hasDeadline := ctx.Deadline(); !hasDeadline && e.Timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, e.Timeout)
				defer cancel()
			}

			return handler.Handle(ctx, msg)
		}
	}

	return nil, fmt.Errorf("no handler found for message")
}

// HandleAll processes a message through all matching handlers
func (e *HandlerEngine) HandleAll(ctx context.Context, msg *Message) ([]*HandlerResult, error) {
	var results []*HandlerResult

	for _, handler := range e.handlers {
		if handler.CanHandle(msg) {
			result, err := handler.Handle(ctx, msg)
			if err != nil {
				return results, err
			}
			results = append(results, result)
		}
	}

	if len(results) == 0 {
		return results, fmt.Errorf("no handler found for message")
	}

	return results, nil
}

// ParseResultOutput parses handler output in various formats
func ParseResultOutput(output string) (map[string]interface{}, error) {
	// Try JSON first
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err == nil {
		return result, nil
	}

	// Try key=value format
	result = make(map[string]interface{})
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("unable to parse output")
	}

	return result, nil
}

// StreamHandler handles messages with streaming I/O
type StreamHandler struct {
	Process func(ctx context.Context, r io.Reader, w io.Writer) error
}

// NewStreamHandler creates a new stream handler
func NewStreamHandler(process func(ctx context.Context, r io.Reader, w io.Writer) error) *StreamHandler {
	return &StreamHandler{Process: process}
}

// CanHandle always returns true for stream handlers
func (h *StreamHandler) CanHandle(msg *Message) bool {
	return h.Process != nil
}

// Handle processes the message with streaming
func (h *StreamHandler) Handle(ctx context.Context, msg *Message) (*HandlerResult, error) {
	// Convert message to JSON
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create pipes for stdin/stdout
	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()

	var output bytes.Buffer

	// Process with streaming
	go func() {
		pw.Write(msgJSON)
		pw.Close()
	}()

	err = h.Process(ctx, pr, &output)

	return &HandlerResult{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "Stream processing completed"
		}(),
		Output: output.String(),
	}, nil
}
