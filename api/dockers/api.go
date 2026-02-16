package dockers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	apiContext "github.com/huskyci-org/huskyCI/api/context"
	"github.com/huskyci-org/huskyCI/api/log"
	goContext "golang.org/x/net/context"
)

// Docker is the docker struct
type Docker struct {
	CID    string `json:"Id"`
	client *client.Client
}

// CreateContainerPayload is a struct that represents all data needed to create a container.
type CreateContainerPayload struct {
	Image string   `json:"Image"`
	Tty   bool     `json:"Tty,omitempty"`
	Cmd   []string `json:"Cmd"`
}

const logActionNew = "NewDocker"
const logInfoAPI = "DOCKERAPI"

// NewDocker returns a new docker.
func NewDocker(dockerHost string) (*Docker, error) {
	configAPI, err := apiContext.DefaultConf.GetAPIConfig()
	if err != nil {
		log.Error(logActionNew, logInfoAPI, 3026, err)
		return nil, err
	}

	// env vars needed by docker/docker library to create a NewEnvClient:
	err = os.Setenv("DOCKER_HOST", dockerHost)
	if err != nil {
		log.Error(logActionNew, logInfoAPI, 3001, err)
		return nil, err
	}

	// Only set TLS-related environment variables for TCP/HTTPS connections, not Unix sockets
	isUnixSocket := strings.HasPrefix(dockerHost, "unix://")
	if !isUnixSocket {
		err = os.Setenv("DOCKER_CERT_PATH", configAPI.DockerHostsConfig.PathCertificate)
		if err != nil {
			log.Error(logActionNew, logInfoAPI, 3019, err)
			return nil, err
		}

		tlsVerify := strconv.Itoa(configAPI.DockerHostsConfig.TLSVerify)
		err = os.Setenv("DOCKER_TLS_VERIFY", tlsVerify)
		if err != nil {
			log.Error(logActionNew, logInfoAPI, 3020, err)
			return nil, err
		}
	} else {
		// Clear TLS-related environment variables for Unix socket connections
		os.Unsetenv("DOCKER_CERT_PATH")
		os.Unsetenv("DOCKER_TLS_VERIFY")
	}

	client, err := client.NewEnvClient()
	if err != nil {
		log.Error(logActionNew, logInfoAPI, 3002, err)
		return nil, err
	}
	docker := &Docker{
		client: client,
	}
	return docker, nil
}

// CreateContainer creates a new container and return its CID and an error
func (d Docker) CreateContainer(image, cmd string) (string, error) {
	return d.CreateContainerWithVolume(image, cmd, "")
}

// CreateContainerWithVolume creates a new container with an optional volume mount and returns its CID and an error
func (d Docker) CreateContainerWithVolume(image, cmd, volumePath string) (string, error) {
	ctx := goContext.Background()
	config := &container.Config{
		Image: image,
		Tty:   true,
		Cmd:   []string{"/bin/sh", "-c", cmd},
	}
	
	var hostConfig *container.HostConfig
	if volumePath != "" {
		// For docker-in-docker, bind mounts are resolved relative to the Docker daemon's host (dockerapi)
		// Since dockerapi has /tmp/huskyci-zips-host:/tmp/huskyci-zips mounted, the path should work
		// Mount the volume at /workspace in the container
		hostConfig = &container.HostConfig{
			Binds: []string{fmt.Sprintf("%s:/workspace:ro", volumePath)},
		}
	}
	
	resp, err := d.client.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	
	if err != nil {
		log.Error("CreateContainer", logInfoAPI, 3005, err)
		// If volume mount fails, log the error with more context
		if volumePath != "" {
			log.Error("CreateContainer", logInfoAPI, 3005, fmt.Errorf("failed to create container with volume mount %s: %v", volumePath, err))
		}
		return "", err
	}
	return resp.ID, nil
}

