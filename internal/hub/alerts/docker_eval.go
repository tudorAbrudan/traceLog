package alerts

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/models"
)

// EvaluateDocker runs docker_mem_pct / docker_cpu_percent rules against one scrape batch (same server_id).
func (e *Engine) EvaluateDocker(ctx context.Context, serverID string, batch []models.DockerMetrics) {
	if len(batch) == 0 || serverID == "" {
		return
	}

	latest := latestDockerSamplePerContainer(batch)

	e.mu.Lock()
	defer e.mu.Unlock()

	for _, rule := range e.rules {
		if !rule.Enabled || !IsDockerResourceMetric(rule.Metric) {
			continue
		}
		if rule.ServerID != "" && rule.ServerID != serverID {
			continue
		}

		for _, m := range latest {
			name := strings.TrimSpace(m.ContainerName)
			if name == "" {
				name = strings.TrimSpace(m.ContainerID)
			}
			if name == "" {
				continue
			}
			if !dockerContainerNameMatches(rule.DockerContainer, name) {
				continue
			}

			value := dockerResourceMetricValue(rule.Metric, &m)
			triggered := checkThreshold(value, rule.Operator, rule.Threshold)
			vk := rule.ID + "\x1e" + name

			if triggered {
				if _, ok := e.violations[vk]; !ok {
					e.violations[vk] = time.Now()
				}
				violationStart := e.violations[vk]
				if time.Since(violationStart) >= time.Duration(rule.DurationS)*time.Second {
					n := name
					v := value
					e.doFireUnlocked(ctx, rule, n, func() *Alert {
						return &Alert{
							RuleID:          rule.ID,
							ServerID:        serverID,
							OriginServerID:  serverID,
							Metric:          rule.Metric,
							Value:           v,
							Threshold:       rule.Threshold,
							FiredAt:         time.Now().UTC(),
							DockerContainer: n,
							Message: fmt.Sprintf("%s container %q is %.1f (threshold: %s %.1f)",
								rule.Metric, n, v, rule.Operator, rule.Threshold),
						}
					}, "Docker metric alert fired", "rule", rule.ID, "metric", rule.Metric, "container", n, "value", v)
				}
			} else {
				delete(e.violations, vk)
			}
		}
	}
}

func dockerContainerNameMatches(pattern, containerName string) bool {
	p := strings.TrimSpace(strings.ToLower(pattern))
	if p == "" {
		return true
	}
	return strings.Contains(strings.ToLower(containerName), p)
}

func dockerResourceMetricValue(metric string, m *models.DockerMetrics) float64 {
	switch metric {
	case "docker_mem_pct":
		if m.MemLimit == 0 {
			return 0
		}
		return float64(m.MemUsed) / float64(m.MemLimit) * 100
	case "docker_cpu_percent":
		return m.CPUPercent
	default:
		return 0
	}
}

// latestDockerSamplePerContainer keeps the newest row per container (id preferred, else name).
func latestDockerSamplePerContainer(batch []models.DockerMetrics) []models.DockerMetrics {
	best := make(map[string]models.DockerMetrics)
	for _, m := range batch {
		if m.ServerID == "" {
			continue
		}
		k := m.ContainerID
		if k == "" {
			k = m.ContainerName
		}
		if k == "" {
			continue
		}
		prev, ok := best[k]
		if !ok || m.Ts.After(prev.Ts) {
			best[k] = m
		}
	}
	out := make([]models.DockerMetrics, 0, len(best))
	for _, m := range best {
		out = append(out, m)
	}
	return out
}
