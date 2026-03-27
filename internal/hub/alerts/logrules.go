package alerts

// AlertNotificationKind returns a short label and a plain-language hint for email/webhook bodies.
func AlertNotificationKind(metric string) (kindTitle, contextHint string) {
	switch {
	case IsLogMetricRule(metric):
		return "Ingested log line (Log Sources)",
			"This comes from a line TraceLog stored from a configured Log Source on the agent host (file path or similar). It is not an Uptime monitor URL check. Docker “Load logs” in the UI does not trigger this unless those lines are ingested via a file/source."
	case IsDockerResourceMetric(metric):
		return "Docker container metric (docker stats)",
			"Evaluated from docker stats on the agent. When a container name appears below, it is the container that exceeded the threshold."
	default:
		return "Host system metric",
			"Evaluated from CPU, memory, disk, or load averages reported by the agent for this server."
	}
}

// IsDockerResourceMetric is true for rules evaluated on per-container docker stats (not host system metrics).
func IsDockerResourceMetric(metric string) bool {
	switch metric {
	case "docker_mem_pct", "docker_cpu_percent":
		return true
	default:
		return false
	}
}

// IsLogMetricRule is true for rules evaluated on ingested log lines (not system metrics).
func IsLogMetricRule(metric string) bool {
	switch metric {
	case "log_critical", "log_error", "log_warn":
		return true
	default:
		return false
	}
}

// LogLevelMatches returns whether an ingested log entry level should trigger this log rule.
func LogLevelMatches(ruleMetric, entryLevel string) bool {
	switch ruleMetric {
	case "log_critical":
		return entryLevel == "critical"
	case "log_error":
		return entryLevel == "critical" || entryLevel == "error"
	case "log_warn":
		return entryLevel == "critical" || entryLevel == "error" || entryLevel == "warn" || entryLevel == "deprecated"
	default:
		return false
	}
}
