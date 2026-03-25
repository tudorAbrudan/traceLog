package alerts

import (
	"context"
	"fmt"
	"log/slog"
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
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, rule := range e.rules {
		if !rule.Enabled || rule.ServerID != serverID {
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
				e.fireAlert(ctx, rule, value)
			}
		} else {
			delete(e.violations, rule.ID)
		}
	}
}

func (e *Engine) fireAlert(ctx context.Context, rule *Rule, value float64) {
	lastFired, ok := e.lastFired[rule.ID]
	cooldown := time.Duration(rule.CooldownS) * time.Second
	if cooldown == 0 {
		cooldown = 5 * time.Minute
	}

	if ok && time.Since(lastFired) < cooldown {
		return
	}

	alert := &Alert{
		RuleID:    rule.ID,
		ServerID:  rule.ServerID,
		Metric:    rule.Metric,
		Value:     value,
		Threshold: rule.Threshold,
		FiredAt:   time.Now().UTC(),
		Message:   fmt.Sprintf("%s is %.1f (threshold: %s %.1f)", rule.Metric, value, rule.Operator, rule.Threshold),
	}

	slog.Warn("Alert fired", "rule", rule.ID, "metric", rule.Metric, "value", value, "threshold", rule.Threshold)

	e.lastFired[rule.ID] = time.Now()

	if e.notifyFunc != nil && rule.ChannelID != "" {
		go func() {
			if err := e.notifyFunc(ctx, rule.ChannelID, alert); err != nil {
				slog.Error("Failed to send alert notification", "error", err, "channel", rule.ChannelID)
			}
		}()
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
