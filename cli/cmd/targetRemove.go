package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// targetRemoveCmd represents the targetRemove command
var targetRemoveCmd = &cobra.Command{
	Use:   "target-remove [name]",
	Short: "Remove a target from the target list",
	Long: `Remove a target from the list of available targets.

Examples:
  # Remove a target named 'staging'
  huskyci target-remove staging

  # Remove a target named 'old-production'
  huskyci target-remove old-production`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		// check if target name is used
		notUsed := true
		targets := viper.GetStringMap("targets")
		for k := range targets {
			if k == args[0] {
				notUsed = false
			}
		}
		if notUsed {
			return fmt.Errorf("target '%s' does not exist\n\nTip: Use 'huskyci target-list' to see available targets", args[0])
		}

		// remove entry from data struct but, before, storing data to show to user
		target := targets[args[0]].(map[string]interface{})
		endpoint := target["endpoint"].(string)
		targets[args[0]] = nil

		// save config
		err := viper.WriteConfig()
		if err != nil {
			return fmt.Errorf("error saving configuration: %w\n\nTip: Check if you have write permissions to the config file", err)
		}

		fmt.Printf("✓ Successfully removed target '%s' (%s) from target list\n", args[0], endpoint)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(targetRemoveCmd)
}
