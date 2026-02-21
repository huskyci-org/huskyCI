package runner

import (
	"context"
	"io"
)

// RunRequest holds parameters for a single container run.
type RunRequest struct {
	Image            string
	Cmd              string
	VolumePath       string
	TimeoutSeconds   int
	Stdin            io.Reader // optional, for streaming (e.g. zip) into container
	ReadWriteVolume  bool
}

// RunResult holds stdout, stderr, exit code and optional error from a container run.
type RunResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

// Runner runs containers via a single abstraction (Docker daemon or remote runner service).
type Runner interface {
	// Run runs a container with the given request and returns output and exit code.
	Run(ctx context.Context, req RunRequest) (RunResult, error)
	// EnsureImage pulls the image if it is not present.
	EnsureImage(ctx context.Context, image string) error
	// Health checks that the runner (Docker daemon or remote service) is reachable.
	Health(ctx context.Context) error
}

// defaultRunner is set when HUSKYCI_INFRASTRUCTURE_USE=docker (e.g. in CheckHuskyRequirements).
var defaultRunner Runner

// SetDefault sets the default runner used by security tests and zip extraction.
func SetDefault(r Runner) {
	defaultRunner = r
}

// Default returns the default runner, or nil if not set.
func Default() Runner {
	return defaultRunner
}
