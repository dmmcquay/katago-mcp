# KataGo MCP API Documentation

This document provides comprehensive documentation for all MCP (Model Context Protocol) tools exposed by the KataGo MCP server.

## Table of Contents

- [Overview](#overview)
- [Tools](#tools)
  - [analyzePosition](#analyzeposition)
  - [getEngineStatus](#getenginestatus)
  - [startEngine](#startengine)
  - [stopEngine](#stopengine)
  - [findMistakes](#findmistakes)
  - [evaluateTerritory](#evaluateterritory)
  - [explainMove](#explainmove)
- [Data Types](#data-types)
- [Error Handling](#error-handling)
- [Examples](#examples)

## Overview

The KataGo MCP server provides tools for analyzing Go game positions using the KataGo AI engine. All tools follow the MCP protocol for tool invocation and response formatting.

### Connection

The server runs as a stdio-based MCP server. Connect to it using any MCP-compatible client.

```bash
# Start the server
katago-mcp

# Or with a config file
KATAGO_MCP_CONFIG=/path/to/config.json katago-mcp
```

## Tools

### analyzePosition

Analyzes a Go position using KataGo's neural network.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `sgf` | string | No* | SGF content to analyze |
| `position` | object | No* | Position object (see [Position](#position) type) |
| `moveNumber` | number | No | Move number to analyze (for SGF input). If not specified, analyzes the final position |
| `maxVisits` | number | No | Maximum visits for analysis (overrides default from config) |
| `maxTime` | number | No | Maximum time in seconds for analysis (overrides default) |
| `includePolicy` | boolean | No | Include policy network output (move probabilities) |
| `includeOwnership` | boolean | No | Include ownership map |
| `verbose` | boolean | No | Include more detailed output |

*Either `sgf` or `position` must be provided.

#### Response

Returns either formatted text (when `verbose=true` or neither `includePolicy` nor `includeOwnership` is set) or JSON.

**Text Response Example:**
```
=== Position Analysis ===
Current player: B
Visits: 1000
Win rate: 52.3%
Score: 1.5

=== Top Moves ===
 1. D4   visits:   400 win:55.0% score:+2.0
 2. Q16  visits:   300 win:52.0% score:+1.5
 3. D16  visits:   200 win:51.5% score:+1.2
```

**JSON Response Structure:**
```json
{
  "moveInfos": [
    {
      "move": "D4",
      "visits": 400,
      "winrate": 0.55,
      "scoreLead": 2.0,
      "scoreMean": 1.5,
      "prior": 0.15,
      "pv": ["D4", "Q16", "D16"]
    }
  ],
  "rootInfo": {
    "visits": 1000,
    "winrate": 0.523,
    "scoreLead": 1.5,
    "scoreMean": 1.5,
    "scoreStdev": 0.8,
    "currentPlayer": "B"
  },
  "policy": [0.001, 0.002, ...],
  "ownership": [-0.95, -0.90, ...]
}
```

### getEngineStatus

Gets the current status of the KataGo engine.

#### Parameters

None

#### Response

Text response indicating engine status.

**Example:**
```
KataGo engine status: running
```

### startEngine

Starts the KataGo engine if not already running.

#### Parameters

None

#### Response

Text response confirming engine start.

**Example:**
```
KataGo engine started successfully
```

### stopEngine

Stops the KataGo engine.

#### Parameters

None

#### Response

Text response confirming engine stop.

**Example:**
```
KataGo engine stopped successfully
```

### findMistakes

Analyzes a complete game to identify mistakes, blunders, and missed opportunities.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `sgf` | string | Yes | SGF content of the game to review |
| `blunderThreshold` | number | No | Win rate drop threshold for blunders (default: 0.15) |
| `mistakeThreshold` | number | No | Win rate drop threshold for mistakes (default: 0.05) |
| `inaccuracyThreshold` | number | No | Win rate drop threshold for inaccuracies (default: 0.02) |
| `maxVisits` | number | No | Maximum visits per position (default: from config) |

#### Response

Formatted markdown text with game review.

**Example:**
```markdown
# Game Review

## Summary
- Total moves: 250
- Black accuracy: 85.2%
- White accuracy: 87.5%
- Black mistakes/blunders: 5/2
- White mistakes/blunders: 4/1
- Estimated level: 5 dan

## Mistakes Found

### Move 45 (B)
- **Category**: Blunder
- **Played**: F3 (42.1% WR)
- **Better**: D4 (58.3% WR)
- **Win rate drop**: 16.2%
- This move loses control of the center. D4 would maintain better influence.
```

### evaluateTerritory

Evaluates territory ownership and control for the current position.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `sgf` | string | Yes | SGF content to analyze |
| `threshold` | number | No | Ownership threshold (0.0-1.0, default: 0.85) |
| `includeEstimates` | boolean | No | Include detailed point estimates |

#### Response

Returns either a visual territory map (text) or detailed JSON estimates.

**Text Response Example:**
```
Territory Estimate:
   A B C D E F G H J K L M N O P Q R S T
19 ● ● ● ● ● ● · · · · · · · ○ ○ ○ ○ ○ ○ 19
18 ● ● ● ● ● · · · · · · · · · ○ ○ ○ ○ ○ 18
17 ● ● ● ● · · · · · · · · · · · ○ ○ ○ ○ 17
...

Black: 45 points
White: 42 points
Neutral: 7 points

Estimated score: B+3.5
```

**JSON Response (when includeEstimates=true):**
```json
{
  "blackTerritory": 45,
  "whiteTerritory": 42,
  "neutralPoints": 7,
  "estimatedScore": 3.5,
  "ownership": [
    [-0.95, -0.90, -0.85, ...],
    ...
  ],
  "pointEstimates": {
    "A19": -0.95,
    "B19": -0.90,
    ...
  }
}
```

### explainMove

Provides detailed explanations for why a specific move is good or bad.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `sgf` | string | Yes | SGF content of the position |
| `move` | string | Yes | Move to explain (e.g., 'D4', 'Q16', 'pass') |
| `maxVisits` | number | No | Maximum visits for analysis |

#### Response

Formatted markdown text with move explanation.

**Example:**
```markdown
# Move Explanation: D4

This move claims the 4-4 point, establishing influence in the lower left corner while maintaining flexibility for future development.

## Statistics
- Win rate: 52.3%
- Score lead: +1.5 points
- Engine visits: 1000

## Strategic Analysis
- Board region: Corner
- Urgency: High
- Purpose: Influence, Territory potential

## Pros
- Establishes strong corner presence
- Flexible for both territory and influence
- Good balance with existing stones

## Cons
- Leaves 3-3 invasion available
- Doesn't directly pressure opponent

## Better Alternatives
- **D16** (53.1% WR): Maintains better whole-board balance
- **Q4** (52.8% WR): Creates symmetrical formation
```

## Data Types

### Position

Represents a Go board position.

```typescript
interface Position {
  rules: string;           // "japanese", "chinese", "korean", etc.
  boardXSize: number;      // Board width (usually 19)
  boardYSize: number;      // Board height (usually 19)
  moves: Move[];           // Sequence of moves
  initialStones?: Stone[]; // Handicap or setup stones
  initialPlayer?: string;  // "B" or "W"
  komi?: number;          // Komi value
}

interface Move {
  color: string;    // "B" or "W"
  location: string; // GTP format: "D4", "Q16", etc. Empty for pass
}

interface Stone {
  color: string;    // "B" or "W"
  location: string; // GTP format: "D4", "Q16", etc.
}
```

### Move Formats

All moves use GTP (Go Text Protocol) format:
- Column: A-T (skipping I)
- Row: 1-19 (or board size)
- Examples: "D4", "Q16", "A1", "T19"
- Pass move: "pass"

**Note:** SGF format (lowercase like "dd") is not accepted and will be rejected with an error.

## Error Handling

All tools return errors following the MCP error format:

```json
{
  "error": {
    "code": "INVALID_PARAMS",
    "message": "Invalid move format at index 0: dd"
  }
}
```

### Common Error Codes

- `INVALID_PARAMS`: Invalid parameters provided
- `INTERNAL_ERROR`: Server-side error (e.g., KataGo crash)
- `TIMEOUT`: Analysis timeout exceeded

### Common Error Scenarios

1. **Invalid SGF Format**
   - Malformed SGF syntax
   - Invalid game properties

2. **Invalid Move Format**
   - Using SGF format instead of GTP
   - Out of bounds coordinates
   - Invalid column letters (e.g., "I")

3. **Engine Not Running**
   - Attempting analysis when engine is stopped
   - Engine failed to start

4. **Resource Limits**
   - Analysis timeout
   - Maximum visits reached

## Examples

### Basic Position Analysis

```json
{
  "tool": "analyzePosition",
  "arguments": {
    "sgf": "(;GM[1]FF[4]SZ[19]KM[6.5]RE[B+R];B[dd];W[pp];B[dp];W[pd])",
    "verbose": true
  }
}
```

### Analyzing Specific Move Number

```json
{
  "tool": "analyzePosition",
  "arguments": {
    "sgf": "(;GM[1]FF[4]SZ[19]KM[6.5];B[dd];W[pp];B[dp];W[pd];B[nq];W[pn])",
    "moveNumber": 4,
    "includePolicy": true
  }
}
```

### Game Review

```json
{
  "tool": "findMistakes",
  "arguments": {
    "sgf": "(;GM[1]FF[4]...complete game...)",
    "blunderThreshold": 0.10,
    "mistakeThreshold": 0.05
  }
}
```

### Territory Evaluation

```json
{
  "tool": "evaluateTerritory",
  "arguments": {
    "sgf": "(;GM[1]FF[4]...game position...)",
    "threshold": 0.90,
    "includeEstimates": true
  }
}
```

### Move Explanation

```json
{
  "tool": "explainMove",
  "arguments": {
    "sgf": "(;GM[1]FF[4]SZ[19];B[dd];W[pp];B[dp];W[pd])",
    "move": "Q16",
    "maxVisits": 2000
  }
}
```

## Configuration

The server behavior can be configured through environment variables or a JSON config file. Key settings that affect API behavior:

- `KATAGO_MAX_VISITS`: Default maximum visits for analysis
- `KATAGO_MAX_TIME`: Default maximum time for analysis
- `KATAGO_CACHE_ENABLED`: Enable position caching
- `KATAGO_RATE_LIMIT`: Requests per second limit

See `config.example.json` for all available options.