package alerts

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
