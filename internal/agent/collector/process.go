package collector

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"
	"github.com/tudorAbrudan/tracelog/internal/models"
)

var defaultWatchNames = map[string]bool{
	"nginx": true, "apache2": true, "httpd": true,
	"php-fpm": true, "node": true, "python": true, "python3": true,
	"java": true, "go": true, "ruby": true, "postgres": true,
	"mysqld": true, "redis-server": true, "mongod": true,
	"pm2": true, "docker": true, "containerd": true,
	"gunicorn": true, "uvicorn": true,
}

type ProcessCollector struct {
	watchNames map[string]bool
	minCPU     float64
}

func NewProcessCollector() *ProcessCollector {
	return &ProcessCollector{
		watchNames: defaultWatchNames,
		minCPU:     1.0,
	}
}

func (c *ProcessCollector) Collect(ctx context.Context) ([]models.ProcessMetrics, error) {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var results []models.ProcessMetrics

	for _, p := range procs {
		if processInDockerCgroup(p.Pid) {
			continue
		}

		name, err := p.NameWithContext(ctx)
		if err != nil {
			continue
		}

		nameLower := strings.ToLower(name)
		watched := c.watchNames[nameLower]

		cpu, _ := p.CPUPercentWithContext(ctx)
		if !watched && cpu < c.minCPU {
			continue
		}

		memInfo, err := p.MemoryInfoWithContext(ctx)
		if err != nil {
			continue
		}

		status, _ := p.StatusWithContext(ctx)
		threads, _ := p.NumThreadsWithContext(ctx)
		cmdline, _ := p.CmdlineWithContext(ctx)

		if len(cmdline) > 256 {
			cmdline = cmdline[:256]
		}

		statusStr := "unknown"
		if len(status) > 0 {
			statusStr = status[0]
		}

		results = append(results, models.ProcessMetrics{
			Ts:      now,
			PID:     p.Pid,
			Name:    name,
			Cmdline: cmdline,
			Status:  statusStr,
			CPU:     cpu,
			MemRSS:  memInfo.RSS,
			MemVMS:  memInfo.VMS,
			Threads: threads,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].CPU > results[j].CPU
	})

	if len(results) > 50 {
		results = results[:50]
	}

	return results, nil
}
