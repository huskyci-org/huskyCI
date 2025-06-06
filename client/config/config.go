package config

import (
	"errors"
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
	var errorString string

	env := make(map[string]string)
	allEnvIsSet = true
	for i := 0; i < len(envVars); i++ {
		env[envVars[i]], envIsSet = os.LookupEnv(envVars[i])
		if !envIsSet {
			errorString = errorString + envVars[i] + " "
			allEnvIsSet = false
		}
	}
	if !allEnvIsSet {
		return errors.New(errorString)
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
