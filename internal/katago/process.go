package katago

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/config"
	"github.com/dmmcquay/katago-mcp/internal/logging"
)

// Engine manages a KataGo process for analysis.
type Engine struct {
	config *config.KataGoConfig
	logger logging.ContextLogger

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr *bufio.Reader

	mu          sync.Mutex
	running     bool
	queryID     int
	pending     map[string]chan *Response
	stopCh      chan struct{}
	healthCheck chan struct{}
}

// Response represents a KataGo analysis response.
type Response struct {
	ID         string                 `json:"id"`
	TurnNumber int                    `json:"turnNumber"`
	MoveInfos  []MoveInfo             `json:"moveInfos"`
	RootInfo   RootInfo               `json:"rootInfo"`
	Error      interface{}            `json:"error,omitempty"` // Can be string or ErrorResponse
	Raw        map[string]interface{} `json:"-"`
}

// MoveInfo contains analysis for a single move.
type MoveInfo struct {
	Move       string   `json:"move"`
	Visits     int      `json:"visits"`
	Winrate    float64  `json:"winrate"`
	ScoreLead  float64  `json:"scoreLead"`
	ScoreMean  float64  `json:"scoreMean"`
	ScoreStdev float64  `json:"scoreStdev,omitempty"`
	Prior      float64  `json:"prior"` // Neural network's initial probability
	Utility    float64  `json:"utility,omitempty"`
	LCB        float64  `json:"lcb,omitempty"`
	PV         []string `json:"pv"`
	Order      int      `json:"order"`
}

// RootInfo contains information about the root position.
type RootInfo struct {
	Visits        int     `json:"visits"`
	Winrate       float64 `json:"winrate"`
	ScoreLead     float64 `json:"scoreLead"`
	ScoreMean     float64 `json:"scoreMean"`
	ScoreStdev    float64 `json:"scoreStdev"`
	CurrentPlayer string  `json:"currentPlayer"`
}

// ErrorResponse represents an error from KataGo.
type ErrorResponse struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// NewEngine creates a new KataGo engine.
func NewEngine(cfg *config.KataGoConfig, logger logging.ContextLogger) *Engine {
	return &Engine{
		config:      cfg,
		logger:      logger,
		pending:     make(map[string]chan *Response),
		stopCh:      make(chan struct{}),
		healthCheck: make(chan struct{}, 1),
	}
}

// Start starts the KataGo process.
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("engine already running")
	}

	// Build command arguments
	args := []string{"analysis"}
	if e.config.ConfigPath != "" {
		args = append(args, "-config", e.config.ConfigPath)
	}
	if e.config.ModelPath != "" {
		args = append(args, "-model", e.config.ModelPath)
	}

	// Create command
	e.cmd = exec.CommandContext(ctx, e.config.BinaryPath, args...) // #nosec G204 -- BinaryPath is validated configuration

	// Set up pipes
	stdin, err := e.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	e.stdin = stdin

	stdout, err := e.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	e.stdout = bufio.NewReader(stdout)

	stderr, err := e.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	e.stderr = bufio.NewReader(stderr)

	// Start the process
	if err := e.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start KataGo: %w", err)
	}

	e.running = true
	e.logger.Info("KataGo engine started",
		"binary", e.config.BinaryPath,
		"model", e.config.ModelPath,
		"threads", e.config.NumThreads,
	)

	// Start reader goroutines
	go e.readStdout()
	go e.readStderr()

	// Send initial configuration
	e.configure()

	// Start health check routine
	go e.healthCheckRoutine()

	return nil
}

// Stop stops the KataGo process.
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil
	}

	close(e.stopCh)
	e.running = false

	// Close stdin to signal shutdown
	if e.stdin != nil {
		_ = e.stdin.Close()
	}

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		if e.cmd != nil && e.cmd.Process != nil {
			done <- e.cmd.Wait()
		} else {
			done <- nil
		}
	}()

	select {
	case err := <-done:
		if err != nil && err.Error() != "signal: killed" {
			e.logger.Warn("KataGo process exited with error", "error", err)
		}
	case <-time.After(5 * time.Second):
		// Force kill if not exited
		if e.cmd != nil && e.cmd.Process != nil {
			_ = e.cmd.Process.Kill()
		}
	}

	// Cancel all pending queries
	for id, ch := range e.pending {
		ch <- &Response{
			ID:    id,
			Error: "engine stopped",
		}
		close(ch)
	}
	e.pending = make(map[string]chan *Response)

	e.logger.Info("KataGo engine stopped")
	return nil
}

// IsRunning returns whether the engine is running.
func (e *Engine) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}

