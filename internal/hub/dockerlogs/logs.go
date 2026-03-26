package dockerlogs

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var safeContainer = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,199}$`)

// Fetch returns the last `tail` lines of docker logs for a container name or ID.
func Fetch(ctx context.Context, container string, tail int) (string, error) {
	container = strings.TrimSpace(container)
	if container == "" || !safeContainer.MatchString(container) {
		return "", fmt.Errorf("invalid container name or id")
	}
	if tail <= 0 {
		tail = 200
	}
	if tail > 10000 {
		tail = 10000
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	//nolint:gosec // container is validated by safeContainer before use.
	cmd := exec.CommandContext(ctx, "docker", "logs", "--tail", fmt.Sprintf("%d", tail), "--timestamps", container)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker logs: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
