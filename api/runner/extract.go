package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/huskyci-org/huskyCI/api/log"
)

// extractImage must have unzip pre-installed (no network in DinD extract container).
// Built from deployments/dockerfiles/extract/; for local DinD run 'make load-extract-image' after compose.
const extractImage = "huskyciorg/extract:latest"

// ExtractZip extracts a zip file in the runner's environment (e.g. in Docker API or remote runner).
// It first tries to stream the zip via Runner.Run with Stdin; if that fails, it runs a container
// that waits for the zip in the shared volume and extracts it. volumePath is the directory that
// contains the zip (and will receive the extracted dir); zipPath is the full path to the zip file
// (used for streaming); destDir is the destination directory name inside the volume.
func ExtractZip(ctx context.Context, r Runner, zipPath, destDir, volumePath string) error {
	zipFileName := filepath.Base(zipPath)
	destDirName := filepath.Base(destDir)
	streamIncomingName := ".incoming-" + zipFileName

	if err := r.EnsureImage(ctx, extractImage); err != nil {
		return fmt.Errorf("failed to ensure extract image %s: %w", extractImage, err)
	}

	// Try streaming the zip into the runner.
	streamSucceeded := false
	zipFile, err := os.Open(zipPath)
	if err != nil {
		log.Info("ExtractZip", logInfoRunner, 16, fmt.Sprintf("Could not open zip for streaming: %v; will use shared-volume extract", err))
	} else {
		streamCmd := fmt.Sprintf("cat > /workspace/%s", streamIncomingName)
		req := RunRequest{
			Image:           extractImage,
			Cmd:             streamCmd,
			VolumePath:      volumePath,
			TimeoutSeconds:  300,
			Stdin:           zipFile,
			ReadWriteVolume: true,
		}
		result, runErr := r.Run(ctx, req)
		zipFile.Close()
		if runErr != nil || result.Err != nil {
			log.Info("ExtractZip", logInfoRunner, 16, fmt.Sprintf("Stream run failed: %v; will use shared-volume extract", runErr))
		} else {
			streamSucceeded = true
		}
	}


	// Use image with unzip pre-installed (huskyciorg/extract:latest); no package install at runtime so extract works without container network (e.g. DinD).
	var extractCmd string
	if streamSucceeded {
		extractCmd = fmt.Sprintf("sh -c 'cd /workspace && mkdir -p %s && unzip -q -o %s -d %s && echo \"Extraction successful\"'",
			destDirName, streamIncomingName, destDirName)
	} else {
		const initialDelaySec = 2
		const retries = 60
		const retryDelaySec = "0.5"
		extractCmd = fmt.Sprintf("sh -c 'sleep %d && cd /workspace && "+
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

	log.Info("ExtractZip", logInfoRunner, 16, fmt.Sprintf("Extracting zip: zipPath=%s, destDir=%s, volumePath=%s", zipPath, destDir, volumePath))

	req := RunRequest{
		Image:           extractImage,
		Cmd:             extractCmd,
		VolumePath:      volumePath,
		TimeoutSeconds:  300,
		ReadWriteVolume: true,
	}
	result, err := r.Run(ctx, req)
	if err != nil {
		return fmt.Errorf("extract container: %w", err)
	}
	if result.Err != nil {
		// Log container output so we can see why it exited (e.g. "Zip not found", unzip error)
		log.Info("ExtractZip", logInfoRunner, 16, fmt.Sprintf("extract container failed: stdout=%q stderr=%q", result.Stdout, result.Stderr))
		return fmt.Errorf("extract container: %w", result.Err)
	}
	output := result.Stdout
	if strings.Contains(output, "ERROR") {
		return fmt.Errorf("extraction failed: %s", output)
	}
	return nil
}
