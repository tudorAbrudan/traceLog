//go:build !linux

package collector

func processInDockerCgroup(pid int32) bool {
	return false
}
