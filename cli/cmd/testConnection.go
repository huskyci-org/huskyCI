package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/huskyci-org/huskyCI/cli/config"
	"github.com/huskyci-org/huskyCI/cli/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// testConnectionCmd represents the test-connection command
var testConnectionCmd = &cobra.Command{
	Use:   "test-connection [target-name]",
	Short: "Test connection to huskyCI API server",
	Long: `Test connectivity and verify configuration for huskyCI API server.

This command performs several connection tests:
  1. Basic connectivity to the API endpoint
  2. Health check endpoint (/healthcheck)
  3. Version endpoint (/version)
  4. Authentication (if token is configured)

You can test:
  - Current target (default)
  - Specific target by name
  - Custom endpoint via --endpoint flag

Examples:
  # Test current target
  huskyci test-connection

  # Test specific target
  huskyci test-connection production

  # Test custom endpoint
  huskyci test-connection --endpoint https://api.huskyci.example.com

  # Test with verbose output
  huskyci test-connection --verbose`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		endpoint, _ := cmd.Flags().GetString("endpoint")
		skipAuth, _ := cmd.Flags().GetBool("skip-auth")
		
		var targetName string
		if len(args) > 0 {
			targetName = args[0]
		}

		return runConnectionTest(endpoint, targetName, skipAuth)
	},
}

func init() {
	rootCmd.AddCommand(testConnectionCmd)
	testConnectionCmd.Flags().String("endpoint", "", "Test a specific endpoint URL (overrides target selection)")
	testConnectionCmd.Flags().Bool("skip-auth", false, "Skip authentication tests")
}

// ConnectionTestResult holds the results of connection tests
type ConnectionTestResult struct {
	TestName      string
	Success       bool
	Status        string
	StatusCode    int
	ResponseTime  time.Duration
	ErrorMessage  string
	ResponseBody  string
}

// runConnectionTest executes connection tests
func runConnectionTest(customEndpoint, targetName string, skipAuth bool) error {
	var endpoint string
	var token string
	var label string

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  🔌 huskyCI API Connection Test")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Determine endpoint and token
	if customEndpoint != "" {
		endpoint = customEndpoint
		label = "custom endpoint"
		token = config.GetTokenFromEnv() // Check environment variable for token
		if IsVerbose() {
			fmt.Printf("[VERBOSE] Using custom endpoint: %s\n", endpoint)
		}
	} else {
		target, err := getTargetForTest(targetName)
		if err != nil {
			return err
		}
		endpoint = target.Endpoint
		token = target.Token
		label = target.Label
		fmt.Printf("Testing target: %s\n", label)
		fmt.Printf("Endpoint: %s\n", endpoint)
		fmt.Println()
	}

	// Run tests
	results := []ConnectionTestResult{}

	// Test 1: Basic connectivity
	fmt.Println("Test 1: Basic Connectivity")
	fmt.Println("──────────────────────────────────────────────────────────────────────────────")
	result := testBasicConnectivity(endpoint)
	results = append(results, result)
	printTestResult(result)
	fmt.Println()

	if !result.Success {
		fmt.Println("⚠️  Basic connectivity failed. Skipping remaining tests.")
		printTestSummary(results)
		return fmt.Errorf("connection test failed: %s", result.ErrorMessage)
	}

	// Test 2: Health check
	fmt.Println("Test 2: Health Check Endpoint")
	fmt.Println("──────────────────────────────────────────────────────────────────────────────")
	result = testHealthCheck(endpoint)
	results = append(results, result)
	printTestResult(result)
	fmt.Println()

	// Test 3: Version endpoint
	fmt.Println("Test 3: Version Endpoint")
	fmt.Println("──────────────────────────────────────────────────────────────────────────────")
	result = testVersionEndpoint(endpoint)
	results = append(results, result)
	printTestResult(result)
	fmt.Println()

	// Test 4: Authentication (if token available and not skipped)
	if !skipAuth && token != "" {
		fmt.Println("Test 4: Authentication")
		fmt.Println("──────────────────────────────────────────────────────────────────────────────")
		result = testAuthentication(endpoint, token)
		results = append(results, result)
		printTestResult(result)
		fmt.Println()
	} else if !skipAuth && token == "" {
		fmt.Println("Test 4: Authentication")
		fmt.Println("──────────────────────────────────────────────────────────────────────────────")
		fmt.Println("⚠️  Skipped: No authentication token configured")
		fmt.Println("   Tip: Set HUSKYCI_CLI_TOKEN or configure token storage")
		fmt.Println()
	}

	// Print summary
	printTestSummary(results)

	// Return error if any critical test failed
	for _, r := range results {
		if !r.Success && r.TestName != "Authentication" {
			return fmt.Errorf("connection test failed: %s", r.ErrorMessage)
		}
	}

	return nil
}

