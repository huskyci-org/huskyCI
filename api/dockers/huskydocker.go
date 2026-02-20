package dockers

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"regexp"

	"github.com/huskyci-org/huskyCI/api/log"
)

const logActionRun = "DockerRun"
const logInfoHuskyDocker = "HUSKYDOCKER"
const logActionPull = "pullImage"

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
func DockerRun(image, imageTag, cmd, dockerHost string, timeOutInSeconds int) (string, string, error) {
	return DockerRunWithVolume(image, imageTag, cmd, dockerHost, "", timeOutInSeconds)
}

// DockerRunWithVolume starts a new container with an optional volume mount and returns its output and an error.
func DockerRunWithVolume(image, imageTag, cmd, dockerHost, volumePath string, timeOutInSeconds int) (string, string, error) {

	// step 1: create a new docker API client
	d, err := NewDocker(dockerHost)
	if err != nil {
		return "", "", err
	}

	canonicalURL, fullContainerImage := configureImagePath(image, imageTag)
	// step 2: pull image if it is not there yet
	if !d.ImageIsLoaded(fullContainerImage) {
		if err := pullImage(d, canonicalURL, fullContainerImage); err != nil {
			return "", "", err
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
		return "", "", err
	}
	d.CID = CID

	// step 4: start container
	if err := d.StartContainer(); err != nil {
		log.Error(logActionRun, logInfoHuskyDocker, 3015, err)
		return "", "", err
	}
	log.Info(logActionRun, logInfoHuskyDocker, 32, fullContainerImage, d.CID)

	// step 5: wait container finish
	if err := d.WaitContainer(timeOutInSeconds); err != nil {
		log.Error(logActionRun, logInfoHuskyDocker, 3016, err)
		return "", "", err
	}

	// step 6: read container's output when it finishes
	cOutput, err := d.ReadOutput()
	if err != nil {
		return "", "", err
	}
	log.Info(logActionRun, logInfoHuskyDocker, 34, fullContainerImage, d.CID)

	// step 7: remove container from docker API
	if err := d.RemoveContainer(); err != nil {
		log.Error(logActionRun, logInfoHuskyDocker, 3027, err)
		return "", "", err
	}

	return CID, cOutput, nil
}

// ExtractZipInDockerAPI extracts a zip file directly in dockerapi using a temporary container
// This ensures dockerapi's Docker daemon can see the extracted files immediately
// Since docker-in-docker doesn't properly share bind mounts, we need to ensure dockerapi can see the files
// The zip file exists on the host at /tmp/huskyci-zips-host/, and dockerapi has this mounted at /tmp/huskyci-zips/
// When dockerapi creates containers, it resolves paths relative to dockerapi's filesystem
// So /tmp/huskyci-zips/<RID>.zip in dockerapi = /tmp/huskyci-zips-host/<RID>.zip on host
func ExtractZipInDockerAPI(dockerHost, zipPath, destDir string) error {
	// Extract zip file name and directory from path
	zipFileName := filepath.Base(zipPath)
	parentDir := filepath.Dir(zipPath)
	
	// Use a temporary alpine container with unzip to extract files
	// Mount the parent directory - dockerapi resolves this relative to its filesystem
	// Since dockerapi has /tmp/huskyci-zips-host:/tmp/huskyci-zips mounted,
	// mounting /tmp/huskyci-zips should work and access files on the host
	// The volume is mounted at /workspace in the container (see CreateContainerWithVolume)
	// So we need to use /workspace instead of the original path
	// The zip file and destination directory are relative to the parent directory
	destDirName := filepath.Base(destDir)
	// Add retry logic to wait for file to be visible to dockerapi's Docker daemon
	// Use a loop with small delays to check if file exists before attempting extraction
	// Retry up to 30 times with 0.5s delay (15s total) so large uploads are visible to dockerapi before extraction
	extractCmd := fmt.Sprintf("sh -c 'apk add --no-cache unzip > /dev/null 2>&1 && cd /workspace && "+
		"for i in $(seq 1 30); do "+
		"if [ -f %s ]; then "+
		"mkdir -p %s && unzip -q -o %s -d %s && echo \"Extraction successful\" && exit 0; "+
		"fi; "+
		"sleep 0.5; "+
		"done; "+
		"echo \"ERROR: Zip file %s not found in /workspace after retries\"; "+
		"ls -la /workspace 2>&1; "+
		"exit 1'", zipFileName, destDirName, zipFileName, destDirName, zipFileName)
	
	// Create Docker client for dockerapi
	d, err := NewDocker(dockerHost)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	
	// Mount the parent directory - dockerapi will resolve this relative to its filesystem
	// We need read-write access to extract files, so we'll mount it as rw
	volumePath := parentDir
	
	// Ensure alpine:latest image is available in dockerapi
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
	
	log.Info("ExtractZipInDockerAPI", logInfoHuskyDocker, 16, fmt.Sprintf("Extracting zip in dockerapi: zipPath=%s, destDir=%s, volumePath=%s", zipPath, destDir, volumePath))
	
	// Create container with read-write mount so we can extract files
	// We need to use CreateContainerWithVolumeRW instead of CreateContainerWithVolume
	CID, err := d.CreateContainerWithVolumeRW(fullContainerImage, extractCmd, volumePath)
	if err != nil {
		return fmt.Errorf("failed to create extract container: %w", err)
	}
	d.CID = CID
	
	// Start container
	if err := d.StartContainer(); err != nil {
		d.RemoveContainer()
		return fmt.Errorf("failed to start extract container: %w", err)
	}
	
	// Wait for container to finish (allow up to 5 minutes for large zip files)
	if err := d.WaitContainer(300); err != nil {
		// Read container output to see what went wrong
		output, _ := d.ReadOutput()
		d.RemoveContainer()
		return fmt.Errorf("extract container error: %w (output: %s)", err, output)
	}
	
	// Verify extraction succeeded by reading output
	output, _ := d.ReadOutput()
	if strings.Contains(output, "ERROR") {
		d.RemoveContainer()
		return fmt.Errorf("extraction failed: %s", output)
	}
	
	// Clean up
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
		d.RemoveContainer()
		return fmt.Errorf("sync container error: %w", err)
	}
	
	// Clean up temporary container
	if err := d.RemoveContainer(); err != nil {
		// Log but don't fail - this is cleanup
		log.Error(logActionRun, logInfoHuskyDocker, 3027, fmt.Errorf("failed to remove sync container: %v", err))
	}
	
	return nil
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
