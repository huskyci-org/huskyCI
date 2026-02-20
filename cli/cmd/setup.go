package cmd

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/huskyci-org/huskyCI/cli/config"
	"github.com/huskyci-org/huskyCI/cli/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	separatorLine = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
)

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard for huskyCI CLI",
	Long: `Interactive setup wizard that guides you through configuring huskyCI CLI.

This command will help you:
  1. Configure your huskyCI API endpoint
  2. Set up authentication (optional)
  3. Verify your connection
  4. Set up your first target

Examples:
  # Start the setup wizard
  huskyci setup

  # Run setup with non-interactive mode (for automation)
  huskyci setup --non-interactive`,
	RunE: func(cmd *cobra.Command, args []string) error {
		nonInteractive, _ := cmd.Flags().GetBool("non-interactive")
		wizard := newSetupWizard(nonInteractive)
		return wizard.run()
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
	setupCmd.Flags().Bool("non-interactive", false, "Run setup in non-interactive mode (for automation)")
}

// setupWizard manages the entire setup flow
type setupWizard struct {
	nonInteractive bool
	scanner        *bufio.Scanner
	existingTargets map[string]interface{}
}

func newSetupWizard(nonInteractive bool) *setupWizard {
	return &setupWizard{
		nonInteractive:  nonInteractive,
		scanner:         bufio.NewScanner(os.Stdin),
		existingTargets: viper.GetStringMap("targets"),
	}
}

// run executes the main setup wizard flow
func (w *setupWizard) run() error {
	w.printWelcome()

	// Handle existing targets if any
	if len(w.existingTargets) > 0 {
		if !w.handleExistingTargets() {
			return nil // User chose to exit
		}
	}

	// Main setup flow: endpoint -> target name -> token -> save -> verify
	endpoint, err := w.collectEndpoint()
	if err != nil {
		return err
	}

	targetName, err := w.collectTargetName()
	if err != nil {
		return err
	}

	token, useToken := w.collectToken()

	if err := w.saveTarget(targetName, endpoint); err != nil {
		return err
	}

	// If user provided a token, set it up
	if useToken && token != "" {
		w.setupToken(token)
	}

	w.verifyConnection(endpoint, token, useToken)
	w.printSummary(endpoint, useToken)

	return nil
}

// ============================================================================
// UI Components
// ============================================================================

func (w *setupWizard) printWelcome() {
	fmt.Println()
	fmt.Println(separatorLine)
	fmt.Println("  🐕  Welcome to huskyCI Setup Wizard!")
	fmt.Println(separatorLine)
	fmt.Println()
	fmt.Println("This wizard will help you configure huskyCI CLI to work with your huskyCI API.")
	fmt.Println()
}

func (w *setupWizard) printSection(title string) {
	fmt.Println()
	fmt.Println(title)
	fmt.Println(strings.Repeat("─", len(title)))
	fmt.Println()
}

func (w *setupWizard) printSuccess(message string) {
	fmt.Printf("✓ %s\n", message)
}

func (w *setupWizard) printWarning(message string) {
	fmt.Printf("⚠️  %s\n", message)
}

func (w *setupWizard) printError(message string) {
	fmt.Printf("✗ %s\n", message)
}

// ============================================================================
// Menu System
// ============================================================================

type menuOption struct {
	key         string
	description string
	handler     func() menuResult
}

type menuResult int

const (
	menuContinue menuResult = iota
	menuReturn
	menuExit
)

func (w *setupWizard) showMenu(title string, options []menuOption) menuResult {
	for {
		fmt.Println()
		if title != "" {
			fmt.Println(title)
		}
		for _, opt := range options {
			fmt.Printf("  %s. %s\n", opt.key, opt.description)
		}
		fmt.Println()
		fmt.Print("Enter your choice: ")

		if !w.scanner.Scan() {
			return menuExit
		}

		choice := strings.TrimSpace(w.scanner.Text())
		fmt.Println()

		for _, opt := range options {
			if opt.key == choice {
				return opt.handler()
			}
		}

		w.printWarning("Invalid choice. Please try again.")
	}
}

// ============================================================================
// Existing Targets Handling
// ============================================================================

