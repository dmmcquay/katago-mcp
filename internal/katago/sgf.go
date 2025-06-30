package katago

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Position represents a board position for KataGo analysis
type Position struct {
	// Board state
	Rules      string     `json:"rules"`
	BoardXSize int        `json:"boardXSize"`
	BoardYSize int        `json:"boardYSize"`
	InitialStones []Stone `json:"initialStones,omitempty"`
	Moves      []Move     `json:"moves"`
	InitialPlayer string  `json:"initialPlayer,omitempty"`
	Komi       float64    `json:"komi"`
}

// Stone represents a stone on the board
type Stone struct {
	Color    string `json:"color"`
	Location string `json:"location"`
}

// Move represents a move in the game
type Move struct {
	Color    string `json:"color"`
	Location string `json:"location"`
}

// SGFParser parses SGF files
type SGFParser struct {
	content string
	index   int
}

// NewSGFParser creates a new SGF parser
func NewSGFParser(content string) *SGFParser {
	return &SGFParser{
		content: strings.TrimSpace(content),
		index:   0,
	}
}

// Parse parses the SGF and returns a Position
func (p *SGFParser) Parse() (*Position, error) {
	// Skip to first '('
	if !p.skipTo('(') {
		return nil, fmt.Errorf("invalid SGF: no opening parenthesis")
	}
	p.index++ // Skip '('

	// Parse game tree
	position := &Position{
		Rules:      "chinese", // Default
		BoardXSize: 19,        // Default
		BoardYSize: 19,        // Default
		Moves:      []Move{},
	}

	// Parse nodes
	for p.index < len(p.content) {
		p.skipWhitespace()
		
		if p.index >= len(p.content) {
			break
		}

		if p.content[p.index] == ')' {
			break
		}

		if p.content[p.index] == ';' {
			p.index++
			if err := p.parseNode(position); err != nil {
				return nil, err
			}
		} else if p.content[p.index] == '(' {
			// Skip variations for now
			p.skipVariation()
		} else {
			p.index++
		}
	}

	// Set initial player if not specified
	if position.InitialPlayer == "" && len(position.Moves) > 0 {
		position.InitialPlayer = position.Moves[0].Color
	}

	return position, nil
}

// parseNode parses a single SGF node
func (p *SGFParser) parseNode(position *Position) error {
	for p.index < len(p.content) {
		p.skipWhitespace()

		if p.index >= len(p.content) || p.content[p.index] == ';' || p.content[p.index] == ')' || p.content[p.index] == '(' {
			break
		}

		// Parse property
		prop, values, err := p.parseProperty()
		if err != nil {
			return err
		}

		// Handle properties
		switch prop {
		case "B", "W":
			color := "b"
			if prop == "W" {
				color = "w"
			}
			if len(values) > 0 {
				if values[0] == "" || values[0] == "tt" { // Empty or tt = pass
					position.Moves = append(position.Moves, Move{
						Color:    color,
						Location: "", // Empty location indicates pass
					})
				} else {
					position.Moves = append(position.Moves, Move{
						Color:    color,
						Location: p.sgfToKataGo(values[0]),
					})
				}
			}

		case "AB": // Add black stones
			for _, v := range values {
				if v != "" {
					position.InitialStones = append(position.InitialStones, Stone{
						Color:    "b",
						Location: p.sgfToKataGo(v),
					})
				}
			}

		case "AW": // Add white stones
			for _, v := range values {
				if v != "" {
					position.InitialStones = append(position.InitialStones, Stone{
						Color:    "w",
						Location: p.sgfToKataGo(v),
					})
				}
			}

		case "SZ": // Board size
			if len(values) > 0 {
				size, err := strconv.Atoi(values[0])
				if err == nil {
					position.BoardXSize = size
					position.BoardYSize = size
				}
			}

		case "KM": // Komi
			if len(values) > 0 {
				komi, err := strconv.ParseFloat(values[0], 64)
				if err == nil {
					position.Komi = komi
				}
			}

		case "RU": // Rules
			if len(values) > 0 {
				rules := strings.ToLower(values[0])
				if strings.Contains(rules, "japan") {
					position.Rules = "japanese"
				} else if strings.Contains(rules, "korea") {
					position.Rules = "korean"
				} else if strings.Contains(rules, "aga") {
					position.Rules = "aga"
				} else if strings.Contains(rules, "new zealand") {
					position.Rules = "new_zealand"
				} else {
					position.Rules = "chinese"
				}
			}

		case "PL": // Player to play
			if len(values) > 0 {
				if values[0] == "B" {
					position.InitialPlayer = "b"
				} else if values[0] == "W" {
					position.InitialPlayer = "w"
				}
			}
		}
	}

	return nil
}

