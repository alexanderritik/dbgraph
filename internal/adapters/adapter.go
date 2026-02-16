package adapters

import (
	"fmt"
	"strings"

	"github.com/alexanderritik/dbgraph/internal/graph"
)

// Adapter is the interface that all database adapters must implement
type Adapter interface {
	Connect(connString string) error
	Close()
	FetchSchema(g *graph.Graph) error
	GetMetrics() (*graph.DBMetrics, error)
	GetColumnDependencies(schema, table, column string) ([]graph.ColumnDependency, error)
	GetTableDependencies(schema, table string) ([]graph.ColumnDependency, error)
	GetTopQueries(limit int, sortBy string) ([]graph.QueryStats, error)
}

// NewAdapter creates a new adapter based on the connection string scheme
func NewAdapter(connString string) (Adapter, error) {
	if strings.HasPrefix(connString, "postgres://") || strings.HasPrefix(connString, "postgresql://") {
		return NewPostgresAdapter(), nil
	}
	// Future: Add MySQL support here
	// if strings.HasPrefix(connString, "mysql://") { ... }

	return nil, fmt.Errorf("unsupported database scheme in connection string: %s", connString)
}
