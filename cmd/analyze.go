package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexanderritik/dbgraph/internal/adapters"
	"github.com/alexanderritik/dbgraph/internal/engine"
	"github.com/alexanderritik/dbgraph/internal/graph"

	"github.com/spf13/cobra"
)

var dbUrl string

// analyzeCmd represents the analyze command
var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze the database schema and build a dependency graph",
	Long:  `Connects to the database, fetches the schema (FKs and Views), and constructs an in-memory DAG.`,
	Run: func(cmd *cobra.Command, args []string) {
		if dbUrl == "" {
			fmt.Println("Error: --db flag is required")
			os.Exit(1)
		}

		g := graph.NewGraph()

		// Use the factory to get the correct adapter
		a, err := adapters.NewAdapter(dbUrl)
		if err != nil {
			fmt.Printf("Error creating adapter: %v\n", err)
			os.Exit(1)
		}

		e := engine.NewEngine(g, a)
		defer a.Close()

		if err := e.Connect(dbUrl); err != nil {
			fmt.Printf("Error connecting to database: %v\n", err)
			os.Exit(1)
		}

		if err := e.BuildGraph(); err != nil {
			fmt.Printf("Error building graph: %v\n", err)
			os.Exit(1)
		}

		// Perform Topological Analysis
		stats := g.AnalyzeTopology()

		// DB Version (Mock for now or fetch if we added it specifically to return value)
		// Ideally adapter returns this, but we'll print what we have.
		dbName := dbUrl
		if strings.Contains(dbUrl, "/") {
			parts := strings.Split(dbUrl, "/")
			lastPart := parts[len(parts)-1]
			if strings.Contains(lastPart, "?") {
				dbName = strings.Split(lastPart, "?")[0]
			} else {
				dbName = lastPart
			}
		}
		fmt.Printf("🔍 DB: %s | Objects: %d\n", dbName, stats.Nodes)
		fmt.Println(strings.Repeat("-", 80))

		fmt.Println("\n🏗️  TOPOLOGICAL CONTEXT")
		fmt.Printf("Graph Type:  Directed Multigraph\n")
		denseLabel := "Sparse"
		if stats.Density > 0.1 {
			denseLabel = "Dense"
		}
		fmt.Printf("Density:     %.3f (%s)\n", stats.Density, denseLabel)
		fmt.Printf("Components:  %d Isolated Sub-graphs\n", stats.Components)
		fmt.Printf("Centrality:  %s (%.2f)\n", stats.CentralNode, stats.MaxCentrality)

		fmt.Println("\n📦 OBJECT DISTRIBUTION")
		tables := 0
		views := 0
		triggers := 0
		for _, n := range g.Nodes {
			switch n.Type {
			case graph.Table:
				tables++
			case graph.View:
				views++
			case graph.Trigger:
				triggers++
			}
		}
		fmt.Printf("Tables:      %d\n", tables)
		fmt.Printf("Views:       %d\n", views)
		fmt.Printf("Triggers:    %d\n", triggers)

		fmt.Println("\n🔗 DEPENDENCY VECTORS")
		fkCount := 0
		viewCount := 0
		triggerCount := 0
		for _, edges := range g.Edges {
			for _, e := range edges {
				switch e.Type {
				case graph.ForeignKey:
					fkCount++
				case graph.ViewDepends:
					viewCount++
				case graph.TriggerAction:
					triggerCount++
				}
			}
		}
		fmt.Printf("Foreign Keys:       %d edges\n", fkCount)
		fmt.Printf("View Definitions:    %d edges\n", viewCount)
		fmt.Printf("Trigger Actions:     %d edges\n", triggerCount)

		fmt.Println("\n🛰️  ISOLATED SUB-GRAPHS (Island Detection)")
		for i, iso := range stats.IsolatedGroups {
			if i >= 5 {
				break
			} // Limit output
			fmt.Printf("%d. Cluster:  %s\n", i+1, iso)
		}

		fmt.Println("\n🧵 SCHEMA LINEAGE DEPTH")
		fmt.Printf("Deepest Chain:  %d Levels\n", stats.LongestPath)

		// --- HEALTH CHECK ---
		fmt.Println("\n🏥 SCHEMA HEALTH REPORT")
		fmt.Println(strings.Repeat("-", 80))

		cycles := g.CheckCycles()
		if len(cycles) > 0 {
			fmt.Printf("🔴 CRITICAL: Found %d circular dependencies (Cycles)!\n", len(cycles))
			for i, c := range cycles {
				fmt.Printf("   %d. %v\n", i+1, c)
			}
		} else {
			fmt.Println("✅ Great! No circular dependencies detected.")
		}

		// --- INDEX HYGIENE ---
		idxStats := g.CheckIndexCoverage()
		if len(idxStats.MissingFKIndexes) > 0 {
			fmt.Printf("\n⚠️  PERFORMANCE RISKS: Found %d FKs missing indexes\n", len(idxStats.MissingFKIndexes))
			for i, miss := range idxStats.MissingFKIndexes {
				if i >= 5 {
					fmt.Printf("   ... and %d more\n", len(idxStats.MissingFKIndexes)-5)
					break
				}
				fmt.Printf("   - %s\n", miss)
			}
			fmt.Printf("   (Suggestion: Add indexes to valid FK columns to prevent locking issues)\n")
		} else if idxStats.TotalFKs > 0 {
			fmt.Printf("\n✅ Index Hygiene: Excellent! All %d FKs are indexed.\n", idxStats.TotalFKs)
		} else {
			fmt.Println("\nℹ️  No Foreign Keys found to check.")
		}

		// --- COMPLEXITY / GOD OBJECTS ---
		gods := g.DetectGodObjects()
		if len(gods) > 0 {
			fmt.Printf("\n😈 COMPLEXITY RISKS: Found %d 'God Objects' (High Coupling)\n", len(gods))
			for _, god := range gods {
				fmt.Printf("   - %s (Connected to %d others: %d in, %d out)\n",
					god.ID, god.Degree, god.Dependents, god.Dependencies)
			}
			fmt.Printf("   (Suggestion: Consider splitting these tables to reduce architectural coupling)\n")
		} else {
			fmt.Println("\n✅ Architecture: No 'God Objects' detected (Clean Separation).")
		}

		fmt.Println(strings.Repeat("-", 80))
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
	analyzeCmd.Flags().StringVar(&dbUrl, "db", "", "Database connection string (postgres://user:pass@host:port/dbname)")
}
