package katago

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/cache"
	"github.com/dmmcquay/katago-mcp/internal/config"
	"github.com/dmmcquay/katago-mcp/internal/logging"
	"github.com/dmmcquay/katago-mcp/internal/metrics"
)

// Engine manages a KataGo process for analysis.
type Engine struct {
	config     *config.KataGoConfig
	logger     logging.ContextLogger
	prometheus *metrics.PrometheusCollector
	cache      *cache.Manager

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
func NewEngine(cfg *config.KataGoConfig, logger logging.ContextLogger, cacheManager *cache.Manager) *Engine {
	return &Engine{
		config:      cfg,
		logger:      logger,
		prometheus:  metrics.NewPrometheusCollector(),
		cache:       cacheManager,
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

	// Record engine status
	version := "unknown"
	if detection, err := DetectKataGo(); err == nil && detection.Version != "" {
		version = detection.Version
	}
	if e.prometheus != nil {
		e.prometheus.RecordEngineStatus(true, version)
	}

	// Start reader goroutines
	go e.readStdout()
	go e.readStderr()

	// Send initial configuration
	e.configure()

	// Start health check routine
	go e.healthCheckRoutine()

	return nil
}

// Stop stops the KataGo process gracefully.
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil
	}

	e.logger.Info("Stopping KataGo engine gracefully")
	close(e.stopCh)
	e.running = false

	// Send quit command if possible
	if e.stdin != nil {
		// Try to send quit command first for graceful shutdown
		_, _ = e.stdin.Write([]byte(`{"id":"quit","action":"quit"}` + "\n"))
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
		if err != nil && err.Error() != "signal: killed" && err.Error() != "signal: terminated" {
			e.logger.Warn("KataGo process exited with error", "error", err)
		}
	case <-time.After(10 * time.Second):
		// Try SIGTERM first
		if e.cmd != nil && e.cmd.Process != nil {
			e.logger.Warn("KataGo not responding to quit, sending SIGTERM")
			_ = e.cmd.Process.Signal(syscall.SIGTERM)

			// Wait a bit more
			select {
			case <-done:
				// Process terminated
			case <-time.After(5 * time.Second):
				// Force kill if still not exited
				e.logger.Warn("KataGo still running, force killing")
				_ = e.cmd.Process.Kill()
			}
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
	if e.prometheus != nil {
		e.prometheus.RecordEngineStatus(false, "")
	}
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

// sendQueryWithCache sends a query to KataGo with caching support.
func (e *Engine) sendQueryWithCache(query map[string]interface{}) (*Response, error) {
	// Check if caching is enabled and this is a cacheable query
	if e.cache != nil && e.cache.IsEnabled() {
		// Generate cache key
		cacheKey, err := e.cache.CacheKey(query)
		if err == nil {
			// Try to get from cache
			if cached, ok := e.cache.Get(cacheKey); ok {
				if resp, ok := cached.(*Response); ok {
					e.logger.Debug("Cache hit", "key", cacheKey)
					if e.prometheus != nil {
						e.prometheus.RecordCacheHit()
					}
					return resp, nil
				}
			}
			if e.prometheus != nil {
				e.prometheus.RecordCacheMiss()
			}

			// Not in cache, execute query
			resp, queryErr := e.sendQuery(query)
			if queryErr != nil {
				return nil, queryErr
			}

			// Cache the successful response
			size := cache.EstimateSize(resp)
			e.cache.Put(cacheKey, resp, size)

			return resp, nil
		} else {
			e.logger.Warn("Failed to generate cache key", "error", err)
		}
	}

	// No caching, just send query
	return e.sendQuery(query)
}

// sendQuery sends a query to KataGo and waits for response.
func (e *Engine) sendQuery(query map[string]interface{}) (*Response, error) {
	start := time.Now()
	queryType := "unknown"
	if action, ok := query["action"].(string); ok {
		queryType = action
	}

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
		if e.prometheus != nil {
			e.prometheus.RecordEngineQuery(queryType, time.Since(start).Seconds())
		}
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

// Ping checks if the engine is responsive.
func (e *Engine) Ping(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		if e.prometheus != nil {
			e.prometheus.RecordEngineHealthCheck(false)
		}
		return fmt.Errorf("engine not running")
	}

	// Check if the process is still alive
	if e.cmd != nil && e.cmd.Process != nil {
		// Try to check process state without killing it
		// On Unix, sending signal 0 checks if process exists
		if err := e.cmd.Process.Signal(syscall.Signal(0)); err != nil {
			if e.prometheus != nil {
				e.prometheus.RecordEngineHealthCheck(false)
			}
			return fmt.Errorf("engine process not responding: %w", err)
		}
	} else {
		if e.prometheus != nil {
			e.prometheus.RecordEngineHealthCheck(false)
		}
		return fmt.Errorf("engine process not found")
	}

	if e.prometheus != nil {
		e.prometheus.RecordEngineHealthCheck(true)
	}
	return nil
}