// parseProperty parses a property and its values
func (p *SGFParser) parseProperty() (string, []string, error) {
	// Parse property name
	propStart := p.index
	for p.index < len(p.content) && p.content[p.index] >= 'A' && p.content[p.index] <= 'Z' {
		p.index++
	}

	if p.index == propStart {
		return "", nil, fmt.Errorf("expected property name at position %d", p.index)
	}

	prop := p.content[propStart:p.index]
	values := []string{}

	// Parse values
	for p.index < len(p.content) {
		p.skipWhitespace()
		
		if p.index >= len(p.content) || p.content[p.index] != '[' {
			break
		}

		p.index++ // Skip '['
		valueStart := p.index
		escaped := false

		for p.index < len(p.content) {
			if p.content[p.index] == '\\' && !escaped {
				escaped = true
			} else if p.content[p.index] == ']' && !escaped {
				break
			} else {
				escaped = false
			}
			p.index++
		}

		if p.index >= len(p.content) {
			return "", nil, fmt.Errorf("unclosed property value")
		}

		value := p.content[valueStart:p.index]
		// Unescape
		value = strings.ReplaceAll(value, "\\]", "]")
		value = strings.ReplaceAll(value, "\\[", "[")
		value = strings.ReplaceAll(value, "\\\\", "\\")
		values = append(values, value)

		p.index++ // Skip ']'
	}

	// Properties must have at least one value
	if len(values) == 0 {
		return "", nil, fmt.Errorf("property %s must have at least one value", prop)
	}

	return prop, values, nil
}

// sgfToKataGo converts SGF coordinates to KataGo format
func (p *SGFParser) sgfToKataGo(coord string) string {
	if len(coord) != 2 {
		return coord
	}

	x := coord[0] - 'a'
	y := coord[1] - 'a'

	// KataGo uses A1 style (A-T, skipping I)
	col := ""
	if x < 8 {
		col = string('A' + x)
	} else {
		col = string('A' + x + 1) // Skip 'I'
	}

	// KataGo counts from bottom (assuming 19x19 for now, will be fixed when we know board size)
	row := fmt.Sprintf("%d", 19-int(y))

	return col + row
}

// skipWhitespace skips whitespace characters
func (p *SGFParser) skipWhitespace() {
	for p.index < len(p.content) && (p.content[p.index] == ' ' || p.content[p.index] == '\t' || 
		p.content[p.index] == '\n' || p.content[p.index] == '\r') {
		p.index++
	}
}

// skipTo skips to the specified character
func (p *SGFParser) skipTo(ch byte) bool {
	for p.index < len(p.content) {
		if p.content[p.index] == ch {
			return true
		}
		p.index++
	}
	return false
}

// skipVariation skips a variation subtree
func (p *SGFParser) skipVariation() {
	depth := 0
	for p.index < len(p.content) {
		if p.content[p.index] == '(' {
			depth++
		} else if p.content[p.index] == ')' {
			depth--
			if depth == 0 {
				p.index++
				break
			}
		}
		p.index++
	}
}

// ValidatePosition validates a position for KataGo analysis
func ValidatePosition(pos *Position) error {
	// Validate board size
	if pos.BoardXSize < 2 || pos.BoardXSize > 25 || pos.BoardYSize < 2 || pos.BoardYSize > 25 {
		return fmt.Errorf("invalid board size: %dx%d", pos.BoardXSize, pos.BoardYSize)
	}

	// Validate rules
	validRules := map[string]bool{
		"chinese": true, "japanese": true, "korean": true,
		"aga": true, "new_zealand": true, "tromp-taylor": true,
	}
	if !validRules[pos.Rules] {
		return fmt.Errorf("invalid rules: %s", pos.Rules)
	}

	// Validate moves
	coordPattern := regexp.MustCompile(`^[A-T]\d{1,2}$`)
	for i, move := range pos.Moves {
		if move.Color != "b" && move.Color != "w" {
			return fmt.Errorf("invalid color in move %d: %s", i, move.Color)
		}
		if move.Location != "" && !coordPattern.MatchString(move.Location) {
			return fmt.Errorf("invalid location in move %d: %s", i, move.Location)
		}
	}

	// Validate initial stones
	for i, stone := range pos.InitialStones {
		if stone.Color != "b" && stone.Color != "w" {
			return fmt.Errorf("invalid color in initial stone %d: %s", i, stone.Color)
		}
		if !coordPattern.MatchString(stone.Location) {
			return fmt.Errorf("invalid location in initial stone %d: %s", i, stone.Location)
		}
	}

	return nil
}