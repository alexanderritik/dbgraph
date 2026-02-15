package graph

import (
	"fmt"
)

// NodeType represents the type of a database object
type NodeType string

const (
	Table NodeType = "TABLE"
	View  NodeType = "VIEW"
)

// DependencyType represents the type of relationship between nodes
type DependencyType string

const (
	ForeignKey  DependencyType = "FOREIGN_KEY"
	ViewDepends DependencyType = "VIEW_DEPENDS"
)

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
	// Build reverse adjacency list on the fly or iterate (inefficient for huge graphs but fine for DB schemas)
	// impacted := []string{}
	// For O(1) lookups, we might want a ReverseEdges map, but iteration is okay for now.

	impactedMap := make(map[string]bool)
	queue := []string{nodeID}
	visited := make(map[string]bool)
	visited[nodeID] = true

	// Standard BFS on the *reverse* graph
	// If Edge is Source -> Target, it means Source depends on Target.
	// If Target changes, Source is impacted.
	// So we want to traverse from Target back to Source.

	// Pre-compute reverse edges for traversal
	reverseEdges := make(map[string][]string)
	for src, edges := range g.Edges {
		for _, edge := range edges {
			reverseEdges[edge.TargetID] = append(reverseEdges[edge.TargetID], src)
		}
	}

	// BFS
	idx := 0
	for idx < len(queue) {
		current := queue[idx]
		idx++

		// Find nodes that depend on 'current'
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
