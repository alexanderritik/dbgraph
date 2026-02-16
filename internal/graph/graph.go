package graph

import (
	"fmt"
	"strings"
)

// NodeType represents the type of a database object
type NodeType string

const (
	Table   NodeType = "TABLE"
	View    NodeType = "VIEW"
	Trigger NodeType = "TRIGGER"
)

// DependencyType represents the type of relationship between nodes
type DependencyType string

const (
	ForeignKey    DependencyType = "FOREIGN_KEY"
	ViewDepends   DependencyType = "VIEW_DEPENDS"
	TriggerAction DependencyType = "TRIGGER_ACTION"
	Inheritance   DependencyType = "INHERITANCE" // For partitions
)

// ColumnDependency represents a database object that depends on a specific column
type ColumnDependency struct {
	Schema string
	Name   string
	Type   string // "VIEW", "TRIGGER", "INDEX", "FUNCTION", "TABLE"
	Detail string // e.g., "src code match", "param ref"
}

// Node represents a database object (Table or View)
type Node struct {
	ID       string // Schema.Name
	Schema   string
	Name     string
	Type     NodeType
	Size     string     // e.g., "12MB", "400kB"
	RowCount int64      // Estimated row count
	Indexes  [][]string // List of indexed column sets
}

// DBMetrics holds real-time database statistics
type DBMetrics struct {
	ActiveLocks    int
	MaxConns       int
	UsedConns      int
	LongestQuery   string // e.g. "4.2s (PID 1294)"
	ConnSaturation string // e.g. "82%"
}

// QueryStats represents performance statistics for a single query
type QueryStats struct {
	QueryID     string
	Query       string
	Calls       int64
	TotalTime   float64 // milliseconds
	AvgTime     float64 // milliseconds
	LoadPercent float64
}

// Edge represents a dependency: Source -> Target
// If A has FK to B, A depends on B. So Edge is A -> B.
type Edge struct {
	SourceID string
	TargetID string
	Type     DependencyType

	// Metadata for High-Fidelity Analysis
	ConstraintName string
	DeleteRule     string // "CASCADE", "RESTRICT", "SET NULL", "NO ACTION"
	MetaData       map[string]string
}

// Graph holds the adjacency list of the database schema
type Graph struct {
	Nodes map[string]*Node
	Edges map[string][]*Edge // Adjacency list: SourceID -> List of Edges
}

// NewGraph creates a new empty graph
func NewGraph() *Graph {
	return &Graph{
		Nodes: make(map[string]*Node),
		Edges: make(map[string][]*Edge),
	}
}

// AddNode adds a node to the graph
func (g *Graph) AddNode(schema, name string, nodeType NodeType, size string, rowCount int64) {
	id := fmt.Sprintf("%s.%s", schema, name)
	if _, exists := g.Nodes[id]; !exists {
		g.Nodes[id] = &Node{
			ID:       id,
			Schema:   schema,
			Name:     name,
			Type:     nodeType,
			Size:     size,
			RowCount: rowCount,
		}
	} else {
		// Update fields if they were missing (e.g. implicitly added)
		if g.Nodes[id].Size == "" && size != "" {
			g.Nodes[id].Size = size
		}
		if g.Nodes[id].RowCount == 0 && rowCount != 0 {
			g.Nodes[id].RowCount = rowCount
		}
	}
}

// AddIndex adds an index definition to a node
func (g *Graph) AddIndex(schema, name string, columns []string) {
	id := fmt.Sprintf("%s.%s", schema, name)
	if node, exists := g.Nodes[id]; exists {
		node.Indexes = append(node.Indexes, columns)
	}
}

// AddEdge adds a directed edge from source to target
func (g *Graph) AddEdge(sourceSchema, sourceName, targetSchema, targetName string, depType DependencyType, constraintName, deleteRule string) {
	sourceID := fmt.Sprintf("%s.%s", sourceSchema, sourceName)
	targetID := fmt.Sprintf("%s.%s", targetSchema, targetName)

	// Ensure nodes exist (implicitly tables if not specified? better to be explicit)
	// For now, we assume nodes are added before edges or we auto-add them.
	// Let's auto-add as Tables if missing, though ideally we should know types.
	if _, ok := g.Nodes[sourceID]; !ok {
		g.AddNode(sourceSchema, sourceName, Table, "", 0)
	}
	if _, ok := g.Nodes[targetID]; !ok {
		g.AddNode(targetSchema, targetName, Table, "", 0)
	}

	edge := &Edge{
		SourceID:       sourceID,
		TargetID:       targetID,
		Type:           depType,
		ConstraintName: constraintName,
		DeleteRule:     deleteRule,
	}

	g.Edges[sourceID] = append(g.Edges[sourceID], edge)
}

