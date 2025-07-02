package ratelimit

import (
	"fmt"
	"sync"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/config"
	"github.com/dmmcquay/katago-mcp/internal/logging"
)

// Limiter manages rate limiting for the MCP server.
type Limiter struct {
	logger       logging.ContextLogger
	config       *config.RateLimitConfig
	globalBucket *TokenBucket
	toolBuckets  map[string]*TokenBucket
	clientLimits map[string]*clientRateLimit
	mu           sync.RWMutex
}

// clientRateLimit tracks rate limits for a specific client.
type clientRateLimit struct {
	globalBucket *TokenBucket
	toolBuckets  map[string]*TokenBucket
	lastSeen     time.Time
}

// NewLimiter creates a new rate limiter.
func NewLimiter(cfg *config.RateLimitConfig, logger logging.ContextLogger) *Limiter {
	if cfg == nil || !cfg.Enabled {
		return nil
	}

	// Calculate tokens per second from requests per minute
	tokensPerSecond := float64(cfg.RequestsPerMin) / 60.0

	limiter := &Limiter{
		logger:       logger,
		config:       cfg,
		globalBucket: NewTokenBucket(cfg.BurstSize, tokensPerSecond),
		toolBuckets:  make(map[string]*TokenBucket),
		clientLimits: make(map[string]*clientRateLimit),
	}

	// Initialize per-tool buckets
	for tool, limit := range cfg.PerToolLimits {
		toolTokensPerSecond := float64(limit) / 60.0
		// Use same burst ratio as global limit
		burstSize := (cfg.BurstSize * limit) / cfg.RequestsPerMin
		if burstSize < 1 {
			burstSize = 1
		}
		limiter.toolBuckets[tool] = NewTokenBucket(burstSize, toolTokensPerSecond)
	}

	// Start cleanup goroutine for stale client limits
	go limiter.cleanupStaleClients()

	return limiter
}

// Allow checks if a request is allowed under the rate limits.
func (l *Limiter) Allow(clientID, toolName string) (bool, error) {
	if l == nil {
		return true, nil // No rate limiting configured
	}

	// Check global limit first
	if !l.globalBucket.Allow(1) {
		l.logger.Warn("Global rate limit exceeded",
			"client", clientID,
			"tool", toolName,
		)
		return false, fmt.Errorf("global rate limit exceeded")
	}

	// Check tool-specific limit if configured
	l.mu.RLock()
	toolBucket, hasToolLimit := l.toolBuckets[toolName]
	l.mu.RUnlock()

	if hasToolLimit && !toolBucket.Allow(1) {
		// Return the token to global bucket since we're rejecting
		l.globalBucket.Allow(-1) // Add token back
		
		l.logger.Warn("Tool rate limit exceeded",
			"client", clientID,
			"tool", toolName,
		)
		return false, fmt.Errorf("rate limit exceeded for tool %s", toolName)
	}

	// Check client-specific limits
	if clientID != "" {
		allowed, err := l.checkClientLimit(clientID, toolName)
		if !allowed {
			// Return tokens since we're rejecting
			l.globalBucket.Allow(-1)
			if hasToolLimit {
				toolBucket.Allow(-1)
			}
			return false, err
		}
	}

	return true, nil
}

// checkClientLimit checks per-client rate limits.
func (l *Limiter) checkClientLimit(clientID, toolName string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	client, exists := l.clientLimits[clientID]
	if !exists {
		// Create new client limit tracking
		tokensPerSecond := float64(l.config.RequestsPerMin) / 60.0
		client = &clientRateLimit{
			globalBucket: NewTokenBucket(l.config.BurstSize, tokensPerSecond),
			toolBuckets:  make(map[string]*TokenBucket),
			lastSeen:     time.Now(),
		}
		l.clientLimits[clientID] = client
	}

	client.lastSeen = time.Now()

	// Check client's global limit
	if !client.globalBucket.Allow(1) {
		l.logger.Warn("Client rate limit exceeded",
			"client", clientID,
			"tool", toolName,
		)
		return false, fmt.Errorf("client rate limit exceeded")
	}

	// Check client's per-tool limit if configured
	if limit, hasLimit := l.config.PerToolLimits[toolName]; hasLimit {
		toolBucket, exists := client.toolBuckets[toolName]
		if !exists {
			toolTokensPerSecond := float64(limit) / 60.0
			burstSize := (l.config.BurstSize * limit) / l.config.RequestsPerMin
			if burstSize < 1 {
				burstSize = 1
			}
			toolBucket = NewTokenBucket(burstSize, toolTokensPerSecond)
			client.toolBuckets[toolName] = toolBucket
		}

		if !toolBucket.Allow(1) {
			// Return token to client's global bucket
			client.globalBucket.Allow(-1)
			
			l.logger.Warn("Client tool rate limit exceeded",
				"client", clientID,
				"tool", toolName,
			)
			return false, fmt.Errorf("client rate limit exceeded for tool %s", toolName)
		}
	}

	return true, nil
}

// Wait returns the duration to wait before the request would be allowed.
func (l *Limiter) Wait(clientID, toolName string) time.Duration {
	if l == nil {
		return 0
	}

	// For simplicity, just check global bucket wait time
	// In a more sophisticated implementation, we'd check all applicable limits
	return l.globalBucket.Wait(1)
}

// Reset resets all rate limit buckets to full capacity.
func (l *Limiter) Reset() {
	if l == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.globalBucket.Reset()
	
	for _, bucket := range l.toolBuckets {
		bucket.Reset()
	}
	
	// Reset all client limits
	for _, client := range l.clientLimits {
		client.globalBucket.Reset()
		for _, bucket := range client.toolBuckets {
			bucket.Reset()
		}
	}
}

// cleanupStaleClients periodically removes client rate limit tracking for inactive clients.
func (l *Limiter) cleanupStaleClients() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		staleTimeout := 30 * time.Minute

		for clientID, client := range l.clientLimits {
			if now.Sub(client.lastSeen) > staleTimeout {
				delete(l.clientLimits, clientID)
				l.logger.Debug("Removed stale client rate limit tracking",
					"client", clientID,
				)
			}
		}
		l.mu.Unlock()
	}
}

// GetStatus returns the current status of rate limits for monitoring.
func (l *Limiter) GetStatus() map[string]interface{} {
	if l == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	status := map[string]interface{}{
		"enabled":         true,
		"requestsPerMin":  l.config.RequestsPerMin,
		"burstSize":       l.config.BurstSize,
		"globalTokens":    l.globalBucket.Tokens(),
		"activeClients":   len(l.clientLimits),
		"toolLimits":      make(map[string]interface{}),
	}

	// Add per-tool status
	for tool, bucket := range l.toolBuckets {
		status["toolLimits"].(map[string]interface{})[tool] = map[string]interface{}{
			"limit":  l.config.PerToolLimits[tool],
			"tokens": bucket.Tokens(),
		}
	}

	return status
}