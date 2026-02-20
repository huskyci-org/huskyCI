package util

import (
	"bufio"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"errors"
	"fmt"

	"github.com/huskyci-org/huskyCI/api/log"
	"github.com/huskyci-org/huskyCI/api/types"
	"github.com/labstack/echo/v4"
)

const (
	// CertFile contains the address for the API's TLS certificate.
	CertFile = "api/api-tls-cert.pem"
	// KeyFile contains the address for the API's TLS certificate key file.
	KeyFile = "api/api-tls-key.pem"
)

const (
	logInfoAnalysis        = "ANALYSIS"
	logActionReceiveRequest = "ReceiveRequest"
	errInternalError       = "internal error"
)

// HandleCmd will extract %GIT_REPO%, %GIT_BRANCH% from cmd and replace it with the proper repository URL.
// For file:// URLs, it replaces git clone commands with commands to use the mounted volume at /workspace.
func HandleCmd(repositoryURL, repositoryBranch, cmd string) string {
	if repositoryURL != "" && repositoryBranch != "" && cmd != "" {
		// Check if this is a file:// URL (local repository)
		if IsFileURL(repositoryURL) {
			// Replace git clone commands with commands to copy from mounted volume
			// The volume is mounted at /workspace in the container
			// Handle various git clone patterns that may have prefixes/suffixes
			
			// Pattern 1: git clone -b %GIT_BRANCH% --single-branch %GIT_REPO% code (with optional prefix/suffix)
			// Match the entire line containing this pattern (handles GIT_TERMINAL_PROMPT=0 prefix, --quiet suffix, etc.)
			// Use cp -r /workspace/. code to copy contents (not the directory itself), or cp -r /workspace/* code
			re1 := regexp.MustCompile(`(?m)^[^\n]*git clone -b %GIT_BRANCH% --single-branch %GIT_REPO% code[^\n]*$`)
			if re1.MatchString(cmd) {
				// Copy contents of /workspace into code directory
				cmd = re1.ReplaceAllString(cmd, "mkdir -p code && cp -r /workspace/. code/ 2>/dev/null || cp -r /workspace/* code/")
			}
			
			// Pattern 2: git clone %GIT_REPO% code (with optional prefix/suffix)
			re2 := regexp.MustCompile(`(?m)^[^\n]*git clone %GIT_REPO% code[^\n]*$`)
			if re2.MatchString(cmd) && !strings.Contains(cmd, "cp -r /workspace") {
				cmd = re2.ReplaceAllString(cmd, "mkdir -p code && cp -r /workspace/. code/ 2>/dev/null || cp -r /workspace/* code/")
			}
			
			// Pattern 3: Fallback - any git clone with %GIT_REPO% that wasn't caught above
			if strings.Contains(cmd, "git clone") && strings.Contains(cmd, "%GIT_REPO%") && !strings.Contains(cmd, "cp -r /workspace") {
				// Match any line containing git clone with %GIT_REPO% and code
				re3 := regexp.MustCompile(`(?m)^[^\n]*git clone[^\n]*%GIT_REPO%[^\n]*code[^\n]*$`)
				cmd = re3.ReplaceAllString(cmd, "mkdir -p code && cp -r /workspace/. code/ 2>/dev/null || cp -r /workspace/* code/")
			}
			
			// Remove remaining placeholders since we're using extracted files
			cmd = strings.Replace(cmd, "%GIT_BRANCH%", repositoryBranch, -1)
			cmd = strings.Replace(cmd, "%GIT_REPO%", repositoryURL, -1)
			return cmd
		}
		// Standard git repository handling
		replace1 := strings.Replace(cmd, "%GIT_REPO%", repositoryURL, -1)
		replace2 := strings.Replace(replace1, "%GIT_BRANCH%", repositoryBranch, -1)
		return replace2
	}
	return ""
}

// HandleGitURLSubstitution will extract GIT_SSH_URL and GIT_URL_TO_SUBSTITUTE from cmd and replace it with the SSH equivalent.
func HandleGitURLSubstitution(rawString string) string {
	gitSSHURL := os.Getenv("HUSKYCI_API_GIT_SSH_URL")
	gitURLToSubstitute := os.Getenv("HUSKYCI_API_GIT_URL_TO_SUBSTITUTE")

	if gitSSHURL == "" || gitURLToSubstitute == "" {
		gitSSHURL = "nil"
		gitURLToSubstitute = "nil"
	}
	cmdReplaced := strings.Replace(rawString, "%GIT_SSH_URL%", gitSSHURL, -1)
	cmdReplaced = strings.Replace(cmdReplaced, "%GIT_URL_TO_SUBSTITUTE%", gitURLToSubstitute, -1)

	return cmdReplaced
}

// HandlePrivateSSHKey will extract %GIT_PRIVATE_SSH_KEY% from cmd and replace it with the proper private SSH key.
func HandlePrivateSSHKey(rawString string) string {
	privKey := os.Getenv("HUSKYCI_API_GIT_PRIVATE_SSH_KEY")
	cmdReplaced := strings.Replace(rawString, "%GIT_PRIVATE_SSH_KEY%", privKey, -1)
	return cmdReplaced
}