// getTargetForTest retrieves the target to test
func getTargetForTest(targetName string) (*types.Target, error) {
	if targetName != "" {
		// Get specific target from config
		targets := viper.GetStringMap("targets")
		if target, exists := targets[targetName]; exists {
			targetMap := target.(map[string]interface{})
			// Get token from environment if available
			token := config.GetTokenFromEnv()
			return &types.Target{
				Label:    targetName,
				Endpoint: targetMap["endpoint"].(string),
				Token:    token,
			}, nil
		}
		return nil, fmt.Errorf("target '%s' not found\n\nTip: Use 'huskyci target-list' to see available targets", targetName)
	}

	// Get current target
	target, err := config.GetCurrentTarget()
	if err != nil {
		return nil, fmt.Errorf("failed to get target: %w\n\nTip: Configure a target using 'huskyci target-add <name> <endpoint>' or 'huskyci setup'", err)
	}

	return target, nil
}

// testBasicConnectivity tests basic connectivity to the endpoint
// A 404 or other 4xx response indicates the server is reachable (just the path doesn't exist)
// Only 5xx errors or connection failures indicate actual connectivity issues
func testBasicConnectivity(endpoint string) ConnectionTestResult {
	start := time.Now()
	result := ConnectionTestResult{
		TestName: "Basic Connectivity",
	}

	client, err := createHTTPClient(endpoint)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to create HTTP client: %v", err)
		return result
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to create request: %v", err)
		return result
	}

	resp, err := client.Do(req)
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Connection failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Status = resp.Status

	// Any HTTP response (including 404) means the server is reachable
	// 2xx/3xx = success, 4xx = endpoint exists but path not found (still success for connectivity)
	// 5xx = server error (still reachable, but server has issues)
	if resp.StatusCode < 500 {
		result.Success = true
		if resp.StatusCode == http.StatusNotFound {
			result.Status = "Endpoint is reachable (404 - path not found, but server is responding)"
		} else if resp.StatusCode < 400 {
			result.Status = "Endpoint is reachable"
		} else {
			result.Status = fmt.Sprintf("Endpoint is reachable (status %d)", resp.StatusCode)
		}
	} else {
		// 5xx errors indicate server issues, but connectivity is still successful
		result.Success = true
		result.Status = fmt.Sprintf("Endpoint is reachable but server returned error (status %d)", resp.StatusCode)
		result.ErrorMessage = fmt.Sprintf("Server error: status %d", resp.StatusCode)
	}

	return result
}

// testHealthCheck tests the /healthcheck endpoint
func testHealthCheck(endpoint string) ConnectionTestResult {
	start := time.Now()
	result := ConnectionTestResult{
		TestName: "Health Check",
	}

	client, err := createHTTPClient(endpoint)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to create HTTP client: %v", err)
		return result
	}

	healthURL := normalizeURL(endpoint) + "/healthcheck"
	req, err := http.NewRequest("GET", healthURL, nil)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to create request: %v", err)
		return result
	}

	resp, err := client.Do(req)
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := strings.TrimSpace(string(body))
	result.StatusCode = resp.StatusCode
	result.Status = resp.Status
	result.ResponseBody = bodyStr

	if resp.StatusCode == http.StatusOK && bodyStr == "WORKING" {
		result.Success = true
		result.Status = "Health check passed"
	} else if resp.StatusCode == http.StatusOK {
		result.Success = true
		result.Status = fmt.Sprintf("Health check responded (unexpected body: %s)", bodyStr)
	} else {
		result.ErrorMessage = fmt.Sprintf("Health check failed: status %d, body: %s", resp.StatusCode, bodyStr)
	}

	return result
}