func (w *setupWizard) handleExistingTargets() bool {
	if w.nonInteractive {
		return true
	}

	w.printSection("Existing Configuration")
	fmt.Println("You already have some targets configured:")
	fmt.Println()

	for name, v := range w.existingTargets {
		target := v.(map[string]interface{})
		current := ""
		if target["current"] != nil && target["current"].(bool) {
			current = " (current)"
		}
		fmt.Printf("  • %s: %s%s\n", name, target["endpoint"], current)
	}
	fmt.Println()

	result := w.showMenu("What would you like to do?", []menuOption{
		{"1", "Add a new target", func() menuResult {
			return menuContinue
		}},
		{"2", "Test connection to current target", func() menuResult {
			return w.handleTestConnection()
		}},
		{"3", "Configure authentication token", func() menuResult {
			return w.handleConfigureToken()
		}},
		{"4", "View current configuration", func() menuResult {
			return w.handleViewConfiguration()
		}},
		{"5", "Exit", func() menuResult {
			w.printSuccess("Exiting setup wizard.")
			fmt.Println("  Any changes made during this session have been applied.")
			return menuExit
		}},
	})

	return result == menuContinue
}

func (w *setupWizard) handleTestConnection() menuResult {
	if _, err := config.GetCurrentTarget(); err != nil {
		w.printError(fmt.Sprintf("Error getting current target: %v", err))
		fmt.Println()
		return w.askNextAction()
	}

	fmt.Println("Running connection test...")
	fmt.Println()

	if err := runConnectionTest("", "", false); err != nil {
		fmt.Println()
		return w.askNextAction()
	}

	fmt.Println()
	return w.askNextAction()
}

func (w *setupWizard) handleConfigureToken() menuResult {
	w.printSection("Token Configuration")
	fmt.Println("How would you like to configure your authentication token?")
	fmt.Println()

	result := w.showMenu("", []menuOption{
		{"1", "Enter token manually", func() menuResult {
			return w.handleManualToken()
		}},
		{"2", "Generate token via API", func() menuResult {
			return w.handleGenerateToken()
		}},
	})

	if result == menuExit {
		return menuExit
	}
	return w.askNextAction()
}

func (w *setupWizard) handleManualToken() menuResult {
	w.printSection("Manual Token Configuration")
	fmt.Print("Enter your huskyCI API token: ")

	if !w.scanner.Scan() {
		return menuExit
	}

	token := strings.TrimSpace(w.scanner.Text())
	if token == "" {
		w.printWarning("No token provided. Token configuration cancelled.")
		fmt.Println()
		return menuReturn
	}

	w.setupToken(token)
	return menuContinue
}

func (w *setupWizard) handleGenerateToken() menuResult {
	w.printSection("Generate Token via API")

	target, err := config.GetCurrentTarget()
	if err != nil {
		w.printError(fmt.Sprintf("Error getting current target: %v", err))
		fmt.Println("  Please configure a target first using option 1.")
		fmt.Println()
		return menuReturn
	}

	endpoint := normalizeURL(target.Endpoint)
	token, err := w.generateTokenFromAPI(endpoint)
	if err != nil {
		w.printError(err.Error())
		fmt.Println()
		return menuReturn
	}

	w.setupToken(token)
	return menuContinue
}

func (w *setupWizard) handleViewConfiguration() menuResult {
	w.printSection("Current Configuration")

	target, err := config.GetCurrentTarget()
	if err == nil {
		fmt.Printf("Current Target: %s\n", target.Label)
		fmt.Printf("Endpoint: %s\n", target.Endpoint)
		if target.Token != "" {
			fmt.Printf("Token: %s... (configured)\n", target.Token[:min(10, len(target.Token))])
		} else {
			fmt.Println("Token: Not configured")
		}
		fmt.Println()
	}

	fmt.Println("All Targets:")
	for name, v := range w.existingTargets {
		target := v.(map[string]interface{})
		current := ""
		if target["current"] != nil && target["current"].(bool) {
			current = " (current)"
		}
		fmt.Printf("  • %s: %s%s\n", name, target["endpoint"], current)
	}
	fmt.Println()

	if os.Getenv("HUSKYCI_CLIENT_API_ADDR") != "" {
		fmt.Println("Environment Variables:")
		fmt.Printf("  HUSKYCI_CLIENT_API_ADDR: %s\n", os.Getenv("HUSKYCI_CLIENT_API_ADDR"))
		token := config.GetTokenFromEnv()
		if token != "" {
			fmt.Printf("  HUSKYCI_CLI_TOKEN: %s... (configured)\n", token[:min(10, len(token))])
		}
		fmt.Println()
	}

	return w.askNextAction()
}

