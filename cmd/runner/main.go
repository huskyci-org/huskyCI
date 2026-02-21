// Runner service: HTTP server that runs containers on the host Docker daemon.
// Used when HUSKYCI_RUNNER_TYPE=remote; the HuskyCI API sends POST /run and GET /health.
// Run with Docker socket mounted (e.g. -v /var/run/docker.sock:/var/run/docker.sock).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const defaultPort = "8090"

type runRequest struct {
	Image           string `json:"image"`
	Cmd             string `json:"cmd"`
	VolumePath      string `json:"volumePath"`
	TimeoutSeconds  int    `json:"timeoutSeconds"`
	ReadWriteVolume bool   `json:"readWriteVolume"`
}

type runResponse struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
	Error    string `json:"error,omitempty"`
}

func main() {
	port := os.Getenv("RUNNER_PORT")
	if port == "" {
		port = defaultPort
	}
	addr := ":" + port

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Docker client: %v", err)
	}
	defer cli.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler(cli))
	mux.HandleFunc("/run", runHandler(cli))

	log.Printf("Runner service listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server: %v", err)
	}
}

func healthHandler(cli *client.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if _, err := cli.Ping(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func runHandler(cli *client.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		req, stdin, err := parseRunRequest(r)
		if err != nil {
			writeRunError(w, 0, err)
			return
		}
		stdout, stderr, exitCode, err := runContainer(r.Context(), cli, req, stdin)
		if err != nil {
			writeRunError(w, exitCode, err)
			return
		}
		json.NewEncoder(w).Encode(runResponse{Stdout: stdout, Stderr: stderr, ExitCode: exitCode})
	}
}

func parseRunRequest(r *http.Request) (runRequest, io.Reader, error) {
	var req runRequest
	var stdin io.Reader
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(50 << 20); err != nil {
			return req, nil, err
		}
		reqStr := r.FormValue("request")
		if reqStr == "" {
			return req, nil, fmt.Errorf("missing form part 'request'")
		}
		if err := json.Unmarshal([]byte(reqStr), &req); err != nil {
			return req, nil, err
		}
		if f, _, err := r.FormFile("stdin"); err == nil {
			stdin = f
			defer f.Close()
		}
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return req, nil, err
		}
	}
	if req.Image == "" || req.Cmd == "" {
		return req, nil, fmt.Errorf("image and cmd are required")
	}
	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = 300
	}
	return req, stdin, nil
}

func writeRunError(w http.ResponseWriter, exitCode int, err error) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(runResponse{ExitCode: exitCode, Error: err.Error()})
}

func runContainer(ctx context.Context, cli *client.Client, req runRequest, stdin io.Reader) (stdout, stderr string, exitCode int, err error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds+30)*time.Second)
	defer cancel()
	if err := ensureImage(ctx, cli, req.Image); err != nil {
		return "", "", 1, err
	}
	config, hostConfig := containerConfig(req, stdin != nil)
	createResp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return "", "", 1, fmt.Errorf("create container: %w", err)
	}
	cid := createResp.ID
	defer func() {
		cli.ContainerRemove(context.Background(), cid, types.ContainerRemoveOptions{Force: true})
	}()
	if err := cli.ContainerStart(ctx, cid, types.ContainerStartOptions{}); err != nil {
		return "", "", 1, fmt.Errorf("start container: %w", err)
	}
	if stdin != nil {
		if err := streamStdin(ctx, cli, cid, stdin); err != nil {
			return "", "", 1, err
		}
	}
	exitCode, err = waitContainer(ctx, cli, cid)
	if err != nil {
		return "", "", exitCode, err
	}
	stdout, stderr, err = containerLogs(ctx, cli, cid)
	return stdout, stderr, exitCode, err
}

func ensureImage(ctx context.Context, cli *client.Client, image string) error {
	_, _, err := cli.ImageInspectWithRaw(ctx, image)
	if err == nil {
		return nil
	}
	rc, pullErr := cli.ImagePull(ctx, image, types.ImagePullOptions{})
	if pullErr != nil {
		return fmt.Errorf("pull image: %w", pullErr)
	}
	io.Copy(io.Discard, rc)
	rc.Close()
	return nil
}

func containerConfig(req runRequest, openStdin bool) (*container.Config, *container.HostConfig) {
	config := &container.Config{
		Image:     req.Image,
		Tty:       false,
		OpenStdin: openStdin,
		StdinOnce: openStdin,
		Cmd:       []string{"/bin/sh", "-c", req.Cmd},
	}
	hostConfig := &container.HostConfig{}
	if req.VolumePath != "" {
		mode := ":ro"
		if req.ReadWriteVolume {
			mode = ""
		}
		hostConfig.Binds = []string{fmt.Sprintf("%s:/workspace%s", req.VolumePath, mode)}
	}
	return config, hostConfig
}

func streamStdin(ctx context.Context, cli *client.Client, cid string, stdin io.Reader) error {
	opts := container.AttachOptions{Stream: true, Stdin: true}
	attachResp, err := cli.ContainerAttach(ctx, cid, opts)
	if err != nil {
		return fmt.Errorf("attach stdin: %w", err)
	}
	defer attachResp.Close()
	if _, err := io.Copy(attachResp.Conn, stdin); err != nil {
		return fmt.Errorf("stream stdin: %w", err)
	}
	return attachResp.CloseWrite()
}

func waitContainer(ctx context.Context, cli *client.Client, cid string) (int, error) {
	waitC, errC := cli.ContainerWait(ctx, cid, container.WaitConditionNotRunning)
	select {
	case err := <-errC:
		return 1, fmt.Errorf("wait: %w", err)
	case status := <-waitC:
		return int(status.StatusCode), nil
	}
}

func containerLogs(ctx context.Context, cli *client.Client, cid string) (stdout, stderr string, err error) {
	out, err := cli.ContainerLogs(ctx, cid, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", "", fmt.Errorf("logs: %w", err)
	}
	defer out.Close()
	var stdoutBuf, stderrBuf strings.Builder
	_, err = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, out)
	if err != nil {
		return "", "", fmt.Errorf("read logs: %w", err)
	}
	return stdoutBuf.String(), stderrBuf.String(), nil
}