// GetDownstream returns all nodes that depend on the given node (Reverse dependency)
// Real impact analysis: If A depends on B (A -> B), and we change B, A is impacted.
// So we need to look for edges where Target == NodeID.
func (g *Graph) GetDownstream(nodeID string) []string {
	impactedMap := make(map[string]bool)
	queue := []string{nodeID}
	visited := make(map[string]bool)
	visited[nodeID] = true

	// Pre-compute reverse edges for traversal
	reverseEdges := make(map[string][]string)
	for src, edges := range g.Edges {
		for _, edge := range edges {
			reverseEdges[edge.TargetID] = append(reverseEdges[edge.TargetID], src)
		}
	}

	idx := 0
	for idx < len(queue) {
		current := queue[idx]
		idx++

		dependents := reverseEdges[current]
		for _, dep := range dependents {
			if !visited[dep] {
				visited[dep] = true
				impactedMap[dep] = true // It is impacted
				queue = append(queue, dep)
			}
		}
	}

	impacted := make([]string, 0, len(impactedMap))
	for id := range impactedMap {
		impacted = append(impacted, id)
	}
	return impacted
}

// NodeRank represents a node's topological importance
type NodeRank struct {
	ID         string
	Type       NodeType
	InDegree   int
	OutDegree  int
	Rows       int64
	Centrality float64
}

// Stats returns topological metrics of the graph
type GraphStats struct {
	Nodes          int
	Edges          int
	Density        float64
	Components     int
	MaxCentrality  float64
	CentralNode    string
	LongestPath    int
	DeepestChain   []string
	IsolatedGroups []string
	TopNodes       []NodeRank // Top nodes by centrality/impact
}

// AnalyzeTopology computes comprehensive graph metrics
func (g *Graph) AnalyzeTopology() *GraphStats {
	stats := &GraphStats{
		Nodes: len(g.Nodes),
	}

	edgeCount := 0
	inDegree := make(map[string]int)
	outDegree := make(map[string]int)

	for src, edges := range g.Edges {
		edgeCount += len(edges)
		outDegree[src] += len(edges)
		for _, edge := range edges {
			inDegree[edge.TargetID]++
		}
	}
	stats.Edges = edgeCount

	// Density: E / (V * (V-1))
	if stats.Nodes > 1 {
		stats.Density = float64(edgeCount) / float64(stats.Nodes*(stats.Nodes-1))
	}

	// Centrality (Degree Centrality: in + out)
	maxDegree := 0
	var centralNode string
	var ranks []NodeRank

	for id, node := range g.Nodes {
		dIn := inDegree[id]
		dOut := outDegree[id]
		dTotal := dIn + dOut

		if dTotal > maxDegree {
			maxDegree = dTotal
			centralNode = id
		}

		ranks = append(ranks, NodeRank{
			ID:         id,
			Type:       node.Type,
			InDegree:   dIn,
			OutDegree:  dOut,
			Rows:       node.RowCount,
			Centrality: float64(dTotal), // Simplified for now
		})
	}
	stats.MaxCentrality = float64(maxDegree)
	stats.CentralNode = centralNode

	// Sort ranks by Centrality (Impact) descending
	// Bubble sort for simplicity (N is small < 1000 usually)
	for i := 0; i < len(ranks)-1; i++ {
		for j := 0; j < len(ranks)-i-1; j++ {
			if ranks[j].Centrality < ranks[j+1].Centrality {
				ranks[j], ranks[j+1] = ranks[j+1], ranks[j]
			}
		}
	}
	// Keep top 20 -> Removed limit to let CLI handle it
	stats.TopNodes = ranks

	// Connected Components (Weakly Connected for Islands)
	visited := make(map[string]bool)
	components := 0
	var isolated []string

	// Build undirected adjacency for component search
	undirected := make(map[string][]string)
	for src, edges := range g.Edges {
		for _, edge := range edges {
			undirected[src] = append(undirected[src], edge.TargetID)
			undirected[edge.TargetID] = append(undirected[edge.TargetID], src)
		}
	}

	for id := range g.Nodes {
		if !visited[id] {
			components++
			// BFS to find all nodes in this component
			componentNodes := []string{id}
			queue := []string{id}
			visited[id] = true

			// If node has no edges at all (in or out), it's highly isolated
			if len(undirected[id]) == 0 {
				isolated = append(isolated, id)
				continue
			}

			// BFS
			cIdx := 0
			for cIdx < len(queue) {
				curr := queue[cIdx]
				cIdx++
				for _, neighbor := range undirected[curr] {
					if !visited[neighbor] {
						visited[neighbor] = true
						queue = append(queue, neighbor)
						componentNodes = append(componentNodes, neighbor)
					}
				}
			}

			// Store isolated small clusters (arbitrary size < 3)
			if len(componentNodes) < 3 && len(componentNodes) > 0 {
				isolated = append(isolated, fmt.Sprintf("%v", componentNodes))
			}
		}
	}
	stats.Components = components
	stats.IsolatedGroups = isolated

	// Longest Path (DAG assumption or limited depth for cycles)
	// Simple DFS with memoization
	memo := make(map[string]int)
	var getDepth func(id string, pathStack map[string]bool) int
	getDepth = func(id string, pathStack map[string]bool) int {
		if d, ok := memo[id]; ok {
			return d
		}
		if pathStack[id] {
			return 0 // Cycle detected, break infinite loop
		}
		pathStack[id] = true

		maxD := 0
		for _, edge := range g.Edges[id] {
			d := getDepth(edge.TargetID, pathStack)
			if d > maxD {
				maxD = d
			}
		}
		pathStack[id] = false
		memo[id] = 1 + maxD
		return 1 + maxD
	}

	maxPath := 0
	for id := range g.Nodes {
		d := getDepth(id, make(map[string]bool))
		if d > maxPath {
			maxPath = d
		}
	}
	stats.LongestPath = maxPath
	return stats
}