func (w *setupWizard) askNextAction() menuResult {
	return w.showMenu("What would you like to do next?", []menuOption{
		{"1", "Return to main menu", func() menuResult {
			return menuReturn
		}},
		{"2", "Add a new target", func() menuResult {
			return menuContinue
		}},
		{"3", "Exit", func() menuResult {
			w.printSuccess("Exiting setup wizard.")
			fmt.Println("  Any changes made during this session have been applied.")
			return menuExit
		}},
	})
}

// ============================================================================
// Data Collection
// ============================================================================

func (w *setupWizard) collectEndpoint() (string, error) {
	if w.nonInteractive {
		endpoint := os.Getenv("HUSKYCI_CLIENT_API_ADDR")
		if endpoint == "" {
			endpoint = "http://localhost:8888"
			fmt.Printf("Using default endpoint: %s\n", endpoint)
		}
		return endpoint, nil
	}

	w.printSection("Step 1: Configure API Endpoint")
	fmt.Print("Enter your huskyCI API endpoint URL (e.g., https://api.huskyci.example.com or http://localhost:8888): ")

	if !w.scanner.Scan() {
		return "", fmt.Errorf("failed to read input")
	}

	endpoint := strings.TrimSpace(w.scanner.Text())
	if err := validateEndpoint(endpoint); err != nil {
		return "", err
	}

	return endpoint, nil
}

func (w *setupWizard) collectTargetName() (string, error) {
	if w.nonInteractive {
		targetName := "default"
		if len(w.existingTargets) > 0 {
			targetName = fmt.Sprintf("target-%d", len(w.existingTargets)+1)
		}
		fmt.Printf("Using target name: %s\n", targetName)
		return targetName, nil
	}

	w.printSection("Step 2: Configure Target Name")
	fmt.Print("Enter a name for this target (letters, numbers, underscores only, e.g., 'production', 'staging'): ")

	if !w.scanner.Scan() {
		return "", fmt.Errorf("failed to read input")
	}

	targetName := strings.TrimSpace(w.scanner.Text())
	if err := validateTargetName(targetName, w.existingTargets); err != nil {
		return "", err
	}

	return targetName, nil
}

func (w *setupWizard) collectToken() (string, bool) {
	if w.nonInteractive {
		token := config.GetTokenFromEnv()
		return token, token != ""
	}

	fmt.Println()
	fmt.Print("Do you want to configure an authentication token now? (y/n): ")

	if !w.scanner.Scan() {
		return "", false
	}

	response := strings.ToLower(strings.TrimSpace(w.scanner.Text()))
	if response != "y" && response != "yes" {
		return "", false
	}

	fmt.Print("Enter your huskyCI API token: ")

	if !w.scanner.Scan() {
		return "", false
	}

	token := strings.TrimSpace(w.scanner.Text())
	if token == "" {
		w.printWarning("No token provided. You can set it later using:")
		fmt.Println("   export HUSKYCI_CLI_TOKEN=\"your-token\"")
		return "", false
	}

	return token, true
}

// ============================================================================
// Validation
// ============================================================================

func validateEndpoint(endpoint string) error {
	if endpoint == "" {
		return fmt.Errorf("endpoint URL cannot be empty")
	}

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w\n\nPlease enter a valid URL (e.g., https://api.huskyci.example.com)", err)
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("URL must include scheme (http:// or https://) and host\n\nExample: https://api.huskyci.example.com")
	}

	return nil
}

func validateTargetName(targetName string, existingTargets map[string]interface{}) error {
	if targetName == "" {
		return fmt.Errorf("target name cannot be empty")
	}

	match, err := regexp.MatchString(`^\w+$`, targetName)
	if err != nil {
		return fmt.Errorf("error validating target name: %w", err)
	}
	if !match {
		return fmt.Errorf("invalid target name '%s': target name must contain only letters, numbers, and underscores\n\nExample: production, staging, local", targetName)
	}

	for k := range existingTargets {
		if k == targetName {
			return fmt.Errorf("target name '%s' already exists\n\nTip: Use 'huskyci target-remove %s' to remove it first, or choose a different name", targetName, targetName)
		}
	}

	return nil
}

// ============================================================================
// Configuration Management
// ============================================================================

