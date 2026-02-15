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

func init() {
	// Add global flags here if needed
}
