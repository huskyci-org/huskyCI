package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// targetSetCmd represents the targetSet command
var targetSetCmd = &cobra.Command{
	Use:   "target-set [name]",
	Short: "Set a target as the current active target",
	Long: `Set a target as the current active target for huskyCI operations.

This will make the specified target the default one used for all huskyCI commands.

Examples:
  # Set 'production' as the current target
  huskyci target-set production

  # Set 'staging' as the current target
  huskyci target-set staging`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		// check if target name is used
		notUsed := true
		endpoint := ""
		targets := viper.GetStringMap("targets")
		for k, v := range targets {
			if k == args[0] {
				notUsed = false
				// set target as current
				target := v.(map[string]interface{})
				target["current"] = true
				endpoint = target["endpoint"].(string)
			} else {
				// unset all others targets as not current
				target := v.(map[string]interface{})
				target["current"] = false
			}
		}
		if notUsed {
			return fmt.Errorf("target '%s' does not exist\n\nTip: Use 'huskyci target-list' to see available targets, or 'huskyci target-add' to add a new one", args[0])
		}

		// save config (only if target is found)
		err := viper.WriteConfig()
		if err != nil {
			return fmt.Errorf("error saving configuration: %w\n\nTip: Check if you have write permissions to the config file", err)
		}
		fmt.Printf("✓ Successfully set '%s' (%s) as the current target\n", args[0], endpoint)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(targetSetCmd)
}
