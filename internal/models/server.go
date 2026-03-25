package models

import "time"

type Server struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Host       string    `json:"host,omitempty"`
	APIKey     string    `json:"api_key,omitempty"`
	Status     string    `json:"status"`
	LastSeenAt time.Time `json:"last_seen_at,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type AlertRule struct {
	ID              string    `json:"id"`
	ServerID        string    `json:"server_id,omitempty"`
	Metric          string    `json:"metric"`
	Operator        string    `json:"operator"`
	Threshold       float64   `json:"threshold"`
	DurationSeconds int       `json:"duration_seconds"`
	CooldownMinutes int       `json:"cooldown_minutes"`
	NotifyChannels  string    `json:"notify_channels"`
	State           string    `json:"state"`
	LastTriggeredAt time.Time `json:"last_triggered_at,omitempty"`
	Enabled         bool      `json:"enabled"`
}

type NotificationChannel struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Name   string `json:"name"`
	Config string `json:"config"`
}
