// Copyright 2020 Globo.com authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/huskyci-org/huskyCI/cli/pkg/github"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with GitHub using device flow",
	Long: `Authenticate with GitHub using OAuth device flow.

This command will:
  1. Open your browser to authorize the application
  2. Display a user code for you to enter
  3. Save your access token for future use

Examples:
  # Login with GitHub
  huskyci login`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🔐 Starting GitHub authentication...")
		fmt.Println()
		
		client := &http.Client{Timeout: time.Minute}
		df := github.NewDeviceFlow(github.DefaultBaseURI, client)
		getCodesResp, err := df.GetCodes(&github.GetCodesRequest{
			ClientID: github.ClientID,
		})
		if err != nil {
			return fmt.Errorf("failed to initiate authentication: %w\n\nTip: Check your internet connection and try again", err)
		}

		fmt.Printf("📱 User code: %s\n", getCodesResp.UserCode)
		fmt.Printf("🌐 Opening browser to: %s\n", getCodesResp.VerificationURI)
		fmt.Println()
		
		if err := browser.OpenURL(getCodesResp.VerificationURI); err != nil {
			fmt.Printf("⚠️  Could not open browser automatically. Please visit:\n   %s\n", getCodesResp.VerificationURI)
			fmt.Println()
		}

		fmt.Println("Please:")
		fmt.Println("  1. Enter the user code shown above in the browser")
		fmt.Println("  2. Authorize the application")
		fmt.Println("  3. Press Enter here to continue...")
		fmt.Print("\nPress Enter when done...")
		if _, err := bufio.NewReader(os.Stdin).ReadBytes('\n'); err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}

		fmt.Println("\n⏳ Verifying authorization...")
		resp, err := df.GetAccessToken(&github.GetAccessTokenRequest{
			ClientID:   github.ClientID,
			DeviceCode: getCodesResp.DeviceCode,
			GrantType:  github.GrantTypeDeviceCode,
		})
		if err != nil {
			return fmt.Errorf("authentication failed: %w\n\nTip: Make sure you authorized the application in the browser", err)
		}

		if err := os.WriteFile(".huskyci", []byte(resp.AccessToken), 0600); err != nil {
			return fmt.Errorf("error saving access token: %w\n\nTip: Check if you have write permissions in the current directory", err)
		}

		fmt.Println("✓ Login successful! 🚀")
		fmt.Println("\nYour access token has been saved.")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
