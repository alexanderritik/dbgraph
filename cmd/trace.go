package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexanderritik/dbgraph/internal/adapters"
	"github.com/alexanderritik/dbgraph/internal/graph"
	"github.com/spf13/cobra"
)

var (
	traceQueryString string
)

// traceCmd represents the trace command
var traceCmd = &cobra.Command{
	Use:   "trace",
	Short: "Analyze a SELECT query with detailed execution stats",
	Long:  `Executes a SELECT query with EXPLAIN (ANALYZE, BUFFERS) and visualizes the execution path, latency, and I/O efficiency.`,
	Run: func(cmd *cobra.Command, args []string) {
		if dbUrl == "" {
			fmt.Println("Error: --db flag is required")
			os.Exit(1)
		}
		if traceQueryString == "" {
			fmt.Println("Error: --query flag is required")
			os.Exit(1)
		}

		// Basic Safety Check (Client-side)
		upperQ := strings.ToUpper(strings.TrimSpace(traceQueryString))
		if !strings.HasPrefix(upperQ, "SELECT") && !strings.HasPrefix(upperQ, "WITH") {
			fmt.Println("Error: Only SELECT queries are supported for tracing.")
			os.Exit(1)
		}

		// Connect
		a, err := adapters.NewAdapter(dbUrl)
		if err != nil {
			fmt.Printf("Error creating adapter: %v\n", err)
			os.Exit(1)
		}
		defer a.Close()
		if err := a.Connect(dbUrl); err != nil {
			fmt.Printf("Error connecting to database: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("🔍 TRACE: Ad-hoc SELECT")
		fmt.Println(strings.Repeat("-", 80))

		// Execute Trace
		result, err := a.TraceQuery(traceQueryString)
		if err != nil {
			fmt.Printf("❌ Trace failed: %v\n", err)
			os.Exit(1)
		}

		// 1. Latency
		fmt.Println("⏱️  LATENCY")
		fmt.Printf("Planning Time:   %.2f ms\n", result.PlanningTime)
		fmt.Printf("Execution Time:  %.2f ms\n", result.ExecutionTime)
		fmt.Printf("Total Time:      %.2f ms\n", result.TotalTime)
		fmt.Println()

		// 2. I/O & Memory
		fmt.Println("💾 I/O & MEMORY (BUFFERS)")

		hits := result.CacheHits
		reads := result.DiskReads
		totalIO := hits + reads
		hitRate := 0.0
		if totalIO > 0 {
			hitRate = float64(hits) / float64(totalIO) * 100.0
		}

		fmt.Printf("Cache Hits:      %d  (%.1f%%)\n", hits, hitRate)
		if hits > 0 && reads == 0 {
			fmt.Println("                 ⚡ (Fast: Data found in Shared Buffers)")
		}

		fmt.Printf("Disk Reads:      %d\n", reads)
		if reads > 0 {
			fmt.Println("                 💾 (Slow: Physical I/O required)")
		}

		// Memory Usage (Approximation if we had it, for now placeholder if 0)
		// fmt.Printf("Memory Usage:    %d KB\n", result.MemoryUsage/1024)
		fmt.Println()

		// 3. Execution Path
		fmt.Println("🌳 EXECUTION PATH")
		fmt.Println(strings.Repeat("-", 80))
		printExplainTree(result.Root, "", true)
		fmt.Println(strings.Repeat("-", 80))

		// 4. Technical Detail / Tips
		fmt.Println("🧪 Technical Detail: The \"Shared Buffers\" Secret")
		if reads == 0 && hits > 0 {
			fmt.Println("This query is \"warm\". All data was found in RAM (Shared Buffers).")
		} else if reads > 0 {
			fmt.Println("This query is \"cold\" or data is too large for cache. Physical disk I/O was required.")
		} else {
			fmt.Println("No I/O activity recorded (likely constants or metadata query).")
		}
	},
}

func init() {
	rootCmd.AddCommand(traceCmd)
	traceCmd.Flags().StringVar(&dbUrl, "db", "", "Database connection string")
	traceCmd.Flags().StringVar(&traceQueryString, "query", "", "The SELECT query to trace")
}

// printExplainTree recursively prints the plan tree
func printExplainTree(node *graph.ExplainNode, prefix string, isLast bool) {
	if node == nil {
		return
	}

	// Marker
	marker := "->"
	if prefix != "" {
		if isLast {
			marker = "-> "
		} else {
			marker = "-> "
		}
	}

	// Cost/Rows
	costStr := fmt.Sprintf("(cost=%.2f..%.2f rows=%.0f)", node.StartupCost, node.TotalCost, node.PlanRows)

	// Node Description
	desc := node.Type
	if node.Strategy != "" {
		desc += fmt.Sprintf(" (%s)", node.Strategy)
	}
	if node.RelationName != "" {
		desc += fmt.Sprintf(" on %s", node.RelationName)
		if node.Alias != "" && node.Alias != node.RelationName {
			desc += fmt.Sprintf(" %s", node.Alias)
		}
	}

	fmt.Printf("%s%s %s %s\n", prefix, marker, desc, costStr)

	// Additional Details (Filter, Index Cond) indented

	// Actually typical tree printing uses pipes for siblings.
	// Let's stick to a simpler indentation for now given the complexity.
	// Standard `tree` command style:
	// prefix is the indentation inherited from parent.

	// Better tree prefix logic:
	// parent passes: `current_indent`
	// we print `current_indent` + `->`
	// children get: `current_indent` + `    ` (if I am last) or `|   ` (if I am not last)

	// Adjust prefix for children
	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "|   "
	}

	// Warning for Seq Scan
	if node.Type == "Seq Scan" {
		fmt.Printf("%s⚠️  Warning: Full table scan.\n", childPrefix)
	}

	if node.IndexCond != "" {
		fmt.Printf("%sIndex Cond: %s\n", childPrefix, node.IndexCond)
	}
	if node.Filter != "" {
		fmt.Printf("%sFilter: %s\n", childPrefix, node.Filter)
	}

	// Strategies / Extra info
	// if node.Strategy == "Hash" ...

	// Children
	count := len(node.Plans)
	for i, child := range node.Plans {
		printExplainTree(child, childPrefix, i == count-1)
	}
}
