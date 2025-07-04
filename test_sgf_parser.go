package main

import (
	"fmt"
	"io/ioutil"
	"log"
	
	"github.com/dmmcquay/katago-mcp/internal/katago"
)

func main() {
	// Read SGF file
	sgfContent, err := ioutil.ReadFile("test_game_76776999.sgf")
	if err != nil {
		log.Fatal(err)
	}
	
	// Parse SGF
	parser := katago.NewSGFParser(string(sgfContent))
	position, err := parser.Parse()
	if err != nil {
		log.Fatal("Parse error:", err)
	}
	
	fmt.Printf("Board size: %dx%d\n", position.BoardXSize, position.BoardYSize)
	fmt.Printf("Rules: %s\n", position.Rules)
	fmt.Printf("Komi: %.1f\n", position.Komi)
	fmt.Printf("Initial stones: %d\n", len(position.InitialStones))
	fmt.Printf("Moves parsed: %d\n", len(position.Moves))
	
	// Show first 10 moves
	fmt.Println("\nFirst 10 moves:")
	for i := 0; i < 10 && i < len(position.Moves); i++ {
		fmt.Printf("Move %d: %s %s\n", i+1, position.Moves[i].Color, position.Moves[i].Location)
	}
	
	// Show last 5 moves
	if len(position.Moves) > 5 {
		fmt.Println("\nLast 5 moves:")
		for i := len(position.Moves)-5; i < len(position.Moves); i++ {
			fmt.Printf("Move %d: %s %s\n", i+1, position.Moves[i].Color, position.Moves[i].Location)
		}
	}
}