// testVersionEndpoint tests the /version endpoint
func testVersionEndpoint(endpoint string) ConnectionTestResult {
	start := time.Now()
	result := ConnectionTestResult{
		TestName: "Version Endpoint",
	}

	client, err := createHTTPClient(endpoint)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to create HTTP client: %v", err)
		return result
	}

	versionURL := normalizeURL(endpoint) + "/version"
	req, err := http.NewRequest("GET", versionURL, nil)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to create request: %v", err)
		return result
	}

	resp, err := client.Do(req)
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := strings.TrimSpace(string(body))
	result.StatusCode = resp.StatusCode
	result.Status = resp.Status
	result.ResponseBody = bodyStr

	if resp.StatusCode == http.StatusOK {
		result.Success = true
		result.Status = fmt.Sprintf("API Version: %s", bodyStr)
	} else {
		result.ErrorMessage = fmt.Sprintf("Version endpoint failed: status %d, body: %s", resp.StatusCode, bodyStr)
	}

	return result
}

// testAuthentication tests authentication with the provided token
func testAuthentication(endpoint, token string) ConnectionTestResult {
	start := time.Now()
	result := ConnectionTestResult{
		TestName: "Authentication",
	}

	client, err := createHTTPClient(endpoint)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to create HTTP client: %v", err)
		return result
	}

	// Use POST /analysis with valid JSON but an invalid repository URL
	// The endpoint checks authentication AFTER JSON validation but BEFORE URL validation
	// This allows us to test auth without triggering an actual analysis
	// Flow: JSON validation -> Auth check -> URL validation
	// - If auth fails: 401 Unauthorized
	// - If auth passes but URL invalid: 400 Bad Request (auth test passed!)
	testURL := normalizeURL(endpoint) + "/analysis"
	
	// Create a minimal valid JSON payload with an invalid repository URL
	// This will pass JSON validation but fail URL validation after auth check
	payload := map[string]string{
		"repositoryURL":      "https://test-auth-validation.invalid/repo.git",
		"repositoryBranch":   "main",
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to create request payload: %v", err)
		return result
	}

	req, err := http.NewRequest("POST", testURL, bytes.NewBuffer(jsonData))
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to create request: %v", err)
		return result
	}

	req.Header.Set("Husky-Token", token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "huskyci-cli")

	resp, err := client.Do(req)
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := strings.TrimSpace(string(body))
	result.StatusCode = resp.StatusCode
	result.Status = resp.Status
	result.ResponseBody = bodyStr

	// Parse response to determine authentication status
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		// Token is invalid or doesn't have permission
		// The server checked auth and rejected it
		result.ErrorMessage = fmt.Sprintf("Authentication failed: token is invalid or expired (status %d)", resp.StatusCode)
		if bodyStr != "" {
			// Try to extract error message from JSON response
			var errorResp map[string]interface{}
			if err := json.Unmarshal(body, &errorResp); err == nil {
				if msg, ok := errorResp["message"].(string); ok {
					result.ErrorMessage = fmt.Sprintf("Authentication failed: %s", msg)
				} else if errMsg, ok := errorResp["error"].(string); ok {
					result.ErrorMessage = fmt.Sprintf("Authentication failed: %s", errMsg)
				}
			}
		}
	case http.StatusBadRequest:
		// Bad request could mean:
		// 1. JSON validation failed (unlikely with our valid JSON)
		// 2. URL validation failed (happens AFTER auth check passes)
		// Since we're sending valid JSON, a 400 typically means auth passed but URL is invalid
		// Check the error message to be more precise
		var errorResp map[string]interface{}
		if bodyStr != "" {
			if err := json.Unmarshal(body, &errorResp); err == nil {
				if msg, ok := errorResp["message"].(string); ok {
					// If error mentions JSON format, it's a JSON validation error (unlikely)
					// Otherwise, it's likely URL validation (which happens after auth)
					if strings.Contains(strings.ToLower(msg), "json") || strings.Contains(strings.ToLower(msg), "format") {
						result.ErrorMessage = fmt.Sprintf("Unexpected JSON validation error: %s", msg)
					} else {
						// URL validation error means auth passed!
						result.Success = true
						result.Status = "Authentication successful (token is valid)"
						if IsVerbose() {
							result.Status += fmt.Sprintf(" - URL validation failed as expected: %s", msg)
						}
					}
				} else {
					// No message field, assume auth passed (URL validation failed)
					result.Success = true
					result.Status = "Authentication successful (token is valid)"
				}
			} else {
				// Couldn't parse JSON, but 400 with our valid JSON likely means auth passed
				result.Success = true
				result.Status = "Authentication successful (token is valid)"
			}
		} else {
			// Empty body with 400, assume auth passed
			result.Success = true
			result.Status = "Authentication successful (token is valid)"
		}
	case http.StatusOK:
		// Request succeeded completely - authentication successful
		result.Success = true
		result.Status = "Authentication successful (token is valid)"
	default:
		// Other status codes
		if resp.StatusCode < 400 {
			result.Success = true
			result.Status = fmt.Sprintf("Authentication successful (status %d)", resp.StatusCode)
		} else if resp.StatusCode >= 500 {
			result.ErrorMessage = fmt.Sprintf("Server error during authentication test (status %d)", resp.StatusCode)
			if bodyStr != "" {
				result.ErrorMessage += fmt.Sprintf(" - %s", bodyStr)
			}
		} else {
			// Other 4xx errors
			result.ErrorMessage = fmt.Sprintf("Unexpected response: status %d", resp.StatusCode)
			if bodyStr != "" {
				result.ErrorMessage += fmt.Sprintf(" - %s", bodyStr)
			}
		}
	}

	return result
}

