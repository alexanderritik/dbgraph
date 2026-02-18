package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dbgraph",
	Short: "A graph-based database CLI",
	Long:  `dbgraph is a CLI tool for managing and querying graph data with pluggable database adapters.`,
	// Run: func(cmd *cobra.Command, args []string) { }, // output help by default
}

// Execute executes the root command
func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var dbUrl string

func init() {
	rootCmd.PersistentFlags().StringVar(&dbUrl, "db", "", "Database connection string (or env DBGRAPH_DB_URL)")
}

// ensureDBConnection checks if dbUrl is set, otherwise tries to read from env
func ensureDBConnection() {
	if dbUrl == "" {
		dbUrl = os.Getenv("DBGRAPH_DB_URL")
	}

	if dbUrl == "" {
		fmt.Println("Error: --db flag or DBGRAPH_DB_URL environment variable is required")
		os.Exit(1)
	}
}