// CreateContainerWithVolumeRW creates a new container with a read-write volume mount
func (d Docker) CreateContainerWithVolumeRW(image, cmd, volumePath string) (string, error) {
	ctx := goContext.Background()
	config := &container.Config{
		Image: image,
		Tty:   true,
		Cmd:   []string{"/bin/sh", "-c", cmd},
	}
	
	var hostConfig *container.HostConfig
	if volumePath != "" {
		// For docker-in-docker, bind mounts are resolved relative to the Docker daemon's host (dockerapi)
		// Since dockerapi has /tmp/huskyci-zips-host:/tmp/huskyci-zips mounted, the path should work
		// Mount the volume at /workspace in the container with read-write access
		hostConfig = &container.HostConfig{
			Binds: []string{fmt.Sprintf("%s:/workspace", volumePath)}, // No :ro, so it's read-write
		}
	}
	
	resp, err := d.client.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	
	if err != nil {
		log.Error("CreateContainer", logInfoAPI, 3005, err)
		// If volume mount fails, log the error with more context
		if volumePath != "" {
			log.Error("CreateContainer", logInfoAPI, 3005, fmt.Errorf("failed to create container with volume mount %s: %v", volumePath, err))
		}
		return "", err
	}
	return resp.ID, nil
}

// StartContainer starts a container and returns its error.
func (d Docker) StartContainer() error {
	ctx := goContext.Background()
	return d.client.ContainerStart(ctx, d.CID, dockerTypes.ContainerStartOptions{})
}

// WaitContainer returns when container finishes executing cmd.
func (d Docker) WaitContainer(timeOutInSeconds int) error {
	ctx := goContext.Background()
	containerWaitC, errC := d.client.ContainerWait(ctx, d.CID, container.WaitConditionNotRunning)

	select {
	case err := <-errC:
		if err != nil {
			return err
		}
	case containerWait := <-containerWaitC:
		if containerWait.StatusCode != 0 {
			return fmt.Errorf("Error in POST to wait the container with statusCode %d", containerWait.StatusCode)
		}
	}

	return nil
}

// StopContainer stops an active container by it's CID
func (d Docker) StopContainer() error {
	ctx := goContext.Background()
	err := d.client.ContainerStop(ctx, d.CID, container.StopOptions{})
	if err != nil {
		log.Error("StopContainer", logInfoAPI, 3022, err)
	}
	return err
}

// RemoveContainer removes a container by it's CID
func (d Docker) RemoveContainer() error {
	ctx := goContext.Background()
	err := d.client.ContainerRemove(ctx, d.CID, dockerTypes.ContainerRemoveOptions{})
	if err != nil {
		log.Error("RemoveContainer", logInfoAPI, 3023, err)
	}
	return err
}

// ListStoppedContainers returns a Docker type list with CIDs of stopped containers
func (d Docker) ListStoppedContainers() ([]Docker, error) {

	ctx := goContext.Background()
	dockerFilters := filters.NewArgs()
	dockerFilters.Add("status", "exited")
	options := dockerTypes.ContainerListOptions{
		All:     true,
		Filters: dockerFilters,
	}

	containerList, err := d.client.ContainerList(ctx, options)
	if err != nil {
		log.Error("ListContainer", logInfoAPI, 3021, err)
		return nil, err
	}

	var dockerList []Docker
	for _, c := range containerList {
		docker := Docker{
			CID:    c.ID,
			client: d.client,
		}
		dockerList = append(dockerList, docker)
	}

	return dockerList, nil
}

// DieContainers stops and removes all containers
func (d Docker) DieContainers() error {
	containerList, err := d.ListStoppedContainers()
	if err != nil {
		return err
	}
	for _, c := range containerList {
		err := c.StopContainer()
		if err != nil {
			return err
		}
	}
	for _, c := range containerList {
		err := c.RemoveContainer()
		if err != nil {
			return err
		}
	}
	return nil
}

// ReadOutput returns STDOUT of a given containerID.
func (d Docker) ReadOutput() (string, error) {
	ctx := goContext.Background()
	out, err := d.client.ContainerLogs(ctx, d.CID, dockerTypes.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		log.Error("ReadOutput", logInfoAPI, 3006, err)
		return "", nil
	}

	body, err := ioutil.ReadAll(out)
	if err != nil {
		log.Error("ReadOutput", logInfoAPI, 3007, err)
		return "", err
	}
	return string(body), err
}