// GetLastLine receives a string with multiple lines and returns it's last
func GetLastLine(s string) string {
	if s == "" {
		return ""
	}
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines[len(lines)-1]
}

// GetAllLinesButLast receives a string with multiple lines and returns all but the last line.
func GetAllLinesButLast(s string) []string {
	if s == "" {
		return []string{}
	}
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	lines = lines[:len(lines)-1]
	return lines
}

// SanitizeSafetyJSON returns a sanitized string from Safety container logs.
// Safety might return a JSON with the "\" and "\"" characters, which needs to be sanitized to be unmarshalled correctly.
func SanitizeSafetyJSON(s string) string {
	if s == "" {
		return ""
	}
	s1 := strings.Replace(s, "\\", "\\\\", -1)
	s2 := strings.Replace(s1, "\\\"", "\\\\\"", -1)
	return s2
}

// RemoveDuplicates remove duplicated itens from a slice.
func RemoveDuplicates(s []string) []string {
	mapS := make(map[string]string, len(s))
	i := 0
	for _, v := range s {
		if _, ok := mapS[v]; !ok {
			mapS[v] = v
			s[i] = v
			i++
		}
	}
	return s[:i]
}

// HandleScanError show the right error when json is not expected as output of scan
func HandleScanError(containerOutput string, otherErr error) error {
	if otherErr != nil && (strings.Contains(otherErr.Error(), "unexpected end of JSON input") || strings.Contains(otherErr.Error(), "EOF")) {
		trimmed := strings.TrimSpace(containerOutput)
		if trimmed == "" {
			return fmt.Errorf("security tool produced no valid JSON output (empty or truncated). This may mean the tool had no code to analyze (e.g. zip extraction in dockerapi failed or workspace was empty): %w", otherErr)
		}
	}
	return fmt.Errorf("%s\nerror from top: %v", containerOutput, otherErr)
}

// CheckValidInput checks if an user's input is "malicious" or not
func CheckValidInput(repository types.Repository, c echo.Context) (string, error) {

	sanitiziedURL, err := CheckMaliciousRepoURL(repository.URL)
	if err != nil {
		if sanitiziedURL == "" {
			log.Error(logActionReceiveRequest, logInfoAnalysis, 1016, repository.URL)
			reply := map[string]interface{}{
				"success": false,
				"error":   "invalid repository URL",
				"message": fmt.Sprintf("The repository URL '%s' is not in a valid format. Please provide a valid Git repository URL (e.g., https://github.com/user/repo.git or git@github.com:user/repo.git)", repository.URL),
			}
			return "", c.JSON(http.StatusBadRequest, reply)
		}
		log.Error(logActionReceiveRequest, logInfoAnalysis, 1008, "Repository URL regexp ", err)
		reply := map[string]interface{}{
			"success": false,
			"error":   "internal server error",
			"message": "An error occurred while validating the repository URL. Please try again.",
		}
		return "", c.JSON(http.StatusInternalServerError, reply)
	}

	if err := CheckMaliciousRepoBranch(repository.Branch, c); err != nil {
		return "", err
	}

	return sanitiziedURL, nil
}

// CheckMaliciousRepoURL verifies if a given URL is a git repository and returns the sanitizied string and its error
// It accepts both git repository URLs (ending in .git) and file:// URLs for local analysis
func CheckMaliciousRepoURL(repositoryURL string) (string, error) {
	// Check for file:// URLs (for local file analysis)
	regexpFile := `file://[a-zA-Z0-9\-_/\.]+`
	rFile := regexp.MustCompile(regexpFile)
	if rFile.MatchString(repositoryURL) {
		return rFile.FindString(repositoryURL), nil
	}
	
	// Check for git repository URLs (must end in .git)
	regexpGit := `((git|ssh|http(s)?)|((git@|gitlab@)[\w\.]+))(:(//)?)([\w\.@\:/\-~]+)(\.git)(/)?`
	r := regexp.MustCompile(regexpGit)
	valid, err := regexp.MatchString(regexpGit, repositoryURL)
	if err != nil {
		return "matchStringError", err
	}
	if !valid {
		errorMsg := fmt.Sprintf("Invalid URL format: %s", repositoryURL)
		return "", errors.New(errorMsg)
	}
	return r.FindString(repositoryURL), nil
}

// CheckMaliciousRepoBranch verifies if a given branch is "malicious" or not
func CheckMaliciousRepoBranch(repositoryBranch string, c echo.Context) error {
	regexpBranch := `^[a-zA-Z0-9_\/.\-\+À-ÿ]*$`
	valid, err := regexp.MatchString(regexpBranch, repositoryBranch)
	if err != nil {
		log.Error(logActionReceiveRequest, logInfoAnalysis, 1008, "Repository Branch regexp ", err)
		reply := map[string]interface{}{"success": false, "error": errInternalError}
		return c.JSON(http.StatusInternalServerError, reply)
	}
	if !valid {
		log.Error(logActionReceiveRequest, logInfoAnalysis, 1017, repositoryBranch)
		reply := map[string]interface{}{
			"success": false,
			"error":   "invalid repository branch",
			"message": fmt.Sprintf("The branch name '%s' contains invalid characters. Branch names can only contain letters, numbers, underscores, forward slashes, dots, hyphens, plus signs, and accented characters.", repositoryBranch),
		}
		return c.JSON(http.StatusBadRequest, reply)
	}
	return nil
}

