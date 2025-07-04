# KataGo-MCP Testing Issues

This document tracks issues discovered during integration testing with ogs-mcp on 2025-07-03.

## Critical Issues

### 1. Missing KataGo Config File Path
**Issue**: KataGo engine starts but the config file path is empty in health check output
**Impact**: Causes "broken pipe" errors when trying to analyze positions
**Error**: `failed to send query: write |1: broken pipe`
**Health Check Output**:
```
KataGo Status:
  Binary: /opt/homebrew/bin/katago
  Version: KataGo v1.16.0
  Model: /Users/dmcquay/katago/models/model.bin
  Config:                    <-- Empty config path
```
**Found Config Location**: `/Users/dmcquay/venvs/system-venv/lib/python3.12/site-packages/katrain/KataGo/analysis_config.cfg`

### 2. Game Analysis Not Processing Full Games
**Issue**: `findMistakes` tool reports analyzing only 1 move regardless of actual game length
**Test Case 1**: 271-move game reported as "Total moves: 1"
**Test Case 2**: Similar issue with other games
**Symptom**: Returns 0% accuracy and 0/0 mistakes for all players in kyu-level games

### 3. Analysis Results Unrealistic
**Issue**: Analysis returns implausible results for amateur games
```
Total moves: 271
Black accuracy: 0.0%
White accuracy: 0.0%
Black mistakes/blunders: 0/0
White mistakes/blunders: 0/0
```
**Expected**: Kyu players should have numerous mistakes and blunders in a 271-move game

### 4. Position Format Unclear
**Issue**: The `position` parameter format for `analyzePosition` is undocumented
**Error**: `json: cannot unmarshal array into Go struct field Position.moves of type katago.Move`
**Attempted Format**:
```json
{
  "rules": "japanese",
  "boardXSize": 19,
  "boardYSize": 19,
  "komi": 6.5,
  "moves": [["B", "Q4"], ["W", "D4"], ["B", "R16"], ["W", "D17"], ["B", "P17"]]
}
```

### 5. Engine Process Stability
**Issue**: Multiple "broken pipe" errors suggest the KataGo process is crashing or not starting properly
**Errors**:
- `failed to send query: write |1: broken pipe`
- Engine reports as "running" in health check but can't accept commands

## Recommendations

1. **Config File Handling**:
   - Make config file path required in configuration
   - Validate config file exists and is readable on startup
   - Provide clear error if config is missing
   - Document common config file locations

2. **SGF Parsing**:
   - Debug why only first move is being analyzed
   - Add logging to show how many moves are being parsed
   - Test with simpler SGF files first

3. **Error Messages**:
   - Improve error messages for broken pipe issues
   - Add startup validation to ensure engine is ready
   - Log the actual KataGo command being executed

4. **Documentation**:
   - Document the exact format expected for `position` parameter
   - Provide working examples for all tools
   - Add troubleshooting guide for common issues

## Test Files

Two SGF files have been saved in this repository for testing:
- `test_game_76776999.sgf` - 271-move game (CoryLR vs Pete_Random)
- `test_game_76393597.sgf` - 232-move game (CoryLR vs jasmin tea)

These can be used to reproduce the analysis issues.