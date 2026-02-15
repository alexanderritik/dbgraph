package cmd

import (
	"fmt"
	"os"

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

		fmt.Println(e.GetGraphStats())
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
	analyzeCmd.Flags().StringVar(&dbUrl, "db", "", "Database connection string (postgres://user:pass@host:port/dbname)")
}
