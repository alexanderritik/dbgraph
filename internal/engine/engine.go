package engine

import (
	"fmt"

	"github.com/alexanderritik/dbgraph/internal/adapters"
	"github.com/alexanderritik/dbgraph/internal/graph"
)

// Engine orchestrates the application logic
type Engine struct {
	Graph   *graph.Graph
	Adapter adapters.Adapter
}

// NewEngine creates a new engine instance
func NewEngine(g *graph.Graph, a adapters.Adapter) *Engine {
	return &Engine{
		Graph:   g,
		Adapter: a,
	}
}

// Connect connects to the database
func (e *Engine) Connect(connString string) error {
	return e.Adapter.Connect(connString)
}

// BuildGraph fetches the schema and builds the graph
func (e *Engine) BuildGraph() error {
	return e.Adapter.FetchSchema(e.Graph)
}

// GetGraphStats returns simple stats about the graph
func (e *Engine) GetGraphStats() string {
	nodeCount := len(e.Graph.Nodes)
	edgeCount := 0
	for _, edges := range e.Graph.Edges {
		edgeCount += len(edges)
	}
	return fmt.Sprintf("Graph built successfully.\nNodes: %d\nEdges: %d", nodeCount, edgeCount)
}

// Run (Legacy/Serve)
func (e *Engine) Run() {
	fmt.Println("Engine is running... (Use 'analyze' or 'impact' commands)")
}
