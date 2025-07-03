//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/cache"
	"github.com/dmmcquay/katago-mcp/internal/config"
	"github.com/dmmcquay/katago-mcp/internal/katago"
)

// TestAnalysisCache tests that the LRU cache is working correctly
func TestAnalysisCache(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create cache manager with small limits to test eviction
	cacheConfig := &config.CacheConfig{
		Enabled:      true,
		MaxItems:     5,
		MaxSizeBytes: 1024 * 1024, // 1MB
		TTLSeconds:   60,
	}
	cacheManager := cache.NewManager(cacheConfig, env.Logger)

	// Create engine with cache
	cfg := &config.KataGoConfig{
		BinaryPath: env.BinaryPath,
		ModelPath:  env.ModelPath,
		ConfigPath: env.ConfigPath,
		NumThreads: 2,
		MaxVisits:  50, // Lower visits for faster tests
		MaxTime:    10.0, // Increased timeout for Docker environment
	}

	engine := katago.NewEngine(cfg, env.Logger, cacheManager)

	// Start engine
	ctx := context.Background()
	if err := engine.Start(ctx); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	t.Cleanup(func() {
		if err := engine.Stop(); err != nil {
			t.Logf("Warning: failed to stop engine: %v", err)
		}
	})

	// Wait for engine to be ready
	time.Sleep(2 * time.Second)

	// Test position for caching
	testSGF := `(;GM[1]FF[4]SZ[19]KM[7.5];B[dd];W[pp];B[dp];W[pd])`

	// First analysis - should be a cache miss
	start1 := time.Now()
	result1, err := engine.AnalyzeSGF(ctx, testSGF, 0)
	if err != nil {
		t.Fatalf("First analysis failed: %v", err)
	}
	duration1 := time.Since(start1)

	// Check cache stats
	stats := cacheManager.Stats()
	if stats.Misses != 1 {
		t.Errorf("Expected 1 cache miss, got %d", stats.Misses)
	}
	if stats.Hits != 0 {
		t.Errorf("Expected 0 cache hits, got %d", stats.Hits)
	}

	// Second analysis of same position - should be a cache hit
	start2 := time.Now()
	result2, err := engine.AnalyzeSGF(ctx, testSGF, 0)
	if err != nil {
		t.Fatalf("Second analysis failed: %v", err)
	}
	duration2 := time.Since(start2)

	// Check cache stats again
	stats = cacheManager.Stats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 cache hit, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Expected 1 cache miss (unchanged), got %d", stats.Misses)
	}

	// Cache hit should be much faster
	if duration2 >= duration1 {
		t.Errorf("Cache hit (%v) was not faster than cache miss (%v)", duration2, duration1)
	}

	// Results should be identical
	if len(result1.MoveInfos) != len(result2.MoveInfos) {
		t.Errorf("Different number of moves: %d vs %d", len(result1.MoveInfos), len(result2.MoveInfos))
	}

	// Check that top move is the same
	if len(result1.MoveInfos) > 0 && len(result2.MoveInfos) > 0 {
		if result1.MoveInfos[0].Move != result2.MoveInfos[0].Move {
			t.Errorf("Different top moves: %s vs %s", result1.MoveInfos[0].Move, result2.MoveInfos[0].Move)
		}
	}

	t.Logf("Cache performance: miss=%v, hit=%v (%.1fx speedup)", duration1, duration2, float64(duration1)/float64(duration2))
	t.Logf("Cache stats: %+v", stats)
}

// TestCacheEviction tests that the cache properly evicts old entries
func TestCacheEviction(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create cache with very small limits
	cacheConfig := &config.CacheConfig{
		Enabled:      true,
		MaxItems:     3,         // Only 3 items
		MaxSizeBytes: 50 * 1024, // 50KB
		TTLSeconds:   60,
	}
	cacheManager := cache.NewManager(cacheConfig, env.Logger)

	cfg := &config.KataGoConfig{
		BinaryPath: env.BinaryPath,
		ModelPath:  env.ModelPath,
		ConfigPath: env.ConfigPath,
		NumThreads: 2,
		MaxVisits:  20, // Very low for speed
		MaxTime:    10.0, // Increased timeout for Docker environment
	}

	engine := katago.NewEngine(cfg, env.Logger, cacheManager)

	ctx := context.Background()
	if err := engine.Start(ctx); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	t.Cleanup(func() {
		if err := engine.Stop(); err != nil {
			t.Logf("Warning: failed to stop engine: %v", err)
		}
	})

	time.Sleep(2 * time.Second)

	// Analyze 5 different positions (more than cache can hold)
	positions := []string{
		`(;GM[1]FF[4]SZ[19]KM[7.5];B[dd])`,
		`(;GM[1]FF[4]SZ[19]KM[7.5];B[pp])`,
		`(;GM[1]FF[4]SZ[19]KM[7.5];B[dp])`,
		`(;GM[1]FF[4]SZ[19]KM[7.5];B[pd])`,
		`(;GM[1]FF[4]SZ[19]KM[7.5];B[cd])`,
	}

	// Analyze all positions
	for i, sgf := range positions {
		_, err := engine.AnalyzeSGF(ctx, sgf, 0)
		if err != nil {
			t.Fatalf("Analysis %d failed: %v", i, err)
		}
	}

	// Check cache stats
	stats := cacheManager.Stats()
	if stats.Items > 3 {
		t.Errorf("Cache has %d items, expected <= 3", stats.Items)
	}
	if stats.Evictions < 2 {
		t.Errorf("Expected at least 2 evictions, got %d", stats.Evictions)
	}

	// First position should have been evicted
	// Analyzing it again should be a miss
	_, err := engine.AnalyzeSGF(ctx, positions[0], 0)
	if err != nil {
		t.Fatalf("Re-analysis failed: %v", err)
	}

	newStats := cacheManager.Stats()
	if newStats.Misses <= stats.Misses {
		t.Error("Expected cache miss for evicted position")
	}

	t.Logf("Eviction test - Cache stats: %+v", newStats)
}

