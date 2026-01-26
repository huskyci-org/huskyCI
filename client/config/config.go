package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// RepositoryURL stores the repository URL of the project to be analyzed.
var RepositoryURL string

// HuskyAPI stores the address of Husky's API.
var HuskyAPI string

// RepositoryBranch stores the repository branch of the project to be analyzed.
var RepositoryBranch string

// HuskyToken is the token used to scan a repository.
var HuskyToken string

// LanguageExclusions stores a map of languages to exclude from analysis.
var LanguageExclusions map[string]bool

// HuskyUseTLS stores if huskyCI is to use an HTTPS connection.
var HuskyUseTLS bool

// SetConfigs sets all configuration needed to start the client.
func SetConfigs() {
	RepositoryURL = os.Getenv(`HUSKYCI_CLIENT_REPO_URL`)
	RepositoryBranch = os.Getenv(`HUSKYCI_CLIENT_REPO_BRANCH`)
	HuskyAPI = os.Getenv(`HUSKYCI_CLIENT_API_ADDR`)
	exclusionsEnv := os.Getenv(`HUSKYCI_LANGUAGE_EXCLUSIONS`)
	if exclusionsEnv != "" {
		languagesToExclude := strings.Split(exclusionsEnv, ",")
		LanguageExclusions = make(map[string]bool)
		for _, lang := range languagesToExclude {
			LanguageExclusions[lang] = true
		}
	}
	HuskyToken = os.Getenv(`HUSKYCI_CLIENT_TOKEN`)
	HuskyUseTLS = getUseTLS()
}

// CheckEnvVars checks if all environment vars are set.
func CheckEnvVars() error {

	envVars := []string{
		"HUSKYCI_CLIENT_API_ADDR",
		"HUSKYCI_CLIENT_REPO_URL",
		"HUSKYCI_CLIENT_REPO_BRANCH",
		// "HUSKYCI_CLIENT_TOKEN", (optional for now)
		// "HUSKYCI_CLIENT_API_USE_HTTPS", (optional)
		// "HUSKYCI_CLIENT_NPM_DEP_URL", (optional)
	}

	var envIsSet bool
	var allEnvIsSet bool
	var missingVars []string

	env := make(map[string]string)
	allEnvIsSet = true
	for i := 0; i < len(envVars); i++ {
		env[envVars[i]], envIsSet = os.LookupEnv(envVars[i])
		if !envIsSet {
			missingVars = append(missingVars, envVars[i])
			allEnvIsSet = false
		}
	}
	if !allEnvIsSet {
		errorMsg := "Missing required environment variables:\n"
		for _, v := range missingVars {
			errorMsg += fmt.Sprintf("  - %s\n", v)
		}
		errorMsg += "\nPlease set these environment variables before running huskyCI client.\n"
		errorMsg += "\nExample:\n"
		errorMsg += "  export HUSKYCI_CLIENT_API_ADDR=\"https://api.huskyci.example.com\"\n"
		errorMsg += "  export HUSKYCI_CLIENT_REPO_URL=\"https://github.com/user/repo.git\"\n"
		errorMsg += "  export HUSKYCI_CLIENT_REPO_BRANCH=\"main\"\n"
		errorMsg += "  export HUSKYCI_CLIENT_TOKEN=\"your-token-here\"\n"
		return errors.New(errorMsg)
	}
	return nil
}

// getUseTLS returns TRUE or FALSE retrieved from an environment variable.
func getUseTLS() bool {
	option := os.Getenv("HUSKYCI_CLIENT_API_USE_HTTPS")
	if option == "true" || option == "1" || option == "TRUE" {
		return true
	}
	return false
}