// CheckCycles implements Tarjan's Algorithm to find Strongly Connected Components (SCCs)
// Any SCC with more than 1 node, or a node with a self-loop, represents a cycle.
func (g *Graph) CheckCycles() [][]string {
	var index int
	var stack []string

	indices := make(map[string]int)
	lowLink := make(map[string]int)
	onStack := make(map[string]bool)
	var sccs [][]string

	var strongconnect func(string)

	strongconnect = func(v string) {
		indices[v] = index
		lowLink[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true

		// Consider neighbours (Dependencies)
		// Edge Source -> Target means Source depends on Target
		if edges, ok := g.Edges[v]; ok {
			for _, wEdge := range edges {
				w := wEdge.TargetID
				if _, ok := indices[w]; !ok {
					strongconnect(w)
					if lowLink[w] < lowLink[v] {
						lowLink[v] = lowLink[w]
					}
				} else if onStack[w] {
					if indices[w] < lowLink[v] {
						lowLink[v] = indices[w]
					}
				}
			}
		}

		// If v is a root node, pop the stack and generate an SCC
		if lowLink[v] == indices[v] {
			var scc []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}

			// Cycle condition: SCC size > 1 OR (size == 1 AND self-loop)
			isCycle := false
			if len(scc) > 1 {
				isCycle = true
			} else if len(scc) == 1 {
				// Check for self-loop
				if edges, ok := g.Edges[v]; ok {
					for _, e := range edges {
						if e.TargetID == v {
							isCycle = true
							break
						}
					}
				}
			}

			if isCycle {
				sccs = append(sccs, scc)
			}
		}
	}

	for nodeID := range g.Nodes {
		if _, ok := indices[nodeID]; !ok {
			strongconnect(nodeID)
		}
	}

	return sccs
}

// IndexIssues represents the result of an index hygiene check
type IndexIssues struct {
	MissingFKIndexes []string // List of FK constraints without a supporting index
	TotalFKs         int
	IndexedFKs       int
}

// CheckIndexCoverage identifies foreign keys that lack a supporting index
// Ideally, every FK source column(s) should be indexed to avoid table scans on delete/update.
// This is a naive check: we look for an index whose *prefix* matches the FK columns.
// Assuming simple single-column FKs for now mostly, but robust enough for multi-column.
func (g *Graph) CheckIndexCoverage() *IndexIssues {
	issues := &IndexIssues{}

	for _, edges := range g.Edges {
		for _, edge := range edges {
			if edge.Type == ForeignKey {
				issues.TotalFKs++

				// Find source table
				srcNode, ok := g.Nodes[edge.SourceID]
				if !ok {
					continue
				}

				// Get FK columns from metadata (we stored it in PostgresAdapter!)
				fkColsStr, ok := edge.MetaData["fk_columns"]
				if !ok || fkColsStr == "" {
					continue // Can't check if we don't know columns
				}
				fkCols := strings.Split(fkColsStr, ",")

				isIndexed := false
				for _, idx := range srcNode.Indexes {
					// Check if index covers FK columns as a prefix
					if len(idx) < len(fkCols) {
						continue
					}

					match := true
					for i, col := range fkCols {
						if idx[i] != col {
							match = false
							break
						}
					}
					if match {
						isIndexed = true
						break
					}
				}

				if isIndexed {
					issues.IndexedFKs++
				} else {
					// Format: Table (fk_col1, fk_col2) -> Target
					issues.MissingFKIndexes = append(issues.MissingFKIndexes,
						fmt.Sprintf("%s (%s) -> %s", edge.SourceID, fkColsStr, edge.TargetID))
				}
			}
		}
	}
	return issues
}

// GodMod represents a node identified as a High Coupling Risk / God Object
type GodMod struct {
	ID           string
	Degree       int
	Dependents   int // Fan-in
	Dependencies int // Fan-out
}

// DetectGodObjects identifies nodes with excessive connectivity
// Threshold: If Degree (In+Out) > 20 (heuristic), it is potential technical debt.
func (g *Graph) DetectGodObjects() []GodMod {
	var gods []GodMod
	threshold := 15 // Lowered slightly for the test DB context, usually 20-30

	inDegree := make(map[string]int)
	outDegree := make(map[string]int)

	for src, edges := range g.Edges {
		outDegree[src] += len(edges)
		for _, edge := range edges {
			inDegree[edge.TargetID]++
		}
	}

	for id := range g.Nodes {
		in := inDegree[id]
		out := outDegree[id]
		total := in + out

		if total >= threshold {
			gods = append(gods, GodMod{
				ID:           id,
				Degree:       total,
				Dependents:   in,
				Dependencies: out,
			})
		}
	}
	return gods
}
