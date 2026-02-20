package util

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ErrConnectionRefused is returned when the connection error looks like "connection refused".
var ErrConnectionRefused = errors.New("connection refused")

// IsConnectionRefused reports whether err indicates a connection refused (e.g. nothing listening on the address).
func IsConnectionRefused(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "connection refused") ||
		strings.Contains(s, "connect: connection refused") ||
		strings.Contains(s, "dial tcp") && strings.Contains(s, "refused")
}

// IsLocalEndpoint reports whether the given endpoint URL is a local address
// (localhost or 127.0.0.1), where "start Docker" is a reasonable suggestion.
func IsLocalEndpoint(endpoint string) bool {
	u, err := url.Parse(endpoint)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// PromptAndStartDocker asks the user if they want to start Docker and, if yes,
// starts Docker (e.g. Docker Desktop). It reads a line from r (e.g. os.Stdin).
// Returns true if the user approved and StartDocker was run (caller may retry);
// false otherwise.
func PromptAndStartDocker(r io.Reader) bool {
	fmt.Println()
	fmt.Println("  Docker may not be running. The huskyCI API often runs in Docker.")
	fmt.Print("  Start Docker now? [y/N]: ")
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		return false
	}
	line := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if line != "y" && line != "yes" {
		return false
	}
	if err := StartDocker(); err != nil {
		fmt.Fprintf(os.Stderr, "  Failed to start Docker: %v\n", err)
		fmt.Fprintln(os.Stderr, "  Please start Docker manually and try again.")
		return false
	}
	fmt.Println("  Docker is starting. Wait a moment, then the CLI will retry.")
	return true
}

// StartDocker starts the Docker daemon / Docker Desktop for the current OS.
// On macOS it runs "open -a Docker". On Linux/Windows it returns a hint error
// so the user can start Docker manually.
func StartDocker() error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-a", "Docker").Run()
	case "linux":
		// systemctl start docker often requires root; suggest manual start
		cmd := exec.Command("systemctl", "--user", "start", "docker")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("could not start Docker: %w (try: systemctl start docker, or start Docker Desktop)", err)
		}
		return nil
	case "windows":
		// Docker Desktop on Windows can be started via COM or by running the executable
		return fmt.Errorf("please start Docker Desktop manually from the Start menu")
	default:
		return fmt.Errorf("please start Docker manually for %s", runtime.GOOS)
	}
}
