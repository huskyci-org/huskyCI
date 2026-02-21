package runner

// NewFromConfig returns a Runner for the current configuration. Today this is always
// a Docker runner using the given dockerHost (formatted address, e.g. from
// HUSKYCI_DOCKERAPI_ADDR and HUSKYCI_DOCKERAPI_PORT). When HUSKYCI_RUNNER_TYPE=remote
// is implemented, this will read env and may return a RemoteRunner instead.
func NewFromConfig(dockerHost string) Runner {
	return NewDockerRunner(dockerHost)
}
