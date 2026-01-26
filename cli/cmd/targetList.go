package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// targetListCmd represents the targetList command
var targetListCmd = &cobra.Command{
	Use:   "target-list",
	Short: "List all configured huskyCI API targets",
	Long: `List all configured huskyCI API targets.

The current active target is marked with an asterisk (*).

Examples:
  # List all targets
  huskyci target-list

  # Output format:
  # * production (https://api.huskyci.example.com)
  #   staging (https://staging-api.huskyci.example.com)
`,
	Run: func(cmd *cobra.Command, args []string) {

		targets := viper.GetStringMap("targets")
		
		if len(targets) == 0 {
			fmt.Println("No targets configured.")
			fmt.Println("\nTip: Use 'huskyci target-add <name> <endpoint>' to add a new target")
			fmt.Println("Example: huskyci target-add production https://api.huskyci.example.com")
			return
		}

		fmt.Println("Configured targets:")
		fmt.Println()
		for k, v := range targets {
			target := v.(map[string]interface{})

			// format output for activated target
			marker := " "
			if target["current"].(bool) {
				marker = "*"
			}

			fmt.Printf("  %s %s (%s)\n", marker, k, target["endpoint"])
		}
		fmt.Println()
		fmt.Println("Legend: * = current target")
	},
}

func init() {
	rootCmd.AddCommand(targetListCmd)
}