// TestCacheTTL tests that cache entries expire after TTL
func TestCacheTTL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TTL test in short mode")
	}

	env := SetupTestEnvironment(t)

	// Create cache with very short TTL
	cacheConfig := &config.CacheConfig{
		Enabled:      true,
		MaxItems:     10,
		MaxSizeBytes: 1024 * 1024,
		TTLSeconds:   2, // 2 second TTL
	}
	cacheManager := cache.NewManager(cacheConfig, env.Logger)

	cfg := &config.KataGoConfig{
		BinaryPath: env.BinaryPath,
		ModelPath:  env.ModelPath,
		ConfigPath: env.ConfigPath,
		NumThreads: 2,
		MaxVisits:  20,
		MaxTime:    10.0, // Increased timeout for Docker environment
	}

	engine := katago.NewEngine(cfg, env.Logger, cacheManager)

	ctx := context.Background()
	if err := engine.Start(ctx); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	t.Cleanup(func() {
		if err := engine.Stop(); err != nil {
			t.Logf("Warning: failed to stop engine: %v", err)
		}
	})

	time.Sleep(2 * time.Second)

	testSGF := `(;GM[1]FF[4]SZ[19]KM[7.5];B[dd];W[pp])`

	// First analysis
	_, err := engine.AnalyzeSGF(ctx, testSGF, 0)
	if err != nil {
		t.Fatalf("First analysis failed: %v", err)
	}

	// Immediate re-analysis should hit cache
	_, err = engine.AnalyzeSGF(ctx, testSGF, 0)
	if err != nil {
		t.Fatalf("Second analysis failed: %v", err)
	}

	stats1 := cacheManager.Stats()
	if stats1.Hits != 1 {
		t.Errorf("Expected 1 cache hit, got %d", stats1.Hits)
	}

	// Wait for TTL to expire
	time.Sleep(3 * time.Second)

	// Analysis after TTL should be a miss
	_, err = engine.AnalyzeSGF(ctx, testSGF, 0)
	if err != nil {
		t.Fatalf("Third analysis failed: %v", err)
	}

	stats2 := cacheManager.Stats()
	if stats2.Misses != stats1.Misses+1 {
		t.Error("Expected cache miss after TTL expiration")
	}

	t.Logf("TTL test - Initial stats: %+v, After TTL: %+v", stats1, stats2)
}

// TestCacheDisabled tests that caching can be disabled
func TestCacheDisabled(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create with caching disabled
	cacheConfig := &config.CacheConfig{
		Enabled: false,
	}
	cacheManager := cache.NewManager(cacheConfig, env.Logger)

	cfg := &config.KataGoConfig{
		BinaryPath: env.BinaryPath,
		ModelPath:  env.ModelPath,
		ConfigPath: env.ConfigPath,
		NumThreads: 2,
		MaxVisits:  20,
		MaxTime:    10.0, // Increased timeout for Docker environment
	}

	engine := katago.NewEngine(cfg, env.Logger, cacheManager)

	ctx := context.Background()
	if err := engine.Start(ctx); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	t.Cleanup(func() {
		if err := engine.Stop(); err != nil {
			t.Logf("Warning: failed to stop engine: %v", err)
		}
	})

	time.Sleep(2 * time.Second)

	testSGF := `(;GM[1]FF[4]SZ[19]KM[7.5];B[dd])`

	// Analyze twice
	_, err := engine.AnalyzeSGF(ctx, testSGF, 0)
	if err != nil {
		t.Fatalf("First analysis failed: %v", err)
	}

	_, err = engine.AnalyzeSGF(ctx, testSGF, 0)
	if err != nil {
		t.Fatalf("Second analysis failed: %v", err)
	}

	// Should have no cache activity
	stats := cacheManager.Stats()
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Errorf("Cache should be inactive but got hits=%d, misses=%d", stats.Hits, stats.Misses)
	}
}
