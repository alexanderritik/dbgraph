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

// impactCmd represents the impact command
var impactCmd = &cobra.Command{
	Use:   "impact [table_name]",
	Short: "Identify downstream dependencies of a table",
	Long:  `Finds all database objects (tables, views) that depend on the specified table/view using the dependency graph.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		tableName := args[0]
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

		// Find ID for the table (we store it as schema.name)
		targetID := ""
		found := false

		// Simple heuristic: if input has dot, use as is. If not, try public.<input>.
		// Or better, search the nodes.
		for id, node := range g.Nodes {
			if node.Name == tableName || id == tableName {
				targetID = id
				found = true
				break
			}
		}

		if !found {
			fmt.Printf("Error: Table or View '%s' not found in the graph.\n", tableName)
			os.Exit(1)
		}

		// Parse DB Name for cleaner output
		// Simple string parsing or use net/url
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

		// High-Fidelity Output - DB Metrcs
		metrics, err := a.GetMetrics()
		if err != nil {
			// Ignore error, just show empty
			metrics = &graph.DBMetrics{}
		}

		nodeRows := g.Nodes[targetID].RowCount
		rowStr := fmt.Sprintf("%d rows", nodeRows)
		if nodeRows > 1000 {
			rowStr = fmt.Sprintf("%.1fk rows", float64(nodeRows)/1000.0)
		}
		if nodeRows > 1000000 {
			rowStr = fmt.Sprintf("%.1fm rows", float64(nodeRows)/1000000.0)
		}

		fmt.Printf("🔍 DB: %s | Target: %s (%s) | Active Locks: %d\n", dbName, targetID, rowStr, metrics.ActiveLocks)
		fmt.Println(strings.Repeat("-", 80))

		// 1. Calculate Metrics & Build Tree
		type TreeNode struct {
			ID       string
			Type     graph.NodeType
			Size     string
			RowCount int64
			EdgeMeta *graph.Edge
			Children []*TreeNode
			Level    int
		}

		var buildTree func(id string, level int, visited map[string]bool) *TreeNode
		totalAffected := 0
		warnings := []string{}
		structures := make(map[graph.NodeType]int)

		// Pre-compute reverse edges for traversal
		reverseEdges := make(map[string][]*graph.Edge)
		for _, edges := range g.Edges {
			for _, edge := range edges {
				reverseEdges[edge.TargetID] = append(reverseEdges[edge.TargetID], edge)
			}
		}

		buildTree = func(id string, level int, visited map[string]bool) *TreeNode {
			node := &TreeNode{
				ID:       id,
				Type:     g.Nodes[id].Type,
				Size:     g.Nodes[id].Size,
				RowCount: g.Nodes[id].RowCount,
				Level:    level,
			}

			if level > 0 {
				totalAffected++
				structures[node.Type]++
			}

			visited[id] = true

			// Find dependents
			for _, edge := range reverseEdges[id] {
				src := edge.SourceID
				if !visited[src] {
					child := buildTree(src, level+1, visited)
					child.EdgeMeta = edge
					node.Children = append(node.Children, child)

					// Detect Warnings
					// 1. Cascade
					if edge.DeleteRule == "CASCADE" {
						desc := fmt.Sprintf("[High] Cascade Delete: Deleting '%s' will recursively delete objects in '%s'", id, src)
						if node.RowCount > 1000 {
							desc += fmt.Sprintf(" (~%d rows potentially locked/deleted)", node.RowCount)
						}
						desc += "."
						warnings = append(warnings, desc)
					}
					// 2. View Coupling
					if edge.Type == graph.ViewDepends && level >= 2 {
						warnings = append(warnings, fmt.Sprintf("[Med] View Coupling: '%s' is %d levels removed but will break on schema change.", src, level))
					}
					// 3. Missing Index
					// Rule: Source (child in this tree view) has FK to Target (node).
					// Source should have index on FK columns.
					if edge.Type == graph.ForeignKey {
						cols, ok := edge.MetaData["fk_columns"]
						if ok && cols != "" {
							// Check if source node (table with FK) has index starting with these cols
							sourceNode := g.Nodes[src] // child
							hasIndex := false
							fkCols := strings.Split(cols, ",")

							for _, idxCols := range sourceNode.Indexes {
								// Check if fkCols is prefix of idxCols
								if len(idxCols) >= len(fkCols) {
									match := true
									for i, col := range fkCols {
										if idxCols[i] != col {
											match = false
											break
										}
									}
									if match {
										hasIndex = true
										break
									}
								}
							}

							if !hasIndex {
								warnings = append(warnings, fmt.Sprintf("[Med] Missing Index: '%s(%s)' is not indexed. Cascade/Delete operations will be slow.", src, cols))
							}
						}
					}
				}
			}
			return node
		}

		root := buildTree(targetID, 0, make(map[string]bool))

		// Wrapper for depth calculation
		var getDepth func(n *TreeNode) int
		getDepth = func(n *TreeNode) int {
			if len(n.Children) == 0 {
				return 0
			}
			max := 0
			for _, child := range n.Children {
				d := getDepth(child)
				if d > max {
					max = d
				}
			}
			return 1 + max
		}

		// 2. Print Metrics
		fmt.Printf("\n📊 IMPACT RADIUS: %d Levels Deep [Load: 🔥 System Active]\n", getDepth(root))
		fmt.Printf("Total Affected Objects: %d (", totalAffected)
		counts := []string{}
		for t, c := range structures {
			counts = append(counts, fmt.Sprintf("%d %ss", c, t))
		}
		fmt.Printf("%s)\n", strings.Join(counts, ", "))
		fmt.Println("\nTREE VIEW")

		// 3. Print Tree
		var printTree func(node *TreeNode, prefix string, isLast bool)
		printTree = func(node *TreeNode, prefix string, isLast bool) {
			marker := "├──"
			if isLast {
				marker = "└──"
			}
			if node.Level == 0 {
				// Root
				fRowStr := fmt.Sprintf("%d rows", node.RowCount)
				if node.RowCount > 1000 {
					fRowStr = fmt.Sprintf("%.1fk rows", float64(node.RowCount)/1000.0)
				}
				fmt.Printf("%s (%s)\n", node.ID, fRowStr)
			} else {
				meta := ""
				if node.EdgeMeta != nil {
					if node.EdgeMeta.Type == graph.ForeignKey {
						meta = fmt.Sprintf("[FK: %s]", node.EdgeMeta.ConstraintName)
						if node.EdgeMeta.DeleteRule == "CASCADE" {
							meta += " (CASCADE)"
						}
					} else {
						meta = "(View)"
					}
				}
				icon := "📥"
				if node.Type == graph.View {
					icon = "👁️ "
				}

				fRowStr := ""
				if node.Type == graph.Table {
					fRowStr = fmt.Sprintf("(%d rows)", node.RowCount)
					if node.RowCount > 1000 {
						fRowStr = fmt.Sprintf("(%.1fk rows)", float64(node.RowCount)/1000.0)
					}
				}

				fmt.Printf("%s%s %s %s %s %s\n", prefix, marker, icon, node.ID, fRowStr, meta)
			}

			childPrefix := prefix
			if node.Level > 0 {
				if isLast {
					childPrefix += "    "
				} else {
					childPrefix += "│   "
				}
			}

			for i, child := range node.Children {
				printTree(child, childPrefix, i == len(node.Children)-1)
			}
		}
		printTree(root, "", true)

		// 4. Print Warnings
		if len(warnings) > 0 {
			fmt.Println("\n⚠️  STRUCTURAL WARNINGS")
			for _, w := range warnings {
				fmt.Println(w)
			}
		}

		// 5. Resource Metrics
		fmt.Println("\n📊 RESOURCE METRICS")
		satLabel := "(Low)"
		if strings.TrimSuffix(metrics.ConnSaturation, "%") > "80" {
			satLabel = "(High)"
		}
		fmt.Printf("Connection Saturation: %s %s\n", metrics.ConnSaturation, satLabel)
		fmt.Printf("Longest Running Query: %s\n", metrics.LongestQuery)
	},
}

func init() {
	rootCmd.AddCommand(impactCmd)
	impactCmd.Flags().StringVar(&dbUrl, "db", "", "Database connection string")
}
