# KataGo Metrics Enhancement Proposal

## Overview

This proposal outlines enhancements to expose additional KataGo engine metrics through the MCP server, providing users with deeper insights into the AI's analysis process.

## Current State

Currently exposed metrics:
- Basic analysis results (win rates, score estimates, best moves)
- Policy network output (move probabilities)
- Ownership maps (territory predictions)

## Proposed Enhancements

### 1. Raw Neural Network Outputs

Add raw neural network predictions before search to understand the difference between instant evaluation and deep analysis:

```go
type RootInfo struct {
    // Existing fields...
    
    // Raw neural network outputs (before search)
    RawWinrate      float64 `json:"rawWinrate,omitempty"`      // NN's instant win rate
    RawScoreLead    float64 `json:"rawScoreLead,omitempty"`    // NN's instant score estimate
    RawScoreStdev   float64 `json:"rawScoreStdev,omitempty"`   // NN's uncertainty estimate
    RawScoreSelfplay float64 `json:"rawScoreSelfplay,omitempty"` // Self-play score estimate
}
```

**Use cases:**
- Compare instant vs. searched evaluations
- Measure search effectiveness
- Debug unusual position evaluations

### 2. Engine Performance Metrics

Track and expose engine performance data:

```go
type AnalysisResult struct {
    // Existing fields...
    
    // Performance metrics
    QueryTime       float64 `json:"queryTime,omitempty"`       // Total processing time (seconds)
    NeuralNetEvals  int     `json:"neuralNetEvals,omitempty"`  // Number of NN evaluations
    SearchEfficiency float64 `json:"searchEfficiency,omitempty"` // Visits per second
}
```

**Use cases:**
- Monitor engine performance
- Optimize analysis parameters
- Debug slow queries

### 3. Search Tree Statistics

Expose detailed search tree information:

```go
type SearchTreeStats struct {
    TreeDepth       int     `json:"treeDepth"`       // Maximum depth explored
    BranchingFactor float64 `json:"branchingFactor"` // Average branching factor
    NodesExpanded   int     `json:"nodesExpanded"`   // Total nodes in tree
    CriticalPath    []Move  `json:"criticalPath"`    // Principal variation
}
```

### 4. Neural Network Confidence Metrics

Add confidence and uncertainty measurements:

```go
type ConfidenceMetrics struct {
    PolicyEntropy    float64 `json:"policyEntropy"`    // Uncertainty in move selection
    ValueConfidence  float64 `json:"valueConfidence"`  // Confidence in position evaluation
    ScoreVariance    float64 `json:"scoreVariance"`    // Variance in score predictions
    ConsistencyScore float64 `json:"consistencyScore"` // Consistency across evaluations
}
```

### 5. Advanced Territory Analysis

Enhanced ownership and territory predictions:

```go
type TerritoryAnalysis struct {
    Ownership         [][]float64 `json:"ownership"`         // Existing
    OwnershipStdev    [][]float64 `json:"ownershipStdev"`    // Uncertainty per point
    TerritoryVolatility [][]float64 `json:"territoryVolatility"` // How likely to change
    EyeSpace          [][]bool    `json:"eyeSpace"`          // Identified eye points
    DeadStones        []string    `json:"deadStones"`        // List of dead groups
}
```

## Implementation Plan

### Phase 1: Core Metrics
1. Add raw neural network outputs to RootInfo
2. Implement performance metrics collection
3. Update MCP tools to expose new fields
4. Add documentation

### Phase 2: Advanced Features
1. Implement search tree statistics
2. Add confidence metrics
3. Enhance territory analysis
4. Create visualization tools

### Phase 3: Integration
1. Update all analysis tools to use new metrics
2. Add filtering options for metric selection
3. Implement metric aggregation for game analysis
4. Performance optimization

## API Changes

### Updated analyzePosition Response

```json
{
  "rootInfo": {
    "winrate": 0.523,
    "scoreLead": 1.5,
    "rawWinrate": 0.498,
    "rawScoreLead": 0.8,
    "visits": 1000,
    "neuralNetEvals": 1000,
    "queryTime": 2.34
  },
  "moveInfos": [...],
  "ownership": [...],
  "searchTreeStats": {
    "treeDepth": 15,
    "branchingFactor": 3.2,
    "nodesExpanded": 4500
  },
  "confidenceMetrics": {
    "policyEntropy": 2.1,
    "valueConfidence": 0.85,
    "scoreVariance": 2.3
  }
}
```

### New Tool: getDetailedMetrics

```typescript
function getDetailedMetrics(params: {
  sgf: string;
  moveNumber?: number;
  metricTypes?: string[]; // Filter which metrics to return
}): DetailedMetrics
```

## Benefits

1. **Deeper Insights**: Understand how KataGo evaluates positions
2. **Performance Monitoring**: Track and optimize engine performance
3. **Educational Value**: Learn from the AI's decision-making process
4. **Debugging**: Identify issues in analysis or unusual positions
5. **Research**: Enable new analysis and visualization possibilities

## Backward Compatibility

All new fields will be optional and omitted by default to maintain compatibility with existing clients. A new `includeMetrics` parameter will control which additional metrics to return.

## Testing Strategy

1. Unit tests for each new metric calculation
2. Integration tests with real KataGo engine
3. Performance benchmarks to ensure minimal overhead
4. Validation against known positions
5. Edge case testing (early game, endgame, unusual positions)

## Future Extensions

- Historical metric tracking across analysis sessions
- Metric comparison between different KataGo models
- Real-time metric streaming during analysis
- Machine learning on collected metrics for insights