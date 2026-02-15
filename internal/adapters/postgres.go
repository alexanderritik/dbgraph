package adapters

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alexanderritik/dbgraph/internal/graph"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresAdapter handles PostgreSQL interactions
type PostgresAdapter struct {
	Pool *pgxpool.Pool
}

// NewPostgresAdapter creates a new postgres adapter
func NewPostgresAdapter() *PostgresAdapter {
	return &PostgresAdapter{}
}

// Connect establishes a connection to the database
func (p *PostgresAdapter) Connect(connString string) error {
	var err error
	p.Pool, err = pgxpool.New(context.Background(), connString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		return err
	}
	return nil
}

// Close closes the connection pool
func (p *PostgresAdapter) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}

// FetchSchema queries the database schema and populates the graph
func (p *PostgresAdapter) FetchSchema(g *graph.Graph) error {
	if p.Pool == nil {
		return fmt.Errorf("database connection not established")
	}

	ctx := context.Background()

	// 1. Fetch Tables, Sizes, and Row Counts (Nodes)
	// pg_class join pg_namespace
	rows, err := p.Pool.Query(ctx, queryFetchNodes)
	if err != nil {
		return fmt.Errorf("failed to fetch nodes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schema, name, kind, size string
		var rowCount float64 // reltuples is float4
		if err := rows.Scan(&schema, &name, &kind, &size, &rowCount); err != nil {
			return err
		}
		var nodeType graph.NodeType
		switch kind {
		case "VIEW", "MATERIALIZED VIEW":
			nodeType = graph.View
		default:
			nodeType = graph.Table
		}

		// Handle -1 for un-analyzed tables
		rc := int64(rowCount)
		if rc < 0 {
			rc = 0
		}
		g.AddNode(schema, name, nodeType, size, rc)
	}

	// 1.5 Fetch Indexes (for Structural Warnings)
	// Simplified robust query to get index columns per table
	ixRows, err := p.Pool.Query(ctx, queryFetchIndexes)
	if err != nil {
		fmt.Printf("Warning: failed to fetch indexes: %v\n", err)
	} else {
		defer ixRows.Close()
		for ixRows.Next() {
			var schema, table string
			var cols []string
			if err := ixRows.Scan(&schema, &table, &cols); err != nil {
				continue
			}
			g.AddIndex(schema, table, cols)
		}
	}

	// 2. Fetch Foreign Keys (Table Dependencies)
	// source_table -> target_table
	fkRows, err := p.Pool.Query(ctx, queryFetchForeignKeys)
	if err != nil {
		return fmt.Errorf("failed to fetch foreign keys: %w", err)
	}
	defer fkRows.Close()

	for fkRows.Next() {
		var schema, table, fSchema, fTable, constraintName, deleteRule string
		var fkCols []string
		if err := fkRows.Scan(&schema, &table, &fSchema, &fTable, &constraintName, &deleteRule, &fkCols); err != nil {
			return err
		}

		g.AddNode(schema, table, graph.Table, "", 0)
		g.AddNode(fSchema, fTable, graph.Table, "", 0)

		g.AddEdge(schema, table, fSchema, fTable, graph.ForeignKey, constraintName, deleteRule)

		// Hack to add metadata to the just-added edge
		srcID := fmt.Sprintf("%s.%s", schema, table)
		edges := g.Edges[srcID]
		if len(edges) > 0 {
			lastEdge := edges[len(edges)-1]
			if lastEdge.MetaData == nil {
				lastEdge.MetaData = make(map[string]string)
			}
			lastEdge.MetaData["fk_columns"] = strings.Join(fkCols, ",")
		}
	}

	// 3. Fetch View Dependencies
	vRows, err := p.Pool.Query(ctx, queryFetchViews)
	if err != nil {
		return fmt.Errorf("failed to fetch view dependencies: %w", err)
	}
	defer vRows.Close()

	for vRows.Next() {
		var vSchema, vName, tSchema, tName string
		if err := vRows.Scan(&vSchema, &vName, &tSchema, &tName); err != nil {
			return err
		}

		// Add View Node
		g.AddNode(vSchema, vName, graph.View, "", 0)
		// Ensure Target Node exists
		g.AddNode(tSchema, tName, graph.Table, "", 0)

		// Add dependency: View -> Target
		g.AddEdge(vSchema, vName, tSchema, tName, graph.ViewDepends, "", "")
	}

	return nil
}

// GetMetrics returns real-time database statistics
func (p *PostgresAdapter) GetMetrics() (*graph.DBMetrics, error) {
	if p.Pool == nil {
		return nil, fmt.Errorf("database connection not established")
	}
	ctx := context.Background()
	m := &graph.DBMetrics{}

	// Active Locks
	// Count of locks not granted? No, locks held or waiting.
	// Simple proxy for "activity": count of locks in pg_locks
	err := p.Pool.QueryRow(ctx, queryActiveLocks).Scan(&m.ActiveLocks)
	if err != nil {
		return nil, err
	}

	// Connections
	// max_connections vs count(*) from pg_stat_activity

	// Robust way: select setting from pg_settings
	err = p.Pool.QueryRow(ctx, queryMaxConns).Scan(&m.MaxConns)
	if err != nil {
		m.MaxConns = 100 // fallback
	}

	err = p.Pool.QueryRow(ctx, queryUsedConns).Scan(&m.UsedConns)
	if err != nil {
		return nil, err
	}

	if m.MaxConns > 0 {
		m.ConnSaturation = fmt.Sprintf("%d%%", int(float64(m.UsedConns)/float64(m.MaxConns)*100))
	}

	// Longest Running Query
	// duration, pid
	var duration float64
	var pid int
	err = p.Pool.QueryRow(ctx, queryLongestRunning).Scan(&duration, &pid)
	if err != nil {
		// No active query logic
		m.LongestQuery = "None"
	} else {
		m.LongestQuery = fmt.Sprintf("%.1fs (PID %d)", duration, pid)
	}

	return m, nil
}
