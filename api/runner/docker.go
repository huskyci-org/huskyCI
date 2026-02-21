package runner

import (
	"context"
	"strings"

	"github.com/huskyci-org/huskyCI/api/dockers"
)

const logActionRun = "DockerRun"
const logInfoRunner = "RUNNER"

// DockerRunner implements Runner by delegating to the existing dockers package.
// It targets a single Docker daemon (host socket or TCP, e.g. no DinD requirement).
type DockerRunner struct {
	dockerHost string
}

// NewDockerRunner returns a Runner that uses the given Docker host address
// (e.g. "https://dockerapi:2376" or "unix:///var/run/docker.sock").
func NewDockerRunner(dockerHost string) *DockerRunner {
	return &DockerRunner{dockerHost: dockerHost}
}

// Run runs a container by calling dockers.DockerRunWithVolume, or when Stdin is set uses the low-level stream path.
// RunRequest.Image must be in "name:tag" form (e.g. "huskyciorg/bandit:1.9.3").
func (r *DockerRunner) Run(ctx context.Context, req RunRequest) (RunResult, error) {
	if req.Stdin != nil && req.ReadWriteVolume {
		return r.runWithStdin(ctx, req)
	}
	image, tag := splitImage(req.Image)
	var stdout, stderr string
	var err error
	if req.ReadWriteVolume {
		_, stdout, stderr, err = dockers.DockerRunWithVolumeRW(image, tag, req.Cmd, r.dockerHost, req.VolumePath, req.TimeoutSeconds)
	} else {
		_, stdout, stderr, err = dockers.DockerRunWithVolume(image, tag, req.Cmd, r.dockerHost, req.VolumePath, req.TimeoutSeconds)
	}
	if err != nil {
		return RunResult{Err: err}, err
	}
	return RunResult{Stdout: stdout, Stderr: stderr, ExitCode: 0}, nil
}

// runWithStdin runs a container with stdin streamed (e.g. zip into container).
func (r *DockerRunner) runWithStdin(ctx context.Context, req RunRequest) (RunResult, error) {
	d, err := dockers.NewDocker(r.dockerHost)
	if err != nil {
		return RunResult{Err: err}, err
	}
	if err := dockers.EnsureImageLoaded(d, req.Image); err != nil {
		return RunResult{Err: err}, err
	}
	timeout := req.TimeoutSeconds
	if timeout <= 0 {
		timeout = 300
	}
	CID, err := d.CreateContainerWithVolumeRWStdin(req.Image, req.Cmd, req.VolumePath)
	if err != nil {
		return RunResult{Err: err}, err
	}
	d.CID = CID
	if err := d.StartContainer(); err != nil {
		dockers.StopAndRemove(d)
		return RunResult{Err: err}, err
	}
	if err := d.AttachAndStreamStdin(req.Stdin); err != nil {
		dockers.StopAndRemove(d)
		return RunResult{Err: err}, err
	}
	if err := d.WaitContainer(timeout); err != nil {
		dockers.StopAndRemove(d)
		return RunResult{Err: err}, err
	}
	stdout, stderr, _ := d.ReadOutputBoth()
	_ = d.RemoveContainer()
	return RunResult{Stdout: stdout, Stderr: stderr, ExitCode: 0}, nil
}

// EnsureImage pulls the image if it is not present.
func (r *DockerRunner) EnsureImage(ctx context.Context, image string) error {
	d, err := dockers.NewDocker(r.dockerHost)
	if err != nil {
		return err
	}
	return dockers.EnsureImageLoaded(d, image)
}

// Health checks that the Docker daemon is reachable.
func (r *DockerRunner) Health(ctx context.Context) error {
	return dockers.HealthCheckDockerAPI(r.dockerHost)
}

func splitImage(full string) (image, tag string) {
	// Support "repo:tag" or "registry/repo:tag" (e.g. huskyciorg/bandit:1.9.3).
	// Use LastIndex so "localhost:5000/repo:tag" splits at the tag colon.
	i := strings.LastIndex(full, ":")
	if i == -1 {
		return full, "latest"
	}
	return full[:i], full[i+1:]
}
