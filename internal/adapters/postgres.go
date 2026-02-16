package adapters

import (
	"context"
	"encoding/json"
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

	viewEdges := make(map[string]bool)
	for vRows.Next() {
		var vSchema, vName, tSchema, tName string
		if err := vRows.Scan(&vSchema, &vName, &tSchema, &tName); err != nil {
			return err
		}
		if err := vRows.Scan(&vSchema, &vName, &tSchema, &tName); err != nil {
			return err
		}

		// Add View Node
		g.AddNode(vSchema, vName, graph.View, "", 0)
		// Ensure Target Node exists
		g.AddNode(tSchema, tName, graph.Table, "", 0)

		// Deduplicate: Check if edge already processed
		edgeKey := fmt.Sprintf("%s.%s->%s.%s", vSchema, vName, tSchema, tName)
		if viewEdges[edgeKey] {
			continue
		}
		viewEdges[edgeKey] = true

		// Add dependency: View -> Target
		g.AddEdge(vSchema, vName, tSchema, tName, graph.ViewDepends, "", "")
	}

	// 4. Fetch Triggers & Analyze Function Bodies
	tRows, err := p.Pool.Query(ctx, queryFetchTriggers)
	if err == nil {
		defer tRows.Close()
		for tRows.Next() {
			var schema, table, trigger, funcName, level string
			if err := tRows.Scan(&schema, &table, &trigger, &funcName, &level); err == nil {
				// Add Trigger Node
				g.AddNode(schema, trigger, graph.Trigger, "", 0)
				// Link: Table -> Trigger (Trigger actions ON table)
				// Note: Originally we did Trigger->Table. But topologically, the Trigger is downstream of Table instructions.
				// However, if we view it as "Trigger Acts On Table", it might be Trigger -> Table.
				// Let's keep existing direction: Trigger -> Table (Dependency: Trigger depends on Table existence)
				g.AddEdge(schema, trigger, schema, table, graph.TriggerAction, "", "")

				// NEW: Fetch Function Body to find downstream dependencies (e.g., Audit Logs)
				var body string
				err := p.Pool.QueryRow(ctx, queryFetchFunctionBody, funcName, schema).Scan(&body)
				if err == nil {
					// Simple Heuristic: Look for "INSERT INTO <table>", "UPDATE <table>"
					// We can iterate over all known nodes to see if they are mentioned?
					// Or just basic regex for "INSERT INTO table"
					// Let's check against all existing nodes to be safe and accurate.
					upperBody := strings.ToUpper(body)
					for id, node := range g.Nodes {
						// generic check: "INSERT INTO <name>" or "UPDATE <name>"
						// schema.name or just name
						// crude check: matches name and is not the source table
						if id == fmt.Sprintf("%s.%s", schema, table) {
							continue
						}

						// Check for "Schema.Name" or "Name" if schema matches
						// This is expensive O(Triggers * Nodes), but N is small.
						targetName := node.Name
						if strings.Contains(upperBody, strings.ToUpper(targetName)) {
							// Check if it looks like a SQL command
							// "INSERT INTO target", "UPDATE target"
							// We'll simplisticly assume if the table name is present, it's a dependency.
							// Add Edge: Trigger -> TargetTable
							g.AddEdge(schema, trigger, node.Schema, node.Name, graph.TriggerAction, "Function Call", "")
						}
					}
				}
			}
		}
	}

	// 5. Fetch Table Inheritance (Partitions)
	iRows, err := p.Pool.Query(ctx, queryFetchInheritance)
	if err == nil {
		defer iRows.Close()
		for iRows.Next() {
			var pSchema, pName, cSchema, cName string
			if err := iRows.Scan(&pSchema, &pName, &cSchema, &cName); err == nil {
				// Add dependency: Child -> Parent (Inheritance)
				g.AddEdge(cSchema, cName, pSchema, pName, graph.Inheritance, "", "")
			}
		}
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

// GetColumnDependencies identifies all objects that depend on a specific column
func (p *PostgresAdapter) GetColumnDependencies(schema, table, column string) ([]graph.ColumnDependency, error) {
	if p.Pool == nil {
		return nil, fmt.Errorf("database connection not established")
	}
	ctx := context.Background()
	var deps []graph.ColumnDependency

	// 0. Get Attribute Number for robust querying
	var attNum int
	err := p.Pool.QueryRow(ctx, queryGetColumnAttNum, schema, table, column).Scan(&attNum)
	if err != nil {
		// If column doesn't exist, return error
		return nil, fmt.Errorf("column '%s.%s' not found: %w", table, column, err)
	}

	// 1. Foreign Keys referencing this column
	fkRows, err := p.Pool.Query(ctx, queryFKRefsByColumn, schema, table, attNum)
	if err == nil {
		defer fkRows.Close()
		for fkRows.Next() {
			var conName, srcTable string
			if err := fkRows.Scan(&conName, &srcTable); err == nil {
				deps = append(deps, graph.ColumnDependency{
					Schema: schema, // Technically referencing table might be elsewhere, but keeping simpler for now
					Name:   srcTable,
					Type:   "FOREIGN_KEY",
					Detail: fmt.Sprintf("Constraint: %s", conName),
				})
			}
		}
	}

	// 2. Indexes using this column
	ixRows, err := p.Pool.Query(ctx, queryIndexesByColumn, schema, table, attNum)
	if err == nil {
		defer ixRows.Close()
		for ixRows.Next() {
			var indexName string
			if err := ixRows.Scan(&indexName); err == nil {
				deps = append(deps, graph.ColumnDependency{
					Schema: schema,
					Name:   indexName,
					Type:   "INDEX",
					Detail: "Index covers this column",
				})
			}
		}
	}

	// 3. Direct Dependencies via pg_depend (Views, Triggers, etc.)
	rows, err := p.Pool.Query(ctx, queryColumnDependencies, table, schema, column)
	if err != nil {
		fmt.Printf("Warning: failed to fetch pg_depend: %v\n", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var depType, depName, depCode, depSchema string
			if err := rows.Scan(&depType, &depName, &depCode, &depSchema); err != nil {
				continue
			}

			// Refine Type
			readableType := "UNKNOWN"
			if strings.Contains(depType, "rewrite") {
				readableType = "VIEW"
			} else if strings.Contains(depType, "trigger") {
				readableType = "TRIGGER"
			} else if strings.Contains(depType, "class") {
				readableType = "RELATION" // Could be View, Table, Index
				if depName == table || strings.Contains(depName, table) {
					// Self references (like PK index) are interesting but maybe filtered if needed
					// But "id" dropping breaks the table itself effectively?
					// No, it breaks the PK.
				}
			}

			// Deduplicate: If we already found this via FK/Index query, skip?
			// Simple check:
			seen := false
			for _, d := range deps {
				if d.Name == depName {
					seen = true
					break
				}
			}
			if !seen {
				deps = append(deps, graph.ColumnDependency{
					Schema: depSchema,
					Name:   depName,
					Type:   readableType,
					Detail: fmt.Sprintf("Deep Dependency (%s)", depCode),
				})
			}
		}
		if err := rows.Err(); err != nil {
			fmt.Printf("Warning: error iterating pg_depend: %v\n", err)
		}
	}

	// 4. Scan Function Bodies (Soft Dependencies)
	fRows, err := p.Pool.Query(ctx, queryScanFunctionsForColumn, column)
	if err == nil {
		defer fRows.Close()
		for fRows.Next() {
			var fSchema, fName, fSrc string
			if err := fRows.Scan(&fSchema, &fName, &fSrc); err != nil {
				continue
			}
			if strings.Contains(strings.ToUpper(fSrc), strings.ToUpper(table)) {
				deps = append(deps, graph.ColumnDependency{
					Schema: fSchema,
					Name:   fName,
					Type:   "FUNCTION",
					Detail: "Code Reference (Regex Scan)",
				})
			}
		}
	}

	return deps, nil
}

// GetTableDependencies identifies all objects that depend on a specific table
func (p *PostgresAdapter) GetTableDependencies(schema, table string) ([]graph.ColumnDependency, error) {
	if p.Pool == nil {
		return nil, fmt.Errorf("database connection not established")
	}
	ctx := context.Background()
	var deps []graph.ColumnDependency

	// 1. Foreign Keys pointing TO this table
	fkRows, err := p.Pool.Query(ctx, queryFKRefsByTable, schema, table)
	if err == nil {
		defer fkRows.Close()
		for fkRows.Next() {
			var conName, srcTable string
			if err := fkRows.Scan(&conName, &srcTable); err == nil {
				deps = append(deps, graph.ColumnDependency{
					Schema: schema,
					Name:   srcTable,
					Type:   "FOREIGN_KEY",
					Detail: fmt.Sprintf("Constraint: %s", conName),
				})
			}
		}
	}

	// 2. Direct Dependencies via pg_depend (Views, Triggers, Mat Views)
	rows, err := p.Pool.Query(ctx, queryTableDependencies, table, schema)
	if err != nil {
		fmt.Printf("Warning: failed to fetch table dependencies: %v\n", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var depType, depName, depCode, depSchema string
			if err := rows.Scan(&depType, &depName, &depCode, &depSchema); err != nil {
				continue
			}

			// Filter: Only show 'n' (Normal) dependencies.
			// 'a' (Automatic) and 'i' (Internal) are deleted with the table, so they are not "blockers" or "breakages" in the same sense.
			// Exception: We might want to warn about Cascades? But standard 'n' means restricted.
			if depCode != "n" {
				continue
			}

			readableType := "UNKNOWN"
			if strings.Contains(depType, "rewrite") {
				readableType = "VIEW"
			} else if strings.Contains(depType, "trigger") {
				readableType = "TRIGGER"
			} else if strings.Contains(depType, "constraint") {
				readableType = "CONSTRAINT"
			} else if strings.Contains(depType, "class") {
				readableType = "RELATION"
			}

			// Deduplicate
			seen := false
			for _, d := range deps {
				if d.Name == depName {
					seen = true
					break
				}
				// Check if this dependency (e.g. Constraint Name) is already mentioned in Detail of another dep
				if strings.Contains(d.Detail, depName) {
					seen = true
					break
				}
			}

			if !seen {
				deps = append(deps, graph.ColumnDependency{
					Schema: depSchema,
					Name:   depName,
					Type:   readableType,
					Detail: fmt.Sprintf("Hard Dependency (%s)", depCode),
				})
			}
		}
		if err := rows.Err(); err != nil {
			fmt.Printf("Warning: error iterating pg_depend table: %v\n", err)
		}
	}

	// 3. Scan Function Bodies (Soft Dependencies)
	fRows, err := p.Pool.Query(ctx, queryScanFunctionsForColumn, table)
	if err == nil {
		defer fRows.Close()
		for fRows.Next() {
			var fSchema, fName, fSrc string
			if err := fRows.Scan(&fSchema, &fName, &fSrc); err != nil {
				continue
			}
			// Strict check: contains "schem.table" or "table"
			// Case insensitive
			upperSrc := strings.ToUpper(fSrc)
			upperTable := strings.ToUpper(table)
			if strings.Contains(upperSrc, upperTable) {
				deps = append(deps, graph.ColumnDependency{
					Schema: fSchema,
					Name:   fName,
					Type:   "FUNCTION",
					Detail: "Code Reference (Regex Scan)",
				})
			}
		}
	}

	return deps, nil
}

// GetTopQueries fetches the top costly queries from pg_stat_statements
func (p *PostgresAdapter) GetTopQueries(limit int, sortBy string) ([]graph.QueryStats, error) {
	if p.Pool == nil {
		return nil, fmt.Errorf("database connection not established")
	}

	// 1. Determine ORDER BY clause safely
	var orderBy string
	switch sortBy {
	case "calls":
		orderBy = "ORDER BY calls DESC"
	case "avg_time":
		orderBy = "ORDER BY avg_time DESC"
	case "total", "total_time":
		orderBy = "ORDER BY total_time DESC"
	default:
		orderBy = "ORDER BY total_time DESC"
	}

	// 2. Construct final query
	finalQuery := fmt.Sprintf("%s %s LIMIT $1", queryTopQueries, orderBy)

	// 3. Execute
	rows, err := p.Pool.Query(context.Background(), finalQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch top queries (ensure pg_stat_statements is enabled): %w", err)
	}
	defer rows.Close()

	var stats []graph.QueryStats
	for rows.Next() {
		var q graph.QueryStats
		// Scan queryid as specific type? pgx handles it?
		// Ensure types match SQL: queryid (int8 or int64), query (text), calls (int8), total_time (float8), avg_time (float8), load_percent (float8)
		// Usually queryid is int64. Let's see.
		// If queryid is null? It's PK so shouldn't be.
		// Actually pg_stat_statements queryid is bigint.
		var qid int64
		if err := rows.Scan(&qid, &q.Query, &q.Calls, &q.TotalTime, &q.AvgTime, &q.LoadPercent); err != nil {
			return nil, err
		}
		q.QueryID = fmt.Sprintf("%d", qid)
		stats = append(stats, q)
	}

	return stats, nil
}

// TraceQuery executes a query with EXPLAIN (ANALYZE, BUFFERS) and returns performance data
func (p *PostgresAdapter) TraceQuery(query string) (*graph.TraceResult, error) {
	if p.Pool == nil {
		return nil, fmt.Errorf("database connection not established")
	}

	ctx := context.Background()

	// Start a transaction to ensure session-level settings (SET LOCAL) are applied to the query
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction for trace: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Apply Safety Wrappers
	// Kill trace if it looks like it will hang (>5s)
	if _, err := tx.Exec(ctx, "SET local statement_timeout = '5000ms'"); err != nil {
		return nil, fmt.Errorf("failed to set statement_timeout: %w", err)
	}
	// Limit memory usage
	if _, err := tx.Exec(ctx, "SET local work_mem = '64MB'"); err != nil {
		return nil, fmt.Errorf("failed to set work_mem: %w", err)
	}

	// 2. Prepare EXPLAIN command
	traceSQL := fmt.Sprintf("EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) %s", query)

	// 3. Execute
	var jsonOutput []byte
	err = tx.QueryRow(ctx, traceSQL).Scan(&jsonOutput)
	if err != nil {
		return nil, fmt.Errorf("trace execution failed: %w", err)
	}

	// 4. Parse JSON
	var explainParams []graph.ExplainOutput
	if err := json.Unmarshal(jsonOutput, &explainParams); err != nil {
		return nil, fmt.Errorf("failed to parse explain json: %w", err)
	}

	if len(explainParams) == 0 {
		return nil, fmt.Errorf("empty explain result")
	}

	result := explainParams[0]
	root := result.Plan

	// 5. Aggregate Stats
	traceResult := &graph.TraceResult{
		PlanningTime:  result.PlanningTime,
		ExecutionTime: result.ExecutionTime,
		TotalTime:     result.PlanningTime + result.ExecutionTime,
		Root:          root,
	}

	// Aggregate I/O from the tree (Recursively)
	var walk func(node *graph.ExplainNode)
	walk = func(node *graph.ExplainNode) {
		if node == nil {
			return
		}

		traceResult.CacheHits += node.SharedHitBlocks
		traceResult.DiskReads += node.SharedReadBlocks
		// Also include Local/Temp if needed
		// traceResult.MemoryUsage += ...

		for _, child := range node.Plans {
			walk(child)
		}
	}
	walk(root)

	return traceResult, nil
}