func (w *setupWizard) saveTarget(targetName, endpoint string) error {
	targets := viper.GetStringMap("targets")
	
	// Mark all existing targets as not current
	for _, v := range targets {
		target := v.(map[string]interface{})
		target["current"] = false
	}

	// Add new target as current
	targets[targetName] = map[string]interface{}{
		"current":       true,
		"endpoint":      endpoint,
		"token-storage": "file",
	}

	viper.Set("targets", targets)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("error saving configuration: %w\n\nTip: Check if you have write permissions to the config file", err)
	}

	fmt.Println()
	w.printSuccess(fmt.Sprintf("Successfully added target '%s' -> %s", targetName, endpoint))
	w.printSuccess(fmt.Sprintf("Set '%s' as the current target", targetName))
	
	return nil
}

// ============================================================================
// Token Management
// ============================================================================

func (w *setupWizard) setupToken(token string) {
	fmt.Println()
	fmt.Println("How would you like to set the token?")
	fmt.Println()

	result := w.showMenu("", []menuOption{
		{"1", "Show command to set for current session", func() menuResult {
			w.showTokenCommand(token, false)
			return menuContinue
		}},
		{"2", "Add to shell profile (detected automatically)", func() menuResult {
			w.addTokenToProfile(token)
			return menuContinue
		}},
		{"3", "Just show the command (I'll set it myself)", func() menuResult {
			w.showTokenCommand(token, true)
			return menuContinue
		}},
	})

	if result == menuExit {
		return
	}

	fmt.Println()
	fmt.Println("After setting up your token, you can test the connection using:")
	fmt.Println("  huskyci test-connection")
	fmt.Println()
}

func (w *setupWizard) showTokenCommand(token string, manual bool) {
	_, profileFile, _ := getDetectedShell()
	exportCmd := getShellExportCommand(token, profileFile)

	if manual {
		fmt.Println("To set the token manually, run:")
		fmt.Printf("  %s\n", exportCmd)
	} else {
		fmt.Println("To set the token for your current shell session, run:")
		fmt.Printf("  %s\n", exportCmd)
		fmt.Println()
		fmt.Println("  Note: This will only last for this terminal session")
		fmt.Println("  To make it permanent, choose option 2 to add it to your shell profile")
	}
}

func (w *setupWizard) addTokenToProfile(token string) {
	if err := addTokenToShellProfile(token); err != nil {
		w.printError(fmt.Sprintf("Error adding to shell profile: %v", err))
		_, profileFile, _ := getDetectedShell()
		exportCmd := getShellExportCommand(token, profileFile)
		fmt.Println("  You can manually add this line to your shell profile:")
		fmt.Printf("  %s\n", exportCmd)
		fmt.Printf("  (Add to: %s)\n", profileFile)
	} else {
		_, profileFile, _ := getDetectedShell()
		w.printSuccess("Token added to shell profile")
		fmt.Printf("  Please restart your terminal or run: source %s\n", profileFile)
	}
}

