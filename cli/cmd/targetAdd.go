package cmd

import (
	"fmt"
	"net/url"
	"regexp"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// targetAddCmd represents the targetAdd command
var targetAddCmd = &cobra.Command{
	Use:   "target-add [name] [endpoint]",
	Short: "Add a new huskyCI API endpoint to the target list",
	Long: `Add a new huskyCI API endpoint to the list of available targets.

The target name must contain only letters, numbers, and underscores.
The endpoint must be a valid URL (e.g., https://api.huskyci.example.com).

Examples:
  # Add a production target
  huskyci target-add production https://api.huskyci.example.com

  # Add a staging target and set it as current
  huskyci target-add staging https://staging-api.huskyci.example.com --set-current

  # Add a local development target
  huskyci target-add local http://localhost:8888`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {

		match, err := regexp.MatchString(`^\w+$`, args[0])
		if err != nil {
			return fmt.Errorf("error validating target name '%s': %w\n\nTip: Target name must contain only letters, numbers, and underscores", args[0], err)
		}
		if !match {
			return fmt.Errorf("invalid target name '%s': target name must contain only letters, numbers, and underscores\n\nExample: huskyci target-add production https://api.huskyci.example.com", args[0])
		}

		// check huskyci-api-endpoint
		parsedURL, err := url.Parse(args[1])
		if err != nil {
			return fmt.Errorf("invalid endpoint URL '%s': %w\n\nTip: Please provide a valid URL (e.g., https://api.huskyci.example.com)", args[1], err)
		}
		if parsedURL.Scheme == "" || parsedURL.Host == "" {
			return fmt.Errorf("invalid endpoint URL '%s': URL must include scheme (http:// or https://) and host\n\nExample: https://api.huskyci.example.com", args[1])
		}

		// check if target name is used before
		targets := viper.GetStringMap("targets")
		for k, v := range targets {
			if k == args[0] {
				target := v.(map[string]interface{})
				return fmt.Errorf("target name '%s' already exists with endpoint: %s\n\nTip: Use 'huskyci target-remove %s' to remove it first, or choose a different name", k, target["endpoint"], k)
			}
		}

		// if new target must be current, we unset all others
		setCurrent, err := cmd.Flags().GetBool("set-current")
		if err != nil {
			return fmt.Errorf("error parsing --set-current flag: %w", err)
		}

		if setCurrent {
			for _, v := range targets {
				target := v.(map[string]interface{})
				target["current"] = false
			}
		}

		// add new entry to data struct
		targets[args[0]] = map[string]interface{}{"current": setCurrent, "endpoint": args[1]}

		// save config
		viper.Set("targets", targets)
		err = viper.WriteConfig()
		if err != nil {
			return fmt.Errorf("error saving configuration: %w\n\nTip: Check if you have write permissions to the config file", err)
		}
		
		currentStatus := ""
		if setCurrent {
			currentStatus = " (set as current)"
		}
		fmt.Printf("✓ Successfully added target '%s' -> %s%s\n", args[0], args[1], currentStatus)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(targetAddCmd)
	targetAddCmd.Flags().BoolP("set-current", "s", false, "Add and define the target as the current target")
}