// CheckMaliciousRID verifies if a given RID is "malicious" or not
func CheckMaliciousRID(RID string, c echo.Context) error {
	regexpRID := `^[-a-zA-Z0-9]*$`
	valid, err := regexp.MatchString(regexpRID, RID)
	if err != nil {
		log.Error("GetAnalysis", logInfoAnalysis, 1008, "RID regexp ", err)
		reply := map[string]interface{}{"success": false, "error": errInternalError}
		return c.JSON(http.StatusInternalServerError, reply)
	}
	if !valid {
		log.Warning("GetAnalysis", logInfoAnalysis, 107, RID)
		reply := map[string]interface{}{
			"success": false,
			"error":   "invalid RID format",
			"message": fmt.Sprintf("The RID '%s' contains invalid characters. RID must only contain letters, numbers, hyphens, and underscores.", RID),
		}
		return c.JSON(http.StatusBadRequest, reply)
	}
	return nil
}

// AdjustWarningMessage returns the Safety Warning string that will be printed.
func AdjustWarningMessage(warningRaw string) string {
	warning := strings.Split(warningRaw, ":")
	if len(warning) > 1 {
		warning[1] = strings.Replace(warning[1], "safety_huskyci_analysis_requirements_raw.txt", "'requirements.txt'", -1)
		warning[1] = strings.Replace(warning[1], " unpinned", "Unpinned", -1)

		return (warning[1] + " huskyCI can check it if you pin it in a format such as this: \"mypacket==3.2.9\" :D")
	}

	return warningRaw
}

// EndOfTheDay returns the the time at the end of the day t.
func EndOfTheDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 23, 59, 59, 0, t.Location())
}

// BeginningOfTheDay returns the the time at the beginning of the day t.
func BeginningOfTheDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

// CountDigits returns the number of digits in an integer.
func CountDigits(i int) int {
	count := 0
	for i != 0 {
		i /= 10
		count = count + 1
	}

	return count
}

func banditCase(code string, lineNumber int) bool {
	lineNumberLength := CountDigits(lineNumber)
	splitCode := strings.Split(code, "\n")
	for _, codeLine := range splitCode {
		if len(codeLine) > 0 {
			codeLineNumber := codeLine[:lineNumberLength]
			if strings.Contains(codeLine, "#nohusky") && (codeLineNumber == strconv.Itoa(lineNumber)) {
				return true
			}
		}
	}
	return false
}

// VerifyNoHusky verifies if the code string is marked with the #nohusky tag.
func VerifyNoHusky(code string, lineNumber int, securityTool string) bool {
	m := map[string]types.NohuskyFunction{
		"Bandit": banditCase,
	}

	return m[securityTool](code, lineNumber)

}

// SliceContains returns true if a given value is present on the given slice
func SliceContains(slice []string, str string) bool {
	for _, value := range slice {
		if value == str {
			return true
		}
	}
	return false
}

// GetTokenFromRequest retrieves the authentication token from the request.
// It first checks the "Husky-Token" header. If the header is empty,
// it checks environment variables based on the request source:
// - HUSKYCI_CLI_TOKEN for CLI requests (detected via User-Agent containing "huskyci-cli")
// - HUSKYCI_CLIENT_TOKEN for client requests (detected via User-Agent containing "huskyci-client")
// Returns empty string if no token is found.
func GetTokenFromRequest(c echo.Context) string {
	// First, check the Husky-Token header
	token := c.Request().Header.Get("Husky-Token")
	if token != "" {
		return token
	}

	// If header is empty, check User-Agent to determine source
	userAgent := c.Request().Header.Get("User-Agent")
	
	// Check if it's a CLI request
	if strings.Contains(strings.ToLower(userAgent), "huskyci-cli") {
		if cliToken := os.Getenv("HUSKYCI_CLI_TOKEN"); cliToken != "" {
			return cliToken
		}
	}
	
	// Check if it's a client request
	if strings.Contains(strings.ToLower(userAgent), "huskyci-client") {
		if clientToken := os.Getenv("HUSKYCI_CLIENT_TOKEN"); clientToken != "" {
			return clientToken
		}
	}
	
	// Fallback: if User-Agent is not set or doesn't match, try both environment variables
	// CLI token takes precedence
	if cliToken := os.Getenv("HUSKYCI_CLI_TOKEN"); cliToken != "" {
		return cliToken
	}
	if clientToken := os.Getenv("HUSKYCI_CLIENT_TOKEN"); clientToken != "" {
		return clientToken
	}
	
	return ""
}
