package uptime

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type Check struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Interval int    `json:"interval_seconds"`
	Timeout  int    `json:"timeout_seconds"`
	Enabled  bool   `json:"enabled"`
}

type Result struct {
	CheckID      string    `json:"check_id"`
	Ts           time.Time `json:"ts"`
	StatusCode   int       `json:"status_code"`
	ResponseTime int64     `json:"response_time_ms"`
	Up           bool      `json:"up"`
	Error        string    `json:"error,omitempty"`
}

type Callback func(result *Result)

type Checker struct {
	mu      sync.RWMutex
	checks  map[string]*Check
	cancels map[string]context.CancelFunc
	cb      Callback
	client  *http.Client
}

func NewChecker(cb Callback) *Checker {
	return &Checker{
		checks:  make(map[string]*Check),
		cancels: make(map[string]context.CancelFunc),
		cb:      cb,
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (c *Checker) AddCheck(check *Check) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if cancel, ok := c.cancels[check.ID]; ok {
		cancel()
	}

	c.checks[check.ID] = check
	if check.Enabled {
		ctx, cancel := context.WithCancel(context.Background())
		c.cancels[check.ID] = cancel
		go c.runCheck(ctx, check)
	}
}

func (c *Checker) RemoveCheck(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if cancel, ok := c.cancels[id]; ok {
		cancel()
		delete(c.cancels, id)
	}
	delete(c.checks, id)
}

func (c *Checker) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, cancel := range c.cancels {
		cancel()
		delete(c.cancels, id)
	}
}

func (c *Checker) runCheck(ctx context.Context, check *Check) {
	interval := time.Duration(check.Interval) * time.Second
	if interval < 10*time.Second {
		interval = 30 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	c.performCheck(check)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.performCheck(check)
		}
	}
}

func (c *Checker) performCheck(check *Check) {
	timeout := time.Duration(check.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", check.URL, nil)
	if err != nil {
		c.cb(&Result{
			CheckID: check.ID,
			Ts:      time.Now().UTC(),
			Up:      false,
			Error:   err.Error(),
		})
		return
	}
	req.Header.Set("User-Agent", "TraceLog/1.0 Uptime Monitor")

	start := time.Now()
	resp, err := c.client.Do(req)
	elapsed := time.Since(start).Milliseconds()

	result := &Result{
		CheckID:      check.ID,
		Ts:           time.Now().UTC(),
		ResponseTime: elapsed,
	}

	if err != nil {
		result.Up = false
		result.Error = err.Error()
	} else {
		resp.Body.Close()
		result.StatusCode = resp.StatusCode
		result.Up = resp.StatusCode >= 200 && resp.StatusCode < 400
	}

	slog.Debug("Uptime check", "name", check.Name, "url", check.URL, "up", result.Up, "ms", result.ResponseTime)
	c.cb(result)
}
