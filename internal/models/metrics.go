package models

import "time"

type SystemMetrics struct {
	ServerID string    `json:"server_id"`
	Ts       time.Time `json:"ts"`

	CPUPercent    float64 `json:"cpu_percent"`
	MemUsed       uint64  `json:"mem_used"`
	MemTotal      uint64  `json:"mem_total"`
	SwapUsed      uint64  `json:"swap_used"`
	SwapTotal     uint64  `json:"swap_total"`
	DiskUsed      uint64  `json:"disk_used"`
	DiskTotal     uint64  `json:"disk_total"`
	DiskReadBytes uint64  `json:"disk_read_bytes"`
	DiskWriteBytes uint64 `json:"disk_write_bytes"`
	NetRxBytes    uint64  `json:"net_rx_bytes"`
	NetTxBytes    uint64  `json:"net_tx_bytes"`
	Load1         float64 `json:"load_1"`
	Load5         float64 `json:"load_5"`
	Load15        float64 `json:"load_15"`
	Uptime        uint64  `json:"uptime"`
}

type DockerMetrics struct {
	ServerID      string    `json:"server_id"`
	Ts            time.Time `json:"ts"`
	ContainerID   string    `json:"container_id"`
	ContainerName string    `json:"container_name"`
	Image         string    `json:"image"`
	Status        string    `json:"status"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemUsed       uint64    `json:"mem_used"`
	MemLimit      uint64    `json:"mem_limit"`
	NetRxBytes    uint64    `json:"net_rx_bytes"`
	NetTxBytes    uint64    `json:"net_tx_bytes"`
}

type LogEntry struct {
	ServerID string    `json:"server_id"`
	Ts       time.Time `json:"ts"`
	Source   string    `json:"source"`
	Level    string    `json:"level"`
	Message  string    `json:"message"`
	Metadata string    `json:"metadata,omitempty"`
}

type AccessLogEntry struct {
	ServerID   string    `json:"server_id"`
	Ts         time.Time `json:"ts"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	StatusCode int       `json:"status_code"`
	DurationMs float64   `json:"duration_ms"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
	BytesSent  uint64    `json:"bytes_sent"`
}