func (w *setupWizard) generateTokenFromAPI(endpoint string) (string, error) {
	fmt.Println("Please provide the following information:")
	fmt.Println()

	// Get username
	fmt.Print("API Username: ")
	if !w.scanner.Scan() {
		return "", fmt.Errorf("failed to read username")
	}
	username := strings.TrimSpace(w.scanner.Text())
	if username == "" {
		return "", fmt.Errorf("username cannot be empty")
	}

	// Get password
	fmt.Print("API Password: ")
	if !w.scanner.Scan() {
		return "", fmt.Errorf("failed to read password")
	}
	password := strings.TrimSpace(w.scanner.Text())
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	// Get repository URL (optional - empty for generic token)
	fmt.Print("Repository URL (e.g., https://github.com/user/repo.git) or press Enter for generic token: ")
	if !w.scanner.Scan() {
		return "", fmt.Errorf("failed to read repository URL")
	}
	repoURL := strings.TrimSpace(w.scanner.Text())
	// Empty URL is now valid - it creates a generic token

	fmt.Println()
	fmt.Println("Generating token...")

	tokenURL := endpoint + "/api/1.0/token"
	payload := map[string]string{
		"repositoryURL": repoURL,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req, err := http.NewRequest("POST", tokenURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	useHTTPS := util.IsHTTPS(endpoint)
	client, err := util.NewHTTPClient(useHTTPS)
	if err != nil {
		return "", fmt.Errorf("error creating HTTP client: %w", err)
	}
	client.Timeout = 30 * time.Second

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error connecting to API: %w\n\nPlease verify:\n  - The API endpoint is correct\n  - The API server is running\n  - Your network connection is working", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// Try to parse error response for better error message
		var errorResponse map[string]interface{}
		if err := json.Unmarshal(body, &errorResponse); err == nil {
			if errorMsg, ok := errorResponse["error"].(string); ok {
				return "", fmt.Errorf("token generation failed (status %d)\n  Error: %s\n\nPlease verify:\n  - Your API credentials are correct\n  - The repository URL is valid\n  - The API server is functioning properly", resp.StatusCode, errorMsg)
			}
		}
		return "", fmt.Errorf("token generation failed (status %d)\n  Response: %s\n\nPlease verify:\n  - Your API credentials are correct\n  - The repository URL is valid\n  - The API server is functioning properly", resp.StatusCode, string(body))
	}

	var token string
	var responseBody map[string]interface{}
	if err := json.Unmarshal(body, &responseBody); err == nil {
		if t, ok := responseBody["huskytoken"].(string); ok {
			token = t
		} else if t, ok := responseBody["token"].(string); ok {
			token = t
		}
	}

	if token == "" {
		token = strings.TrimSpace(string(body))
		token = strings.Trim(token, "\"'")
	}

	if token == "" {
		return "", fmt.Errorf("token generation succeeded but no token found in response\n  Response: %s", string(body))
	}

	w.printSuccess("Token generated successfully!")
	return token, nil
}

// ============================================================================
// Connection Verification
// ============================================================================

func (w *setupWizard) verifyConnection(endpoint, token string, useToken bool) {
	if w.nonInteractive {
		if useToken {
			if err := verifyConnection(endpoint, token); err != nil {
				w.printWarning(fmt.Sprintf("Connection verification failed: %v", err))
			} else {
				w.printSuccess("Connection verified successfully!")
			}
		}
		return
	}

	fmt.Println()
	fmt.Print("Do you want to verify the connection to the API? (y/n): ")

	if !w.scanner.Scan() {
		return
	}

	response := strings.ToLower(strings.TrimSpace(w.scanner.Text()))
	if response != "y" && response != "yes" {
		return
	}

	err := verifyConnection(endpoint, token)
	if err != nil {
		// If connection refused to a local endpoint, offer to start Docker and retry once
		if !w.nonInteractive && util.IsConnectionRefused(err) && util.IsLocalEndpoint(endpoint) && util.PromptAndStartDocker(os.Stdin) {
			fmt.Println("   Retrying connection in 5 seconds...")
			time.Sleep(5 * time.Second)
			err = verifyConnection(endpoint, token)
		}
		if err != nil {
			w.printWarning(fmt.Sprintf("Connection verification failed: %v", err))
			fmt.Println("   You can still use huskyCI CLI, but please verify your endpoint and token.")
		} else {
			w.printSuccess("Connection verified successfully!")
		}
	} else {
		w.printSuccess("Connection verified successfully!")
	}
}

// ============================================================================
// Summary
// ============================================================================

func (w *setupWizard) printSummary(endpoint string, useToken bool) {
	fmt.Println()
	fmt.Println(separatorLine)
	fmt.Println("  ✓ Setup Complete!")
	fmt.Println(separatorLine)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println()

	if !useToken {
		fmt.Println("  1. Set your authentication token:")
		fmt.Println("     export HUSKYCI_CLI_TOKEN=\"your-token\"")
		fmt.Println()
		fmt.Println("     Or generate a token via the API:")
		fmt.Printf("     curl -X POST %s/api/1.0/token \\\n", endpoint)
		fmt.Println("       -u username:password \\")
		fmt.Println("       -H \"Content-Type: application/json\" \\")
		fmt.Println("       -d '{\"repositoryURL\": \"https://github.com/user/repo.git\"}'")
		fmt.Println()
	}

	fmt.Println("  2. Run your first security analysis:")
	fmt.Println("     huskyci run ./my-project")
	fmt.Println()
	fmt.Println("  3. List all configured targets:")
	fmt.Println("     huskyci target-list")
	fmt.Println()
	fmt.Println("For more information, visit: https://github.com/huskyci-org/huskyCI")
	fmt.Println()
}

// ============================================================================
// Helper Functions
// ============================================================================

func verifyConnection(endpoint string, token string) error {
	client, err := createHTTPClient(endpoint)
	if err != nil {
		return err
	}

	baseURL := normalizeURL(endpoint)
	if err := tryConnection(client, baseURL, token, endpoint); err == nil {
		return nil
	}

	rootURL := baseURL + "/"
	return tryConnection(client, rootURL, token, endpoint)
}

func createHTTPClient(endpoint string) (*http.Client, error) {
	useHTTPS := util.IsHTTPS(endpoint)
	client, err := util.NewHTTPClient(useHTTPS)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}
	client.Timeout = 10 * time.Second
	return client, nil
}

