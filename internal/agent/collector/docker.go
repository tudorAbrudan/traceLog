package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/models"
)

type DockerCollector struct{}

func NewDockerCollector() *DockerCollector {
	return &DockerCollector{}
}

type dockerStats struct {
	ID       string `json:"ID"`
	Name     string `json:"Name"`
	CPUPerc  string `json:"CPUPerc"`
	MemUsage string `json:"MemUsage"`
	NetIO    string `json:"NetIO"`
}

func (d *DockerCollector) Collect(ctx context.Context) ([]models.DockerMetrics, error) {
	out, err := exec.CommandContext(ctx, "docker", "stats", "--no-stream", "--format", `{"ID":"{{.ID}}","Name":"{{.Name}}","CPUPerc":"{{.CPUPerc}}","MemUsage":"{{.MemUsage}}","NetIO":"{{.NetIO}}"}`).Output()
	if err != nil {
		return nil, fmt.Errorf("docker stats: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var result []models.DockerMetrics
	now := time.Now().UTC()

	for _, line := range lines {
		if line == "" {
			continue
		}
		var ds dockerStats
		if err := json.Unmarshal([]byte(line), &ds); err != nil {
			continue
		}

		m := models.DockerMetrics{
			Ts:            now,
			ContainerID:   ds.ID,
			ContainerName: strings.TrimPrefix(ds.Name, "/"),
			CPUPercent:    parsePercent(ds.CPUPerc),
		}

		memParts := strings.Split(ds.MemUsage, " / ")
		if len(memParts) == 2 {
			m.MemUsed = parseSize(strings.TrimSpace(memParts[0]))
			m.MemLimit = parseSize(strings.TrimSpace(memParts[1]))
		}

		netParts := strings.Split(ds.NetIO, " / ")
		if len(netParts) == 2 {
			m.NetRxBytes = parseSize(strings.TrimSpace(netParts[0]))
			m.NetTxBytes = parseSize(strings.TrimSpace(netParts[1]))
		}

		result = append(result, m)
	}

	return result, nil
}

func parsePercent(s string) float64 {
	s = strings.TrimSuffix(s, "%")
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func parseSize(s string) uint64 {
	s = strings.TrimSpace(s)
	multiplier := uint64(1)

	switch {
	case strings.HasSuffix(s, "GiB"):
		multiplier = 1 << 30
		s = strings.TrimSuffix(s, "GiB")
	case strings.HasSuffix(s, "MiB"):
		multiplier = 1 << 20
		s = strings.TrimSuffix(s, "MiB")
	case strings.HasSuffix(s, "KiB"):
		multiplier = 1 << 10
		s = strings.TrimSuffix(s, "KiB")
	case strings.HasSuffix(s, "GB"):
		multiplier = 1e9
		s = strings.TrimSuffix(s, "GB")
	case strings.HasSuffix(s, "MB"):
		multiplier = 1e6
		s = strings.TrimSuffix(s, "MB")
	case strings.HasSuffix(s, "KB"):
		multiplier = 1e3
		s = strings.TrimSuffix(s, "KB")
	case strings.HasSuffix(s, "kB"):
		multiplier = 1e3
		s = strings.TrimSuffix(s, "kB")
	case strings.HasSuffix(s, "B"):
		s = strings.TrimSuffix(s, "B")
	}

	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return uint64(v * float64(multiplier))
}
