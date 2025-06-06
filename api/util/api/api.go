package util

import (
	"errors"
	"fmt"
	"os"

	apiContext "github.com/huskyci-org/huskyCI/api/context"
	docker "github.com/huskyci-org/huskyCI/api/dockers"
	kube "github.com/huskyci-org/huskyCI/api/kubernetes"
	"github.com/huskyci-org/huskyCI/api/log"
	"github.com/huskyci-org/huskyCI/api/types"
	"github.com/huskyci-org/huskyCI/api/user"
	"go.mongodb.org/mongo-driver/mongo"
)

const logActionCheckReqs = "CheckHuskyRequirements"
const logInfoAPIUtil = "API-UTIL"

// CheckHuskyRequirements checks for all requirements needed before starting huskyCI.
func (hU HuskyUtils) CheckHuskyRequirements(configAPI *apiContext.APIConfig) error {

	// check if all environment variables are properly set.
	if err := hU.CheckHandler.checkEnvVars(); err != nil {
		return err
	}
	log.Info(logActionCheckReqs, logInfoAPIUtil, 12)

	// check infrastructure selection
	infrastructureSelected, hasSelected := os.LookupEnv("HUSKYCI_INFRASTRUCTURE_USE")
	if hasSelected {
		if infrastructureSelected == "docker" {
			// check if all docker hosts are up and running docker API.
			if err := hU.CheckHandler.checkDockerHosts(configAPI); err != nil {
				return err
			}
			log.Info(logActionCheckReqs, logInfoAPIUtil, 13)
		} else if infrastructureSelected == "kubernetes" {
			// check if all kubernetes hosts are up and running Kubernetes API.
			if err := hU.CheckHandler.checkKubernetesHosts(configAPI); err != nil {
				return err
			}
			log.Info(logActionCheckReqs, logInfoAPIUtil, 13)
		} else {
			return errors.New("invalid HUSKYCI_INFRASTRUCTURE_USE value")
		}
	} else {
		return errors.New("HUSKYCI_INFRASTRUCTURE_USE environment variable not set")
	}

	// check if DB is accessible and credentials received are working.
	if err := hU.CheckHandler.checkDB(configAPI); err != nil {
		return err
	}
	log.Info(logActionCheckReqs, logInfoAPIUtil, 14)

	// check if default securityTests are set into MongoDB.
	if err := hU.CheckHandler.checkEachSecurityTest(configAPI); err != nil {
		return err
	}
	log.Info(logActionCheckReqs, logInfoAPIUtil, 15)

	// check if default user is set into MongoDB.
	if err := hU.CheckHandler.checkDefaultUser(configAPI); err != nil {
		return err
	}
	log.Info(logActionCheckReqs, logInfoAPIUtil, 20)

	return nil
}

