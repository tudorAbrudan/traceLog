//go:build linux

package collector

import (
	"os"
	"strconv"
	"strings"
)

// processInDockerCgroup skips processes that appear to run inside a container (host-visible PIDs in docker/containerd/k8s cgroups).
func processInDockerCgroup(pid int32) bool {
	if pid <= 0 {
		return false
	}
	b, err := os.ReadFile("/proc/" + strconv.FormatInt(int64(pid), 10) + "/cgroup")
	if err != nil {
		return false
	}
	s := string(b)
	return strings.Contains(s, "docker") || strings.Contains(s, "containerd") || strings.Contains(s, "kubepods")
}
