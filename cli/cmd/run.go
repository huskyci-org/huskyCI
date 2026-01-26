package cmd

import (
	"errors"
	"fmt"

	"github.com/huskyci-org/huskyCI/cli/analysis"
	"github.com/huskyci-org/huskyCI/cli/errorcli"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [path]",
	Short: "Run a security analysis on a local directory",
	Long: `Run a security analysis on a local directory or file path.

This command will:
  1. Scan the directory for supported programming languages
  2. Compress the code
  3. Send it to the huskyCI API for analysis
  4. Monitor the analysis progress
  5. Display the results

Examples:
  # Analyze current directory
  huskyci run .

  # Analyze a specific directory
  huskyci run ./my-project

  # Analyze a specific subdirectory
  huskyci run ./src/main`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("path argument is required\n\nExample: huskyci run ./my-project")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		pathReceived := args[0]
		currentAnalysis := analysis.New()

		fmt.Println()
		if err := currentAnalysis.CheckPath(pathReceived); err != nil {
			errorcli.Handle(err)
		}

		fmt.Println()
		if err := currentAnalysis.CompressFiles(pathReceived); err != nil {
			errorcli.Handle(err)
		}

		fmt.Println()
		if err := currentAnalysis.SendZip(); err != nil {
			errorcli.Handle(err)
		}

		fmt.Println()
		if err := currentAnalysis.CheckStatus(); err != nil {
			errorcli.Handle(err)
		}

		fmt.Println()
		currentAnalysis.PrintVulns()

		if err := currentAnalysis.HouseCleaning(); err != nil {
			errorcli.Handle(err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
