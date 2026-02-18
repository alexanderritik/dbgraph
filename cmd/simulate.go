package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexanderritik/dbgraph/internal/adapters"
	"github.com/alexanderritik/dbgraph/internal/graph"
	"github.com/spf13/cobra"
)

// simulateCmd represents the simulate command
var simulateCmd = &cobra.Command{
	Use:   "simulate",
	Short: "Simulate schema changes and predict impact",
	Long:  `Simulates schema changes (like dropping a column) and reports impacted database objects using strict dependency analysis and code scanning.`,
	Run: func(cmd *cobra.Command, args []string) {
		ensureDBConnection()

		dropCol, _ := cmd.Flags().GetString("drop-column")
		dropTbl, _ := cmd.Flags().GetString("drop-table")

		if dropCol == "" && dropTbl == "" {
			fmt.Println("Error: Either --drop-column or --drop-table flag is required")
			os.Exit(1)
		}
		if dropCol != "" && dropTbl != "" {
			fmt.Println("Error: --drop-column and --drop-table are mutually exclusive")
			os.Exit(1)
		}

		// Initialize Adapter
		adapter, err := adapters.NewAdapter(dbUrl)
		if err != nil {
			fmt.Printf("Error creating adapter: %v\n", err)
			os.Exit(1)
		}
		defer adapter.Close()

		if err := adapter.Connect(dbUrl); err != nil {
			fmt.Printf("Error connecting to database: %v\n", err)
			os.Exit(1)
		}

		var deps []graph.ColumnDependency
		var schema, targetLabel string

		if dropCol != "" {
			// Parse table.column or schema.table.column
			parts := strings.Split(dropCol, ".")
			var table, column string
			if len(parts) == 3 {
				schema = parts[0]
				table = parts[1]
				column = parts[2]
			} else if len(parts) == 2 {
				schema = "public" // Default
				table = parts[0]
				column = parts[1]
			} else {
				fmt.Println("Error: Invalid format for --drop-column. Use 'table.column' or 'schema.table.column'")
				os.Exit(1)
			}
			targetLabel = fmt.Sprintf("%s.%s.%s", schema, table, column)
			fmt.Printf("🧪 Simulating DROP COLUMN on %s...\n", targetLabel)

			deps, err = adapter.GetColumnDependencies(schema, table, column)
		} else {
			// DROP TABLE
			parts := strings.Split(dropTbl, ".")
			var table string
			if len(parts) == 2 {
				schema = parts[0]
				table = parts[1]
			} else if len(parts) == 1 {
				schema = "public"
				table = parts[0]
			} else {
				fmt.Println("Error: Invalid format for --drop-table. Use 'table' or 'schema.table'")
				os.Exit(1)
			}
			targetLabel = fmt.Sprintf("%s.%s", schema, table)
			fmt.Printf("🧪 Simulating DROP TABLE on %s...\n", targetLabel)

			deps, err = adapter.GetTableDependencies(schema, table)
		}

		if err != nil {
			fmt.Printf("Error analyzing dependencies: %v\n", err)
			os.Exit(1)
		}

		// Print Report
		printSafetyVerdict(targetLabel, deps)
	},
}

func printSafetyVerdict(target string, deps []graph.ColumnDependency) {
	if len(deps) == 0 {
		fmt.Printf("\n%s\n└── (Safe to drop - No dependencies found) ✅\n\n", target)
		return
	}

	fmt.Printf("\n%s\n", target)
	for _, dep := range deps {
		desc := dep.Detail
		if dep.Type == "VIEW" && strings.Contains(desc, "Deep Dependency") {
			desc = "View Dependency"
		} else if dep.Type == "FUNCTION" && strings.Contains(desc, "Regex") {
			desc = "Used in Function Body"
		} else if dep.Type == "FOREIGN_KEY" {
			desc = "Foreign Key Constraint"
		} else if dep.Type == "INDEX" {
			desc = "Used in Index"
		} else if dep.Type == "RELATION" {
			desc = "Dependent Object"
		}

		fmt.Printf("└── %s (%s)\n", dep.Name, desc)
	}
	fmt.Println()
}

func init() {
	rootCmd.AddCommand(simulateCmd)
	simulateCmd.Flags().String("drop-column", "", "Column to simulate dropping (format: table.column)")
	simulateCmd.Flags().String("drop-table", "", "Table to simulate dropping (format: table)")
}
