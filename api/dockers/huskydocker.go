package dockers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/huskyci-org/huskyCI/api/log"
)

const logActionRun = "DockerRun"
const logInfoHuskyDocker = "HUSKYDOCKER"
const logActionPull = "pullImage"

// #region agent log
const debugLogPath = "/debug/debug-c3d850.log"

func debugLog(message, hypothesisId string, data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["hypothesisId"] = hypothesisId
	payload := map[string]interface{}{
		"sessionId":    "c3d850",
		"timestamp":    time.Now().UnixMilli(),
		"location":    "huskydocker.go:ExtractZipInDockerAPI",
		"message":     message,
		"data":        data,
		"hypothesisId": hypothesisId,
	}
	b, _ := json.Marshal(payload)
	b = append(b, '\n')
	f, err := os.OpenFile(debugLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	_, _ = f.Write(b)
	_ = f.Close()
}

// #endregion

const urlRegexp = `([\w\-_]+(?:(?:\.[\w\-_]+)+))([\w\-\.,@?^=%&amp;:/~\+#]*[\w\-\@?^=%&amp;/~\+#])?`

func configureImagePath(image, tag string) (string, string) {
	fullContainerImage := fmt.Sprintf("%s:%s", image, tag)
	regex := regexp.MustCompile(urlRegexp)
	canonicalURL := image
	if !regex.MatchString(canonicalURL) {
		canonicalURL = fmt.Sprintf("docker.io/%s", fullContainerImage)
	} else {
		canonicalURL = fullContainerImage
	}

	return canonicalURL, fullContainerImage
}

// DockerRun starts a new container and returns its output and an error.
func DockerRun(image, imageTag, cmd, dockerHost string, timeOutInSeconds int) (string, string, string, error) {
	return DockerRunWithVolume(image, imageTag, cmd, dockerHost, "", timeOutInSeconds)
}

// DockerRunWithVolume starts a new container with an optional volume mount and returns CID, stdout, stderr and an error.
// Uses a single ContainerLogs + StdCopy pass so stderr is available when stdout is empty for diagnostics.
func DockerRunWithVolume(image, imageTag, cmd, dockerHost, volumePath string, timeOutInSeconds int) (string, string, string, error) {

	// step 1: create a new docker API client
	d, err := NewDocker(dockerHost)
	if err != nil {
		return "", "", "", err
	}

	canonicalURL, fullContainerImage := configureImagePath(image, imageTag)
	// step 2: pull image if it is not there yet
	if !d.ImageIsLoaded(fullContainerImage) {
		if err := pullImage(d, canonicalURL, fullContainerImage); err != nil {
			return "", "", "", err
		}
	}

	// step 2.5: For file:// URLs, ensure dockerapi can see the files
	// docker-in-docker has issues with bind mounts - dockerapi can't see files written by API container
	// Use a temporary container to refresh dockerapi's view of the mount
	if volumePath != "" {
		log.Info(logActionRun, logInfoHuskyDocker, 16, fmt.Sprintf("Mounting volume path: %s (resolved relative to Docker daemon host)", volumePath))
		// Sync files to dockerapi using a temporary container
		if err := syncFilesToDockerAPI(d, volumePath); err != nil {
			log.Error(logActionRun, logInfoHuskyDocker, 3016, fmt.Errorf("failed to sync files to dockerapi: %v (continuing anyway)", err))
			// Continue anyway - the mount might still work
		}
	}

	// step 3: create a new container given an image and it's cmd
	CID, err := d.CreateContainerWithVolume(fullContainerImage, cmd, volumePath)
	if err != nil {
		return "", "", "", err
	}
	d.CID = CID

	// step 4: start container
	if err := d.StartContainer(); err != nil {
		log.Error(logActionRun, logInfoHuskyDocker, 3015, err)
		return "", "", "", err
	}
	log.Info(logActionRun, logInfoHuskyDocker, 32, fullContainerImage, d.CID)

	// step 5: wait container finish
	if err := d.WaitContainer(timeOutInSeconds); err != nil {
		log.Error(logActionRun, logInfoHuskyDocker, 3016, err)
		return "", "", "", err
	}

	// step 6: read container output (single-pass stdout + stderr)
	stdout, stderr, err := d.ReadOutputBoth()
	if err != nil {
		return "", "", "", err
	}
	log.Info(logActionRun, logInfoHuskyDocker, 34, fullContainerImage, d.CID)

	// step 7: remove container from docker API
	if err := d.RemoveContainer(); err != nil {
		log.Error(logActionRun, logInfoHuskyDocker, 3027, err)
		return "", "", "", err
	}

	return CID, stdout, stderr, nil
}

// DockerRunWithVolumeRW is like DockerRunWithVolume but mounts the volume read-write (no :ro).
// Use when the container must write to the mount (e.g. unzip extraction).
func DockerRunWithVolumeRW(image, imageTag, cmd, dockerHost, volumePath string, timeOutInSeconds int) (string, string, string, error) {
	d, err := NewDocker(dockerHost)
	if err != nil {
		return "", "", "", err
	}
	canonicalURL, fullContainerImage := configureImagePath(image, imageTag)
	if !d.ImageIsLoaded(fullContainerImage) {
		if err := pullImage(d, canonicalURL, fullContainerImage); err != nil {
			return "", "", "", err
		}
	}
	if volumePath != "" {
		log.Info(logActionRun, logInfoHuskyDocker, 16, fmt.Sprintf("Mounting volume path (rw): %s", volumePath))
		if err := syncFilesToDockerAPI(d, volumePath); err != nil {
			log.Error(logActionRun, logInfoHuskyDocker, 3016, fmt.Errorf("failed to sync files to dockerapi: %v (continuing anyway)", err))
		}
	}
	CID, err := d.CreateContainerWithVolumeRW(fullContainerImage, cmd, volumePath)
	if err != nil {
		return "", "", "", err
	}
	d.CID = CID
	if err := d.StartContainer(); err != nil {
		log.Error(logActionRun, logInfoHuskyDocker, 3015, err)
		return "", "", "", err
	}
	log.Info(logActionRun, logInfoHuskyDocker, 32, fullContainerImage, d.CID)
	waitErr := d.WaitContainer(timeOutInSeconds)
	stdout, stderr, readErr := d.ReadOutputBoth()
	if readErr != nil {
		log.Error(logActionRun, logInfoHuskyDocker, 3006, readErr)
	}
	if err := d.RemoveContainer(); err != nil {
		log.Error(logActionRun, logInfoHuskyDocker, 3027, err)
	}
	if waitErr != nil {
		log.Error(logActionRun, logInfoHuskyDocker, 3016, waitErr)
		// Include container output so callers can see why the container exited (e.g. unzip error, "Zip not found")
		return "", stdout, stderr, fmt.Errorf("%w (stdout: %q stderr: %q)", waitErr, stdout, stderr)
	}
	log.Info(logActionRun, logInfoHuskyDocker, 34, fullContainerImage, d.CID)
	return CID, stdout, stderr, nil
}

// StopAndRemove stops the container (if running) then removes it. Use in error paths to avoid "container is running" on remove.
func StopAndRemove(d *Docker) {
	_ = d.StopContainer()
	_ = d.RemoveContainer()
}

// ExtractZipInDockerAPI extracts a zip file directly in dockerapi using a temporary container.
// It first tries to stream the zip from the API into dockerapi; if that fails (e.g. attach not supported),
// it falls back to waiting for the zip in the shared volume and then extracting.
func ExtractZipInDockerAPI(dockerHost, zipPath, destDir string) error {
	zipFileName := filepath.Base(zipPath)
	parentDir := filepath.Dir(zipPath)
	destDirName := filepath.Base(destDir)
	volumePath := parentDir

	d, err := NewDocker(dockerHost)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	canonicalURL, fullContainerImage := configureImagePath("alpine", "latest")
	log.Info("ExtractZipInDockerAPI", logInfoHuskyDocker, 16, fmt.Sprintf("Checking for image %s (canonical: %s) in dockerapi...", fullContainerImage, canonicalURL))
	isLoaded := d.ImageIsLoaded(fullContainerImage)
	log.Info("ExtractZipInDockerAPI", logInfoHuskyDocker, 16, fmt.Sprintf("Image %s loaded: %v", fullContainerImage, isLoaded))
	if !isLoaded {
		log.Info("ExtractZipInDockerAPI", logInfoHuskyDocker, 31, fmt.Sprintf("Pulling image %s (canonical: %s) in dockerapi...", fullContainerImage, canonicalURL))
		if err := pullImage(d, canonicalURL, fullContainerImage); err != nil {
			return fmt.Errorf("failed to pull alpine:latest image: %w", err)
		}
		log.Info("ExtractZipInDockerAPI", logInfoHuskyDocker, 35, fmt.Sprintf("Successfully pulled image %s", fullContainerImage))
	} else {
		log.Info("ExtractZipInDockerAPI", logInfoHuskyDocker, 35, fmt.Sprintf("Image %s already loaded, skipping pull", fullContainerImage))
	}

	// Stream to a temporary path so we never truncate the shared-volume zip (API may have already written RID.zip).
	streamIncomingName := ".incoming-" + zipFileName
	streamSucceeded := false
	log.Info("ExtractZipInDockerAPI", logInfoHuskyDocker, 16, fmt.Sprintf("Streaming zip into dockerapi: %s", zipFileName))
	// #region agent log
	debugLog("stream_create_start", "H2", map[string]interface{}{"volumePath": volumePath})
	// #endregion
	zipFile, err := os.Open(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file for streaming: %w", err)
	}
	streamCmd := fmt.Sprintf("cat > /workspace/%s", streamIncomingName)
	streamCID, errCreate := d.CreateContainerWithVolumeRWStdin(fullContainerImage, streamCmd, volumePath)
	zipFile.Close()
	if errCreate != nil {
		log.Info("ExtractZipInDockerAPI", logInfoHuskyDocker, 16, fmt.Sprintf("Stream container create failed, will use shared-volume extract: %v", errCreate))
	} else {
		d.CID = streamCID
		if err := d.StartContainer(); err != nil {
			StopAndRemove(d)
			log.Info("ExtractZipInDockerAPI", logInfoHuskyDocker, 16, fmt.Sprintf("Stream container start failed, will use shared-volume extract: %v", err))
		} else {
			zipFile2, _ := os.Open(zipPath)
			attachErr := d.AttachAndStreamStdin(zipFile2)
			zipFile2.Close()
			if attachErr != nil {
				StopAndRemove(d)
				log.Info("ExtractZipInDockerAPI", logInfoHuskyDocker, 16, fmt.Sprintf("Stream attach failed, will use shared-volume extract: %v", attachErr))
			} else if err := d.WaitContainer(300); err != nil {
				output, _ := d.ReadOutput()
				// #region agent log
				debugLog("stream_wait_failed", "H2", map[string]interface{}{"cid": d.CID, "err": err.Error(), "output_len": len(output)})
				// #endregion
				StopAndRemove(d)
				log.Info("ExtractZipInDockerAPI", logInfoHuskyDocker, 16, fmt.Sprintf("Stream container wait failed, will use shared-volume extract: %v (output: %s)", err, output))
			} else {
				streamSucceeded = true
				if err := d.RemoveContainer(); err != nil {
					log.Error("ExtractZipInDockerAPI", logInfoHuskyDocker, 3027, fmt.Errorf("failed to remove stream container: %v", err))
				}
			}
		}
	}

	var extractCmd string
	if streamSucceeded {
		// Extract from the streamed file (temporary path).
		extractCmd = fmt.Sprintf("sh -c 'apk add --no-cache unzip > /dev/null 2>&1 && cd /workspace && mkdir -p %s && unzip -q -o %s -d %s && echo \"Extraction successful\"'",
			destDirName, streamIncomingName, destDirName)
	} else {
		// Fallback: wait for zip in /workspace (shared volume) or non-empty .incoming-* (from stream), then extract.
		// Remove only empty .incoming-* left by a failed stream.
		const initialDelaySec = 2
		const retries = 60
		const retryDelaySec = "0.5"
		extractCmd = fmt.Sprintf("sh -c 'apk add --no-cache unzip > /dev/null 2>&1 && sleep %d && cd /workspace && "+
			"for f in .incoming-*; do [ -f \"$f\" ] && [ ! -s \"$f\" ] && rm -f \"$f\"; done 2>/dev/null; "+
			"for i in $(seq 1 %d); do "+
			"if [ -f %s ] && [ -s %s ]; then mkdir -p %s && unzip -q -o %s -d %s && echo \"Extraction successful\" && exit 0; fi; "+
			"if [ -f %s ] && [ -s %s ]; then mkdir -p %s && unzip -q -o %s -d %s && echo \"Extraction successful\" && exit 0; fi; "+
			"sleep %s; done; "+
			"echo \"ERROR: Zip not found or empty in /workspace after retries. Ensure API and Docker API share the same volume (e.g. -v /tmp/huskyci-zips-host:/tmp/huskyci-zips on both).\"; ls -la /workspace 2>&1; exit 1'",
			initialDelaySec, retries, zipFileName, zipFileName, destDirName, zipFileName, destDirName,
			streamIncomingName, streamIncomingName, destDirName, streamIncomingName, destDirName,
			retryDelaySec)
	}
	log.Info("ExtractZipInDockerAPI", logInfoHuskyDocker, 16, fmt.Sprintf("Extracting zip in dockerapi: zipPath=%s, destDir=%s, volumePath=%s", zipPath, destDir, volumePath))
	// #region agent log
	debugLog("extract_create_start", "H3", map[string]interface{}{"volumePath": volumePath})
	// #endregion

	CID, err := d.CreateContainerWithVolumeRW(fullContainerImage, extractCmd, volumePath)
	if err != nil {
		return fmt.Errorf("failed to create extract container: %w", err)
	}
	d.CID = CID
	if err := d.StartContainer(); err != nil {
		StopAndRemove(d)
		return fmt.Errorf("failed to start extract container: %w", err)
	}
	if err := d.WaitContainer(300); err != nil {
		output, _ := d.ReadOutput()
		// #region agent log
		debugLog("extract_wait_failed", "H3", map[string]interface{}{"cid": d.CID, "err": err.Error(), "output_len": len(output)})
		// #endregion
		StopAndRemove(d)
		return fmt.Errorf("extract container error: %w (output: %s)", err, output)
	}
	output, _ := d.ReadOutput()
	if strings.Contains(output, "ERROR") {
		StopAndRemove(d)
		return fmt.Errorf("extraction failed: %s", output)
	}
	if err := d.RemoveContainer(); err != nil {
		log.Error("ExtractZipInDockerAPI", logInfoHuskyDocker, 3027, fmt.Errorf("failed to remove extract container: %v", err))
	}
	return nil
}

// syncFilesToDockerAPI ensures dockerapi can see files by using a temporary container
// to refresh dockerapi's view of the mount. Since docker-in-docker doesn't properly
// share bind mounts between containers, we use a temporary container to ensure
// dockerapi's Docker daemon can see files written by the API container.
func syncFilesToDockerAPI(d *Docker, volumePath string) error {
	// Use a temporary alpine container to list files in the volume
	// This forces dockerapi's Docker daemon to refresh its view of the mount
	// The container mounts the volume and lists files to ensure they're visible
	syncCmd := fmt.Sprintf("sh -c 'ls -la %s > /dev/null 2>&1 || true'", volumePath)
	
	// Create a temporary container with the volume mounted
	tempCID, err := d.CreateContainerWithVolume("alpine:latest", syncCmd, volumePath)
	if err != nil {
		return fmt.Errorf("failed to create sync container: %w", err)
	}
	
	// Start and wait for the container
	d.CID = tempCID
	if err := d.StartContainer(); err != nil {
		d.RemoveContainer() // Clean up on error
		return fmt.Errorf("failed to start sync container: %w", err)
	}
	
	// Wait for container to finish (should be very quick)
	if err := d.WaitContainer(30); err != nil {
		StopAndRemove(d)
		return fmt.Errorf("sync container error: %w", err)
	}

	// Clean up temporary container
	if err := d.RemoveContainer(); err != nil {
		// Log but don't fail - this is cleanup
		log.Error(logActionRun, logInfoHuskyDocker, 3027, fmt.Errorf("failed to remove sync container: %v", err))
	}
	
	return nil
}

// EnsureImageLoaded ensures the image (format "name:tag") is available on the given Docker client, pulling if necessary.
func EnsureImageLoaded(d *Docker, fullImage string) error {
	parts := strings.SplitN(fullImage, ":", 2)
	image, tag := parts[0], "latest"
	if len(parts) == 2 {
		tag = parts[1]
	}
	canonicalURL, full := configureImagePath(image, tag)
	if d.ImageIsLoaded(full) {
		return nil
	}
	return pullImage(d, canonicalURL, full)
}

func pullImage(d *Docker, canonicalURL, image string) error {
	timeout := time.After(15 * time.Minute)
	retryTick := time.NewTicker(15 * time.Second)
	maxRetries := 3
	retryCount := 0
	
	for {
		select {
		case <-timeout:
			timeOutErr := errors.New("timeout after 15 minutes")
			log.Error(logActionPull, logInfoHuskyDocker, 3013, fmt.Sprintf("Image pull timeout for %s: %v", image, timeOutErr))
			return timeOutErr
		case <-retryTick.C:
			log.Info(logActionPull, logInfoHuskyDocker, 31, fmt.Sprintf("Attempting to pull image: %s (attempt %d)", image, retryCount+1))
			
			// Check if image is already loaded
			if d.ImageIsLoaded(image) {
				log.Info(logActionPull, logInfoHuskyDocker, 35, fmt.Sprintf("Image already loaded: %s", image))
				return nil
			}
			
			// Attempt to pull the image
			if err := d.PullImage(canonicalURL); err != nil {
				retryCount++
				
				// Check if it's a platform mismatch error - fail immediately
				errStr := err.Error()
				if strings.Contains(strings.ToLower(errStr), "no matching manifest") ||
					strings.Contains(strings.ToLower(errStr), "platform") ||
					strings.Contains(strings.ToLower(errStr), "manifest unknown") ||
					strings.Contains(strings.ToLower(errStr), "manifest not found") {
					log.Error(logActionPull, logInfoHuskyDocker, 3013, fmt.Sprintf("Platform mismatch error for %s - failing immediately: %v", image, err))
					return fmt.Errorf("platform mismatch or manifest not found for %s: %w", image, err)
				}
				
				// For other errors, retry up to maxRetries times
				if retryCount >= maxRetries {
					log.Error(logActionPull, logInfoHuskyDocker, 3013, fmt.Sprintf("Failed to pull image %s (attempt %d/%d): %v", image, retryCount, maxRetries, err))
					return fmt.Errorf("failed to pull image %s after %d attempts: %w", image, maxRetries, err)
				}
				
				log.Info(logActionPull, logInfoHuskyDocker, 31, fmt.Sprintf("Failed to pull image %s (attempt %d/%d), retrying in 15 seconds...", image, retryCount, maxRetries))
				continue
			}
			
			// Pull succeeded, verify image is loaded
			if d.ImageIsLoaded(image) {
				log.Info(logActionPull, logInfoHuskyDocker, 35, fmt.Sprintf("Successfully pulled and loaded image: %s", image))
				return nil
			}
			
			// Pull reported success but image not loaded - retry
			retryCount++
			if retryCount >= maxRetries {
				return fmt.Errorf("image %s pull reported success but image not found after %d attempts", image, maxRetries)
			}
			log.Info(logActionPull, logInfoHuskyDocker, 31, fmt.Sprintf("Pull succeeded but image not found, retrying for %s...", image))
		}
	}
}