// printTestResult prints the result of a single test
func printTestResult(result ConnectionTestResult) {
	if result.Success {
		fmt.Printf("✓ %s\n", result.Status)
	} else {
		fmt.Printf("✗ %s\n", result.Status)
		if result.ErrorMessage != "" {
			fmt.Printf("  Error: %s\n", result.ErrorMessage)
		}
	}

	if IsVerbose() {
		fmt.Printf("  Status Code: %d\n", result.StatusCode)
		fmt.Printf("  Response Time: %v\n", result.ResponseTime)
		if result.ResponseBody != "" {
			fmt.Printf("  Response Body: %s\n", result.ResponseBody)
		}
	} else {
		fmt.Printf("  Response Time: %v\n", result.ResponseTime)
	}
}

// printTestSummary prints a summary of all test results
func printTestSummary(results []ConnectionTestResult) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  📊 Test Summary")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	passed := 0
	failed := 0
	skipped := 0

	for _, result := range results {
		if result.Success {
			passed++
			fmt.Printf("✓ %s\n", result.TestName)
		} else if result.ErrorMessage != "" {
			failed++
			fmt.Printf("✗ %s: %s\n", result.TestName, result.ErrorMessage)
		} else {
			skipped++
			fmt.Printf("⊘ %s (skipped)\n", result.TestName)
		}
	}

	fmt.Println()
	fmt.Printf("Total: %d passed, %d failed", passed, failed)
	if skipped > 0 {
		fmt.Printf(", %d skipped", skipped)
	}
	fmt.Println()

	if failed == 0 {
		fmt.Println()
		fmt.Println("✅ All connection tests passed!")
		fmt.Println("   Your huskyCI CLI is properly configured and ready to use.")
	} else {
		fmt.Println()
		fmt.Println("⚠️  Some tests failed. Please check:")
		fmt.Println("   • Network connectivity")
		fmt.Println("   • API endpoint URL is correct")
		fmt.Println("   • API server is running")
		fmt.Println("   • Authentication token is valid (if required)")
		fmt.Println()
		fmt.Println("   Use 'huskyci setup' to reconfigure or 'huskyci target-list' to check targets.")
	}
	fmt.Println()
}