// ReadOutputStderr returns STDERR of a given containerID.
func (d Docker) ReadOutputStderr() (string, error) {
	ctx := goContext.Background()
	out, err := d.client.ContainerLogs(ctx, d.CID, dockerTypes.ContainerLogsOptions{ShowStderr: true})
	if err != nil {
		log.Error("ReadOutputStderr", logInfoAPI, 3006, err)
		return "", nil
	}

	body, err := ioutil.ReadAll(out)
	if err != nil {
		log.Error("ReadOutputStderr", logInfoAPI, 3008, err)
		return "", err
	}
	return string(body), err
}

// PullImage pulls an image, like docker pull.
// It reads the pull stream to capture detailed error messages, including platform mismatch errors.
func (d Docker) PullImage(image string) error {
	ctx := goContext.Background()
	reader, err := d.client.ImagePull(ctx, image, dockerTypes.ImagePullOptions{})
	if err != nil {
		log.Error("PullImage", logInfoAPI, 3009, fmt.Sprintf("Failed to start image pull for %s: %v", image, err))
		return err
	}
	defer reader.Close()

	// Read the pull stream to capture errors
	scanner := bufio.NewScanner(reader)
	var lastError string
	var pullError error

	for scanner.Scan() {
		line := scanner.Text()
		
		// Parse JSON lines from Docker pull stream
		var jsonLine map[string]interface{}
		if err := json.Unmarshal([]byte(line), &jsonLine); err == nil {
			// Check for error field
			if errorDetail, ok := jsonLine["errorDetail"].(map[string]interface{}); ok {
				if errorMsg, ok := errorDetail["message"].(string); ok {
					lastError = errorMsg
					// Check for platform mismatch errors
					if strings.Contains(strings.ToLower(errorMsg), "no matching manifest") ||
						strings.Contains(strings.ToLower(errorMsg), "platform") ||
						strings.Contains(strings.ToLower(errorMsg), "manifest unknown") {
						pullError = fmt.Errorf("platform mismatch or manifest not found: %s", errorMsg)
					} else {
						pullError = fmt.Errorf("pull error: %s", errorMsg)
					}
				}
			}
			// Check for error field at top level
			if errorMsg, ok := jsonLine["error"].(string); ok && errorMsg != "" {
				lastError = errorMsg
				if pullError == nil {
					pullError = fmt.Errorf("pull error: %s", errorMsg)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Error("PullImage", logInfoAPI, 3009, fmt.Sprintf("Error reading pull stream for %s: %v", image, err))
		if pullError == nil {
			return fmt.Errorf("failed to read pull stream: %w", err)
		}
	}

	if pullError != nil {
		log.Error("PullImage", logInfoAPI, 3009, fmt.Sprintf("Image pull failed for %s: %v (last error: %s)", image, pullError, lastError))
		return pullError
	}

	return nil
}

// ImageIsLoaded returns a bool if a a docker image is loaded or not.
func (d Docker) ImageIsLoaded(image string) bool {
	args := filters.NewArgs()
	args.Add("reference", image)
	options := dockerTypes.ImageListOptions{Filters: args}

	ctx := goContext.Background()
	result, err := d.client.ImageList(ctx, options)
	if err != nil {
		log.Error("ImageIsLoaded", logInfoAPI, 3010, err)
		panic(err)
	}

	return len(result) != 0
}

// ListImages returns docker images, like docker image ls.
func (d Docker) ListImages() ([]dockerTypes.ImageSummary, error) {
	ctx := goContext.Background()
	return d.client.ImageList(ctx, dockerTypes.ImageListOptions{})
}

// RemoveImage removes an image.
func (d Docker) RemoveImage(imageID string) ([]dockerTypes.ImageDeleteResponseItem, error) {
	ctx := goContext.Background()
	return d.client.ImageRemove(ctx, imageID, dockerTypes.ImageRemoveOptions{Force: true})
}

// HealthCheckDockerAPI returns true if a 200 status code is received from dockerAddress or false otherwise.
func HealthCheckDockerAPI(dockerHost string) error {
	d, err := NewDocker(dockerHost)
	if err != nil {
		log.Error("HealthCheckDockerAPI", logInfoAPI, 3011, err)
		return err
	}

	ctx := goContext.Background()
	_, err = d.client.Ping(ctx)
	return err
}
