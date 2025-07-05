package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/config"
	"github.com/dmmcquay/katago-mcp/internal/katago"
	"github.com/dmmcquay/katago-mcp/internal/logging"
)

func main() {
	// Read SGF file
	sgfFile := "test_game_76776999.sgf"
	sgfContent, err := os.ReadFile(sgfFile)
	if err != nil {
		log.Fatalf("Failed to read SGF file: %v", err)
	}

	fmt.Printf("Testing with SGF file: %s\n", sgfFile)
	fmt.Printf("SGF length: %d characters\n", len(sgfContent))

	// Create logger using the factory
	logConfig := &logging.Config{
		Level:   "debug",
		Format:  logging.FormatJSON,
		Service: "test-direct",
		Version: "1.0",
	}
	logger := logging.NewLoggerFromConfig(logConfig)

	// Create config
	cfg := &config.KataGoConfig{
		BinaryPath: "/opt/homebrew/bin/katago",
		ModelPath:  "/Users/dmcquay/venvs/system-venv/lib/python3.12/site-packages/katrain/models/kata1-b18c384nbt-s9996604416-d4316597426.bin.gz",
		ConfigPath: "/Users/dmcquay/venvs/system-venv/lib/python3.12/site-packages/katrain/KataGo/analysis_config.cfg",
		NumThreads: 4,
		MaxVisits:  100,
		MaxTime:    30.0,
	}

	// Create engine
	engine := katago.NewEngine(cfg, logger, nil)

	// Start engine
	ctx := context.Background()
	fmt.Println("\nStarting KataGo engine...")
	if startErr := engine.Start(ctx); startErr != nil {
		log.Fatalf("Failed to start engine: %v", startErr)
	}
	defer func() {
		if err := engine.Stop(); err != nil {
			log.Printf("Failed to stop engine: %v", err)
		}
	}()

	fmt.Println("Engine started successfully!")

	// Review the game
	fmt.Println("\nAnalyzing game for mistakes...")
	startTime := time.Now()

	thresholds := &katago.MistakeThresholds{
		Blunder:       0.15,
		Mistake:       0.05,
		Inaccuracy:    0.02,
		MinimumVisits: 50,
	}

	review, err := engine.ReviewGame(ctx, string(sgfContent), thresholds)
	if err != nil {
		// Don't use log.Fatalf here because it would skip the defer
		log.Printf("Failed to review game: %v", err)
		return
	}

	elapsed := time.Since(startTime)
	fmt.Printf("Analysis completed in %.1f seconds\n", elapsed.Seconds())

	// Display results
	fmt.Printf("\n=== ANALYSIS RESULTS ===\n")
	fmt.Printf("Total moves analyzed: %d\n", review.Summary.TotalMoves)

	switch review.Summary.TotalMoves {
	case 271:
		fmt.Println("✅ SUCCESS: Correctly analyzed all 271 moves!")
	case 1:
		fmt.Println("❌ FAILURE: Bug still present - only analyzed 1 move")
	default:
		fmt.Printf("⚠️  Analyzed %d moves (expected 271)\n", review.Summary.TotalMoves)
	}

	fmt.Printf("\nBlack accuracy: %.1f%%\n", review.Summary.BlackAccuracy)
	fmt.Printf("White accuracy: %.1f%%\n", review.Summary.WhiteAccuracy)
	fmt.Printf("Black mistakes: %d\n", review.Summary.BlackMistakes)
	fmt.Printf("White mistakes: %d\n", review.Summary.WhiteMistakes)
	fmt.Printf("Total mistakes found: %d\n", len(review.Mistakes))

	// Show first few mistakes
	fmt.Printf("\nFirst few mistakes:\n")
	for i := range review.Mistakes {
		mistake := &review.Mistakes[i]
		if i >= 5 {
			fmt.Printf("... and %d more mistakes\n", len(review.Mistakes)-5)
			break
		}
		fmt.Printf("  Move %d (%s): %s played %s (best: %s, drop: %.1f%%)\n",
			mistake.MoveNumber, mistake.Color, mistake.Category,
			mistake.PlayedMove, mistake.BestMove, mistake.WinrateDrop*100)
	}
}
