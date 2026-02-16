package graph

// TraceResult holds the parsed performance data from an EXPLAIN ANALYZE
type TraceResult struct {
	PlanningTime  float64
	ExecutionTime float64
	TotalTime     float64
	CacheHits     int64
	DiskReads     int64
	MemoryUsage   int64 // in bytes (approximated from shared/temp buffers)
	Root          *ExplainNode
}

// ExplainNode represents a node in the Postgres execution plan tree
type ExplainNode struct {
	// Identity
	Type     string `json:"Node Type"`
	Strategy string `json:"Strategy,omitempty"` // e.g. "Plain", "Sorted" for Aggregate, or Join Type?
	// Join Type sometimes is separate field "Join Type"

	// Costs & Rows
	StartupCost float64 `json:"Startup Cost"`
	TotalCost   float64 `json:"Total Cost"`
	PlanRows    float64 `json:"Plan Rows"`
	ActualRows  float64 `json:"Actual Rows"`
	ActualLoops float64 `json:"Actual Loops"`

	// Context (Tables, Conditions)
	RelationName string `json:"Relation Name,omitempty"`
	Schema       string `json:"Schema,omitempty"`
	Alias        string `json:"Alias,omitempty"`
	IndexName    string `json:"Index Name,omitempty"`
	IndexCond    string `json:"Index Cond,omitempty"`
	Filter       string `json:"Filter,omitempty"`

	// Buffers (I/O)
	SharedHitBlocks   int64 `json:"Shared Hit Blocks"`
	SharedReadBlocks  int64 `json:"Shared Read Blocks"`
	LocalHitBlocks    int64 `json:"Local Hit Blocks"`
	LocalReadBlocks   int64 `json:"Local Read Blocks"`
	TempReadBlocks    int64 `json:"Temp Read Blocks"`
	TempWrittenBlocks int64 `json:"Temp Written Blocks"`

	// Children
	Plans []*ExplainNode `json:"Plans,omitempty"`
}

// ExplainOutput represents the top-level array returned by EXPLAIN JSON
// Postgres returns [ { "Plan": ..., "Planning Time": ..., "Execution Time": ... } ]
type ExplainOutput struct {
	Plan          *ExplainNode  `json:"Plan"`
	PlanningTime  float64       `json:"Planning Time"`
	ExecutionTime float64       `json:"Execution Time"`
	Triggers      []interface{} `json:"Triggers,omitempty"`
}