func normalizeURL(url string) string {
	if strings.HasSuffix(url, "/") {
		return url[:len(url)-1]
	}
	return url
}

func tryConnection(client *http.Client, url, token, endpoint string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if token != "" {
		req.Header.Set("Husky-Token", token)
	}
	req.Header.Set("User-Agent", "huskyci-cli")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("unable to connect to %s: %w\n\nPlease verify:\n  - The API endpoint is correct\n  - The API server is running\n  - Your network connection is working", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return fmt.Errorf("endpoint is reachable but authentication failed (status %d)\n\nTip: Verify your token is correct. You can still use huskyCI CLI, but authentication may be required for actual operations", resp.StatusCode)
		}
		return nil
	}

	if resp.StatusCode >= 500 {
		return fmt.Errorf("endpoint is reachable but server returned error (status %d)\n\nTip: The API server may be experiencing issues. You can still use huskyCI CLI, but operations may fail", resp.StatusCode)
	}

	return nil
}

func getDetectedShell() (shellName string, profileFile string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	shell = strings.ToLower(shell)
	shellBase := filepath.Base(shell)

	switch {
	case strings.Contains(shellBase, "fish"):
		configDir := home + "/.config/fish"
		profileFile = configDir + "/config.fish"
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return "", "", fmt.Errorf("failed to create fish config directory: %w", err)
		}
		return "fish", profileFile, nil
	case strings.Contains(shellBase, "zsh"):
		profileFile = home + "/.zshrc"
		return "zsh", profileFile, nil
	case strings.Contains(shellBase, "bash"):
		profileFile = home + "/.bashrc"
		if _, err := os.Stat(profileFile); os.IsNotExist(err) {
			profileFile = home + "/.bash_profile"
		}
		return "bash", profileFile, nil
	case strings.Contains(shellBase, "csh") || strings.Contains(shellBase, "tcsh"):
		profileFile = home + "/.cshrc"
		if _, err := os.Stat(profileFile); os.IsNotExist(err) {
			profileFile = home + "/.tcshrc"
		}
		return "csh", profileFile, nil
	default:
		profileFile = home + "/.bashrc"
		if _, err := os.Stat(profileFile); os.IsNotExist(err) {
			profileFile = home + "/.bash_profile"
		}
		return "bash", profileFile, nil
	}
}

func getShellExportCommand(token string, profileFile string) string {
	if strings.Contains(profileFile, "fish") {
		return fmt.Sprintf("set -x HUSKYCI_CLI_TOKEN \"%s\"", token)
	}
	return fmt.Sprintf("export HUSKYCI_CLI_TOKEN=\"%s\"", token)
}

func addTokenToShellProfile(token string) error {
	_, profileFile, err := getDetectedShell()
	if err != nil {
		return err
	}

	content, err := os.ReadFile(profileFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	exportLine := getShellExportCommand(token, profileFile)
	hasCLIToken := strings.Contains(string(content), "HUSKYCI_CLI_TOKEN")

	if hasCLIToken {
		lines := strings.Split(string(content), "\n")
		var newLines []string
		replaced := false
		for _, line := range lines {
			if strings.Contains(line, "HUSKYCI_CLI_TOKEN") {
				if !replaced {
					newLines = append(newLines, exportLine)
					replaced = true
				}
			} else {
				newLines = append(newLines, line)
			}
		}
		content = []byte(strings.Join(newLines, "\n"))
	} else {
		if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
			content = append(content, '\n')
		}
		content = append(content, []byte(fmt.Sprintf("\n# huskyCI CLI Token\n%s\n", exportLine))...)
	}

	return os.WriteFile(profileFile, content, 0644)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
