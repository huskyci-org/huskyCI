package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the huskyCI CLI version",
	Long: `Print the version information for the huskyCI command-line tool.

Examples:
  # Show version
  huskyci version`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("huskyCI CLI version: 0.12")
		fmt.Println("For more information, visit: https://github.com/huskyci-org/huskyCI")
	},
}
