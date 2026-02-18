package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/alexanderritik/dbgraph/internal/adapters"
	"github.com/alexanderritik/dbgraph/internal/graph"
	"github.com/spf13/cobra"
)

var (
	topInterval int
	topSort     string
	topLimit    int
	topWatch    bool
)

// topCmd represents the top command
var topCmd = &cobra.Command{
	Use:   "top",
	Short: "Real-time query performance monitoring (like htop)",
	Long:  `Displays a ranking of the most resource-intensive queries in real-time.`,
	Run: func(cmd *cobra.Command, args []string) {
		ensureDBConnection()

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

		// Contextual Analysis: Fetch basic schema for mapping names
		// We use a simple lightweight graph for context mapping
		g := graph.NewGraph()
		// Suppress errors for context fetching, it's optional flair
		_ = a.FetchSchema(g)

		// Loop
		for {
			// Clear Screen if watching
			if topWatch {
				c := exec.Command("clear")
				c.Stdout = os.Stdout
				c.Run()
			}

			// Header
			fmt.Printf("⏱️  Sampling: %ds | Sort: %s | Mode: Cumulative stats\n", topInterval, topSort)
			fmt.Println(strings.Repeat("-", 80))

			// Fetch Data
			queries, err := a.GetTopQueries(topLimit, topSort)
			if err != nil {
				fmt.Printf("Error fetching queries: %v\n", err)
				if !topWatch {
					os.Exit(1)
				}
				time.Sleep(time.Duration(topInterval) * time.Second)
				continue
			}

			if len(queries) == 0 {
				fmt.Println("No queries recorded yet.")
			} else {
				// 1. Render Summary Table
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
				fmt.Fprintln(w, "RANK\tLOAD %\tTIME (ms)\tCALLS\tAVG (ms)\tQUERY PREVIEW")
				fmt.Fprintln(w, "----\t------\t---------\t-----\t--------\t-------------")

				for i, q := range queries {
					// Preview: truncate nicely
					preview := strings.ReplaceAll(q.Query, "\n", " ")
					preview = strings.Join(strings.Fields(preview), " ") // normalize spaces
					preview = truncate(preview, 50)

					fmt.Fprintf(w, "%d\t%.2f\t%.2f\t%d\t%.2f\t%s\n",
						i+1, q.LoadPercent, q.TotalTime, q.Calls, q.AvgTime, preview)
				}
				w.Flush()

				// 2. Render Details
				fmt.Println()
				fmt.Println(strings.Repeat("-", 80))
				fmt.Println("QUERY DETAILS")
				fmt.Println(strings.Repeat("-", 80))
				fmt.Println()

				for i, q := range queries {
					fmt.Printf("[RANK %d]\n", i+1)

					// Basic syntax highlighting (very poor man's)
					formattedQuery := q.Query
					formattedQuery = strings.ReplaceAll(formattedQuery, "SELECT", "\033[1;34mSELECT\033[0m")
					formattedQuery = strings.ReplaceAll(formattedQuery, "FROM", "\033[1;34mFROM\033[0m")
					formattedQuery = strings.ReplaceAll(formattedQuery, "WHERE", "\033[1;34mWHERE\033[0m")
					formattedQuery = strings.ReplaceAll(formattedQuery, "JOIN", "\033[1;34mJOIN\033[0m")
					formattedQuery = strings.ReplaceAll(formattedQuery, "LEFT", "\033[1;34mLEFT\033[0m")
					formattedQuery = strings.ReplaceAll(formattedQuery, "GROUP BY", "\033[1;34mGROUP BY\033[0m")
					formattedQuery = strings.ReplaceAll(formattedQuery, "ORDER BY", "\033[1;34mORDER BY\033[0m")
					formattedQuery = strings.ReplaceAll(formattedQuery, "WITH", "\033[1;34mWITH\033[0m")
					fmt.Println(formattedQuery)

					// Context detection
					var contexts []string
					upperQ := strings.ToUpper(q.Query)
					for _, node := range g.Nodes {
						if strings.Contains(upperQ, strings.ToUpper(node.Name)) {
							contexts = append(contexts, fmt.Sprintf("%s (%s)", node.Name, node.Type))
						}
					}
					if len(contexts) > 0 {
						// Deduplicate
						unique := make(map[string]bool)
						var clean []string
						for _, c := range contexts {
							if !unique[c] {
								unique[c] = true
								clean = append(clean, c)
							}
						}
						// Limit context output
						if len(clean) > 5 {
							clean = clean[:5]
							clean = append(clean, "...")
						}
						fmt.Printf("\nLOCAL CONTEXT: %s\n", strings.Join(clean, ", "))
					}

					fmt.Println()
				}
				fmt.Println(strings.Repeat("-", 80))
			}

			if !topWatch {
				break
			}
			time.Sleep(time.Duration(topInterval) * time.Second)
		}
	},
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

func init() {
	rootCmd.AddCommand(topCmd)
	topCmd.Flags().IntVar(&topInterval, "interval", 5, "Seconds between refreshes")
	topCmd.Flags().StringVar(&topSort, "sort", "total", "Sort by total_time, calls, or avg_time")
	topCmd.Flags().IntVar(&topLimit, "limit", 10, "How many queries to show")

	topCmd.Flags().BoolVar(&topWatch, "watch", false, "Live watch mode")
}
