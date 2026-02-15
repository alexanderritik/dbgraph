package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of dbgraph",
	Long:  `All software has versions. This is dbgraph's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("dbgraph v0.1.0")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
