package alerts

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/models"
)

type Rule struct {
	ID         string  `json:"id"`
	ServerID   string  `json:"server_id"`
	Metric     string  `json:"metric"`
	Operator   string  `json:"operator"`
	Threshold  float64 `json:"threshold"`
	DurationS  int     `json:"duration_seconds"`
	CooldownS  int     `json:"cooldown_seconds"`
	ChannelID  string  `json:"channel_id"`
	Enabled    bool    `json:"enabled"`
}

type Alert struct {
	RuleID    string    `json:"rule_id"`
	ServerID  string    `json:"server_id"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	FiredAt   time.Time `json:"fired_at"`
	Message   string    `json:"message"`
}

type NotifyFunc func(ctx context.Context, channelID string, alert *Alert) error

// LogSilenceFn returns true if notifications for this log line and rule should be suppressed.
type LogSilenceFn func(serverID, level, message, ruleMetric string) bool

type Engine struct {
	mu          sync.RWMutex
	rules       map[string]*Rule
	lastFired   map[string]time.Time
	violations  map[string]time.Time
	notifyFunc  NotifyFunc
}

func NewEngine(notify NotifyFunc) *Engine {
	return &Engine{
		rules:      make(map[string]*Rule),
		lastFired:  make(map[string]time.Time),
		violations: make(map[string]time.Time),
		notifyFunc: notify,
	}
}

func (e *Engine) AddRule(rule *Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules[rule.ID] = rule
}

func (e *Engine) RemoveRule(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.rules, id)
	delete(e.lastFired, id)
	delete(e.violations, id)
}

func (e *Engine) Evaluate(ctx context.Context, serverID string, metrics *models.SystemMetrics) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, rule := range e.rules {
		if !rule.Enabled || rule.ServerID != serverID {
			continue
		}
		if IsLogMetricRule(rule.Metric) {
			continue
		}

		value := extractMetricValue(rule.Metric, metrics)
		triggered := checkThreshold(value, rule.Operator, rule.Threshold)

		if triggered {
			if _, ok := e.violations[rule.ID]; !ok {
				e.violations[rule.ID] = time.Now()
			}
			violationStart := e.violations[rule.ID]
			if time.Since(violationStart) >= time.Duration(rule.DurationS)*time.Second {
				e.doFireUnlocked(ctx, rule, func() *Alert {
					return &Alert{
						RuleID:    rule.ID,
						ServerID:  rule.ServerID,
						Metric:    rule.Metric,
						Value:     value,
						Threshold: rule.Threshold,
						FiredAt:   time.Now().UTC(),
						Message:   fmt.Sprintf("%s is %.1f (threshold: %s %.1f)", rule.Metric, value, rule.Operator, rule.Threshold),
					}
				}, "Metric alert fired", "rule", rule.ID, "metric", rule.Metric, "value", value, "threshold", rule.Threshold)
			}
		} else {
			delete(e.violations, rule.ID)
		}
	}
}

// EvaluateLog runs log-based rules when a line is stored (any source: files, apache, docker→ingested, etc.).
func (e *Engine) EvaluateLog(ctx context.Context, serverID, level, source, message string, silence LogSilenceFn) {
	var matches []*Rule
	e.mu.RLock()
	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}
		if rule.ServerID != "" && rule.ServerID != serverID {
			continue
		}
		if !IsLogMetricRule(rule.Metric) || !LogLevelMatches(rule.Metric, level) {
			continue
		}
		if silence != nil && silence(serverID, level, message, rule.Metric) {
			continue
		}
		r := rule
		matches = append(matches, r)
	}
	e.mu.RUnlock()

	for _, rule := range matches {
		e.fireLogAlert(ctx, rule, level, source, message)
	}
}

func (e *Engine) fireLogAlert(ctx context.Context, rule *Rule, level, source, msg string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	preview := strings.TrimSpace(msg)
	if len([]rune(preview)) > 400 {
		preview = string([]rune(preview)[:400]) + "…"
	}
	e.doFireUnlocked(ctx, rule, func() *Alert {
		return &Alert{
			RuleID:    rule.ID,
			ServerID:  rule.ServerID,
			Metric:    rule.Metric,
			Value:     0,
			Threshold: 0,
			FiredAt:   time.Now().UTC(),
			Message:   fmt.Sprintf("Ingested log level=%s source=%q: %s", level, source, preview),
		}
	}, "Log alert fired", "rule", rule.ID, "metric", rule.Metric, "level", level, "source", source)
}

// doFireUnlocked requires e.mu locked (Evaluate path) or is called from fireLogAlert after Lock.
func (e *Engine) doFireUnlocked(ctx context.Context, rule *Rule, buildAlert func() *Alert, warnMsg string, warnKV ...any) {
	lastFired, ok := e.lastFired[rule.ID]
	cooldown := time.Duration(rule.CooldownS) * time.Second
	if cooldown == 0 {
		cooldown = 5 * time.Minute
	}
	if ok && time.Since(lastFired) < cooldown {
		return
	}

	alert := buildAlert()
	slog.Warn(warnMsg, warnKV...)

	e.lastFired[rule.ID] = time.Now()

	ch := rule.ChannelID
	if e.notifyFunc != nil && ch != "" {
		go func(a *Alert, channelID string) {
			if err := e.notifyFunc(ctx, channelID, a); err != nil {
				slog.Error("Failed to send alert notification", "error", err, "channel", channelID)
			}
		}(alert, ch)
	}
}

func extractMetricValue(metric string, m *models.SystemMetrics) float64 {
	switch metric {
	case "cpu_percent":
		return m.CPUPercent
	case "mem_percent":
		if m.MemTotal == 0 {
			return 0
		}
		return float64(m.MemUsed) / float64(m.MemTotal) * 100
	case "disk_percent":
		if m.DiskTotal == 0 {
			return 0
		}
		return float64(m.DiskUsed) / float64(m.DiskTotal) * 100
	case "load_1":
		return m.Load1
	case "load_5":
		return m.Load5
	case "load_15":
		return m.Load15
	default:
		return 0
	}
}

func checkThreshold(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	default:
		return value > threshold
	}
}