// configure sends initial configuration commands to KataGo.
func (e *Engine) configure() {
	// The analysis engine doesn't need initial configuration
	// Configuration is passed via command line args and config file
	// Wait a bit for KataGo to fully start up before sending queries
	time.Sleep(500 * time.Millisecond)
}

// readStdout reads responses from KataGo.
func (e *Engine) readStdout() {
	for {
		select {
		case <-e.stopCh:
			return
		default:
			line, err := e.stdout.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					e.logger.Error("Failed to read stdout", "error", err)
				}
				return
			}

			if line == "" || line == "\n" {
				continue
			}

			// Parse JSON response
			var response Response
			if err := json.Unmarshal([]byte(line), &response); err != nil {
				e.logger.Warn("Failed to parse response", "line", line, "error", err)
				continue
			}
			e.logger.Debug("Received response", "id", response.ID, "hasError", response.Error != nil)

			// Also unmarshal into raw map for debugging
			_ = json.Unmarshal([]byte(line), &response.Raw)

			// Handle health check responses
			if response.ID == "health" {
				select {
				case e.healthCheck <- struct{}{}:
				default:
				}
				continue
			}

			// Skip startup responses that we're not waiting for
			if response.ID == "startup" {
				e.logger.Debug("Received startup response, ignoring")
				continue
			}

			// Send to waiting channel
			e.mu.Lock()
			if ch, ok := e.pending[response.ID]; ok {
				ch <- &response
				close(ch)
				delete(e.pending, response.ID)
			} else {
				e.logger.Warn("Received response for unknown query", "id", response.ID)
			}
			e.mu.Unlock()
		}
	}
}

// readStderr logs stderr output.
func (e *Engine) readStderr() {
	scanner := bufio.NewScanner(e.stderr)
	for scanner.Scan() {
		select {
		case <-e.stopCh:
			return
		default:
			line := scanner.Text()
			if line != "" {
				e.logger.Debug("KataGo stderr", "line", line)
			}
		}
	}
}

// healthCheckRoutine periodically checks if the engine is responsive.
func (e *Engine) healthCheckRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopCh:
			return
		case <-ticker.C:
			// Send a simple query to check if engine is responsive
			query := map[string]interface{}{
				"id":     "health",
				"action": "query_version",
			}

			data, _ := json.Marshal(query)
			e.mu.Lock()
			if e.running && e.stdin != nil {
				_, _ = fmt.Fprintf(e.stdin, "%s\n", data)
			}
			e.mu.Unlock()

			// Wait for response
			select {
			case <-e.healthCheck:
				// Healthy
			case <-time.After(5 * time.Second):
				e.logger.Error("KataGo health check timeout")
				// Could implement auto-restart here
			}
		}
	}
}

// sendQuery sends a query to KataGo and waits for response.
func (e *Engine) sendQuery(query map[string]interface{}) (*Response, error) {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return nil, fmt.Errorf("engine not running")
	}

	// Generate query ID
	e.queryID++
	id := fmt.Sprintf("q%d", e.queryID)
	query["id"] = id

	// Create response channel
	respCh := make(chan *Response, 1)
	e.pending[id] = respCh

	// Marshal and send query
	data, err := json.Marshal(query)
	if err != nil {
		delete(e.pending, id)
		e.mu.Unlock()
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	if _, err := fmt.Fprintf(e.stdin, "%s\n", data); err != nil {
		delete(e.pending, id)
		e.mu.Unlock()
		return nil, fmt.Errorf("failed to send query: %w", err)
	}
	e.logger.Debug("Sent query", "id", id, "query", string(data))
	e.mu.Unlock()

	// Wait for response with timeout
	select {
	case resp := <-respCh:
		if resp.Error != nil {
			switch v := resp.Error.(type) {
			case string:
				return nil, fmt.Errorf("KataGo error: %s", v)
			case map[string]interface{}:
				if msg, ok := v["message"].(string); ok {
					return nil, fmt.Errorf("KataGo error: %s", msg)
				}
			case *ErrorResponse:
				return nil, fmt.Errorf("KataGo error: %s", v.Message)
			}
			return nil, fmt.Errorf("KataGo error: %v", resp.Error)
		}
		return resp, nil
	case <-time.After(time.Duration(e.config.MaxTime*2) * time.Second):
		e.mu.Lock()
		delete(e.pending, id)
		e.mu.Unlock()
		e.logger.Error("Query timeout", "id", id, "timeout", e.config.MaxTime*2)
		return nil, fmt.Errorf("query timeout after %.1f seconds", e.config.MaxTime*2)
	}
}