// checkEnvVar verifies if all required environment variables are set
func (cH *CheckUtils) checkEnvVars() error {

	envVars := []string{
		"HUSKYCI_DATABASE_DB_ADDR",
		"HUSKYCI_DATABASE_DB_NAME",
		"HUSKYCI_DATABASE_DB_USERNAME",
		"HUSKYCI_DATABASE_DB_PASSWORD",
		"HUSKYCI_API_DEFAULT_USERNAME",
		"HUSKYCI_API_DEFAULT_PASSWORD",
		"HUSKYCI_API_ALLOW_ORIGIN_CORS",
		"HUSKYCI_INFRASTRUCTURE_USE",
	}

	dockerEnvVars := []string{
		"HUSKYCI_DOCKERAPI_ADDR",
		"HUSKYCI_DOCKERAPI_CERT_PATH",
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

	infrastructureSelected, hasSelected := os.LookupEnv("HUSKYCI_INFRASTRUCTURE_USE")
	if hasSelected && infrastructureSelected == "docker" {
		for i := 0; i < len(dockerEnvVars); i++ {
			env[dockerEnvVars[i]], envIsSet = os.LookupEnv(dockerEnvVars[i])
			if !envIsSet {
				errorString = errorString + dockerEnvVars[i] + " "
				allEnvIsSet = false
			}
		}
	}

	if !allEnvIsSet {
		finalError := fmt.Sprintf("Check environment variables: %s", errorString)
		return errors.New(finalError)
	}

	return nil
}

func (cH *CheckUtils) checkDockerHosts(configAPI *apiContext.APIConfig) error {
	// writes necessary keys for TLS to respective files
	if err := createAPIKeys(); err != nil {
		return err
	}

	dockerHost := fmt.Sprintf("https://%s", configAPI.DockerHostsConfig.Host)

	return docker.HealthCheckDockerAPI(dockerHost)
}

func (cH *CheckUtils) checkKubernetesHosts(configAPI *apiContext.APIConfig) error {

	return kube.HealthCheckKubernetesAPI()
}

func (cH *CheckUtils) checkDB(configAPI *apiContext.APIConfig) error {
	if err := configAPI.DBInstance.ConnectDB(
		configAPI.DBConfig.Address,
		configAPI.DBConfig.DatabaseName,
		configAPI.DBConfig.Username,
		configAPI.DBConfig.Password,
		configAPI.DBConfig.Timeout,
		configAPI.DBConfig.PoolLimit,
		configAPI.DBConfig.Port,
		configAPI.DBConfig.MaxOpenConns,
		configAPI.DBConfig.MaxIdleConns,
		configAPI.DBConfig.ConnMaxLifetime); err != nil {
		dbError := fmt.Sprintf("Check DB: %s", err)
		return errors.New(dbError)
	}
	return nil
}

func (cH *CheckUtils) checkEachSecurityTest(configAPI *apiContext.APIConfig) error {
	securityTests := []string{"enry", "gitauthors", "gosec", "brakeman", "bandit", "npmaudit", "yarnaudit", "spotbugs", "gitleaks", "safety", "tfsec", "securitycodescan"}
	for _, securityTest := range securityTests {
		if err := checkSecurityTest(securityTest, configAPI); err != nil {
			errMsg := fmt.Sprintf("%s %s", securityTest, err)
			log.Error("checkEachSecurityTest", logInfoAPIUtil, 1023, errMsg)
			return err
		}
		log.Info("checkEachSecurityTest", logInfoAPIUtil, 19, securityTest)
	}
	return nil
}

func (cH *CheckUtils) checkDefaultUser(configAPI *apiContext.APIConfig) error {

	defaultUserQuery := map[string]interface{}{"username": user.DefaultAPIUser}
	_, err := configAPI.DBInstance.FindOneDBUser(defaultUserQuery)
	if err != nil {
		if err == mongo.ErrNoDocuments || err.Error() == "No data found" {
			// user not found, add default user
			if err := user.InsertDefaultUser(); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

func FormatDockerHostAddress(dockerHost types.DockerAPIAddresses, configAPI *apiContext.APIConfig) (string, error) {
	if len(dockerHost.HostList) == 0 {
		return "", errors.New("Docker host list is empty")
	}
	hostIndex := dockerHost.CurrentHostIndex % len(dockerHost.HostList)
	host := dockerHost.HostList[hostIndex]
	return fmt.Sprintf("https://%s:%d", host, configAPI.DockerHostsConfig.DockerAPIPort), nil
}

func checkSecurityTest(securityTestName string, configAPI *apiContext.APIConfig) error {

	var securityTestConfig types.SecurityTest

	switch securityTestName {
	case "enry":
		securityTestConfig = *configAPI.EnrySecurityTest
	case "gitauthors":
		securityTestConfig = *configAPI.GitAuthorsSecurityTest
	case "gosec":
		securityTestConfig = *configAPI.GosecSecurityTest
	case "brakeman":
		securityTestConfig = *configAPI.BrakemanSecurityTest
	case "bandit":
		securityTestConfig = *configAPI.BanditSecurityTest
	case "npmaudit":
		securityTestConfig = *configAPI.NpmAuditSecurityTest
	case "yarnaudit":
		securityTestConfig = *configAPI.YarnAuditSecurityTest
	case "spotbugs":
		securityTestConfig = *configAPI.SpotBugsSecurityTest
	case "gitleaks":
		securityTestConfig = *configAPI.GitleaksSecurityTest
	case "safety":
		securityTestConfig = *configAPI.SafetySecurityTest
	case "tfsec":
		securityTestConfig = *configAPI.TFSecSecurityTest
	case "securitycodescan":
		securityTestConfig = *configAPI.SecurityCodeScanSecurityTest
	default:
		return errors.New("securityTest name not defined")
	}

	securityTestQuery := map[string]interface{}{"name": securityTestName}
	_, err := configAPI.DBInstance.UpsertOneDBSecurityTest(securityTestQuery, securityTestConfig)
	if err != nil {
		return err
	}
	return nil
}

func createAPIKeys() error {
	err := createAPICert()
	if err != nil {
		return err
	}

	err = createAPIKey()
	if err != nil {
		return err
	}

	err = createAPITLSCert()
	if err != nil {
		return err
	}

	err = createAPITLSKey()
	if err != nil {
		return err
	}

	err = createAPICA()
	if err != nil {
		return err
	}

	return nil
}

func createAPICert() error {
	certValue, check := os.LookupEnv("HUSKYCI_DOCKERAPI_CERT_FILE_VALUE")
	if check {
		f, err := os.OpenFile("/home/application/current/api/cert.pem", os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}

		_, err = f.WriteString(certValue)
		if err != nil {
			return err
		}

		if err := f.Close(); err != nil {
			return err
		}

	}
	return nil
}

func createAPIKey() error {
	certKeyValue, check := os.LookupEnv("HUSKYCI_DOCKERAPI_CERT_KEY_VALUE")
	if check {
		f, err := os.OpenFile("/home/application/current/api/key.pem", os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}

		_, err = f.WriteString(certKeyValue)
		if err != nil {
			return err
		}

		if err := f.Close(); err != nil {
			return err
		}

	}
	return nil
}

func createAPITLSCert() error {
	apiCertValue, check := os.LookupEnv("HUSKYCI_DOCKERAPI_API_TLS_CERT_VALUE")
	if check {
		f, err := os.OpenFile("/home/application/current/api/api-tls-cert.pem", os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}

		_, err = f.WriteString(apiCertValue)
		if err != nil {
			return err
		}

		if err := f.Close(); err != nil {
			return err
		}

	}
	return nil
}

func createAPITLSKey() error {
	apiKeyValue, check := os.LookupEnv("HUSKYCI_DOCKERAPI_API_TLS_KEY_VALUE")
	if check {
		f, err := os.OpenFile("/home/application/current/api/api-tls-key.pem", os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}

		_, err = f.WriteString(apiKeyValue)
		if err != nil {
			return err
		}

		if err := f.Close(); err != nil {
			return err
		}

	}
	return nil
}

func createAPICA() error {
	caValue, check := os.LookupEnv("HUSKYCI_DOCKERAPI_CERT_CA_VALUE")
	if check {
		f, err := os.OpenFile("/home/application/current/api/ca.pem", os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}

		_, err = f.WriteString(caValue)
		if err != nil {
			return err
		}

		if err := f.Close(); err != nil {
			return err
		}

	}
	return nil
}
