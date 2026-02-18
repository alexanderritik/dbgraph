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

var (
	showAll   bool
	limitRows int
)

// summaryCmd represents the summary command
var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Print a high-level architectural summary of the database",
	Long:  `Displays a ranked table of database objects based on their topological impact (Centrality), Risk, and Connectivity.`,
	Run: func(cmd *cobra.Command, args []string) {
		ensureDBConnection()

		g := graph.NewGraph()
		// ... existing code ...
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

		// Perform Analysis
		stats := g.AnalyzeTopology()

		fmt.Println("\n📊 ARCHITECTURAL TOPOLOGY (Top Impact)")
		fmt.Println(strings.Repeat("-", 80))
		fmt.Printf("%-30s %-10s %-10s %-10s %-10s %-10s\n", "OBJECT NAME", "TYPE", "IN/OUT", "ROWS", "IMPACT", "RISK")
		fmt.Println(strings.Repeat("-", 80))

		count := 0
		limit := 10
		if limitRows > 0 {
			limit = limitRows
		}
		if showAll {
			limit = len(stats.TopNodes)
		}

		for _, n := range stats.TopNodes {
			if count >= limit {
				break
			}
			// Risk Calculation
			risk := "LOW"
			if n.Centrality > 5 {
				risk = "MED"
			}
			if n.Centrality > 10 {
				risk = "HIGH"
			}
			if n.InDegree > 5 && n.OutDegree > 2 {
				risk = "CRITICAL"
			}

			// Row Count Formatting
			rowStr := fmt.Sprintf("%d", n.Rows)
			if n.Type == graph.Trigger {
				rowStr = "-"
			} else if n.Type == graph.View {
				// Standard Views (0 rows) -> "-"
				// Materialized Views (>0 rows) -> Show count
				if n.Rows == 0 {
					rowStr = "-"
				}
			}

			// Type formatting
			t := string(n.Type)
			if len(t) > 8 {
				t = t[:8]
			}

			inOut := fmt.Sprintf("%d/%d", n.InDegree, n.OutDegree)
			fmt.Printf("%-30s %-10s %-10s %-10s %-10.2f %-10s\n",
				n.ID, t, inOut, rowStr, n.Centrality, risk)
			count++
		}
		fmt.Println(strings.Repeat("-", 80))
		if !showAll && len(stats.TopNodes) > limit {
			fmt.Printf("... and %d more. Use --all or --limit to see more.\n", len(stats.TopNodes)-limit)
		}
	},
}

func init() {
	rootCmd.AddCommand(summaryCmd)
	summaryCmd.Flags().BoolVar(&showAll, "all", false, "Show all objects")
	summaryCmd.Flags().IntVar(&limitRows, "limit", 10, "Number of rows to show")
}
