package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// RemoteRunner implements Runner by sending HTTP requests to a runner service
// that runs containers on the host (e.g. with Docker socket mounted).
type RemoteRunner struct {
	baseURL    string
	httpClient *http.Client
}

// NewRemoteRunner returns a Runner that uses the runner service at baseURL
// (e.g. "http://runner-service:8090"). baseURL must not have a trailing slash.
func NewRemoteRunner(baseURL string) *RemoteRunner {
	return &RemoteRunner{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

// remoteRunRequest is the JSON body for POST /run (when not using multipart).
type remoteRunRequest struct {
	Image           string `json:"image"`
	Cmd             string `json:"cmd"`
	VolumePath      string `json:"volumePath"`
	TimeoutSeconds  int    `json:"timeoutSeconds"`
	ReadWriteVolume bool   `json:"readWriteVolume"`
}

// remoteRunResponse is the JSON response from POST /run.
type remoteRunResponse struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
	Error    string `json:"error,omitempty"`
}

// Run sends the run request to the remote runner service. If req.Stdin is not nil,
// it is sent as a multipart form part "stdin"; otherwise the body is JSON only.
func (r *RemoteRunner) Run(ctx context.Context, req RunRequest) (RunResult, error) {
	if req.Stdin != nil {
		return r.runWithStdin(ctx, req)
	}
	body := remoteRunRequest{
		Image:           req.Image,
		Cmd:             req.Cmd,
		VolumePath:      req.VolumePath,
		TimeoutSeconds:  req.TimeoutSeconds,
		ReadWriteVolume: req.ReadWriteVolume,
	}
	if body.TimeoutSeconds <= 0 {
		body.TimeoutSeconds = 300
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return RunResult{Err: err}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/run", bytes.NewReader(jsonBody))
	if err != nil {
		return RunResult{Err: err}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return RunResult{Err: err}, err
	}
	defer resp.Body.Close()
	return r.parseRunResponse(resp)
}

// runWithStdin sends multipart/form-data: part "request" (JSON) and part "stdin" (req.Stdin).
func (r *RemoteRunner) runWithStdin(ctx context.Context, req RunRequest) (RunResult, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	// part "request"
	reqJSON := remoteRunRequest{
		Image:           req.Image,
		Cmd:             req.Cmd,
		VolumePath:      req.VolumePath,
		TimeoutSeconds:  req.TimeoutSeconds,
		ReadWriteVolume: req.ReadWriteVolume,
	}
	if reqJSON.TimeoutSeconds <= 0 {
		reqJSON.TimeoutSeconds = 300
	}
	jsonBytes, err := json.Marshal(reqJSON)
	if err != nil {
		return RunResult{Err: err}, err
	}
	if err := mw.WriteField("request", string(jsonBytes)); err != nil {
		return RunResult{Err: err}, err
	}
	// part "stdin"
	stdinPart, err := mw.CreateFormFile("stdin", "stdin")
	if err != nil {
		return RunResult{Err: err}, err
	}
	if _, err := io.Copy(stdinPart, req.Stdin); err != nil {
		return RunResult{Err: err}, err
	}
	if err := mw.Close(); err != nil {
		return RunResult{Err: err}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/run", &buf)
	if err != nil {
		return RunResult{Err: err}, err
	}
	httpReq.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return RunResult{Err: err}, err
	}
	defer resp.Body.Close()
	return r.parseRunResponse(resp)
}

func (r *RemoteRunner) parseRunResponse(resp *http.Response) (RunResult, error) {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return RunResult{Err: fmt.Errorf("runner service returned %d: %s", resp.StatusCode, string(body))},
			fmt.Errorf("runner service: %d", resp.StatusCode)
	}
	var out remoteRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return RunResult{Err: err}, err
	}
	result := RunResult{Stdout: out.Stdout, Stderr: out.Stderr, ExitCode: out.ExitCode}
	if out.Error != "" {
		result.Err = fmt.Errorf("%s", out.Error)
		return result, result.Err
	}
	return result, nil
}

// EnsureImage is a no-op for RemoteRunner; the runner service ensures the image on its side.
func (r *RemoteRunner) EnsureImage(ctx context.Context, image string) error {
	return nil
}

// Health calls GET /health on the runner service.
func (r *RemoteRunner) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.baseURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("runner health returned %d", resp.StatusCode)
	}
	return nil
}
