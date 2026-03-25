package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func New(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		return nil, fmt.Errorf("create data dir %s: %w", dataDir, err)
	}

	dbPath := filepath.Join(dataDir, "tracelog.db")
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	slog.Info("Database initialized", "path", dbPath)
	return s, nil
}

func (s *Store) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER PRIMARY KEY
		);
	`)
	if err != nil {
		return fmt.Errorf("create schema_version table: %w", err)
	}

	var currentVersion int
	err = s.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("query schema version: %w", err)
	}

	migrations := []string{
		migration001,
	}

	for i := currentVersion; i < len(migrations); i++ {
		slog.Info("Running migration", "version", i+1)
		if _, err := s.db.Exec(migrations[i]); err != nil {
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
		if _, err := s.db.Exec("INSERT INTO schema_version (version) VALUES (?)", i+1); err != nil {
			return fmt.Errorf("update schema version to %d: %w", i+1, err)
		}
	}

	return nil
}

const migration001 = `
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    token TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    expires_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS servers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    host TEXT,
    api_key TEXT UNIQUE NOT NULL,
    status TEXT DEFAULT 'pending',
    last_seen_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS metrics (
    server_id TEXT NOT NULL,
    ts DATETIME NOT NULL,
    cpu_percent REAL,
    mem_used INTEGER, mem_total INTEGER,
    swap_used INTEGER, swap_total INTEGER,
    disk_used INTEGER, disk_total INTEGER,
    disk_read_bytes INTEGER, disk_write_bytes INTEGER,
    net_rx_bytes INTEGER, net_tx_bytes INTEGER,
    load_1 REAL, load_5 REAL, load_15 REAL,
    uptime INTEGER,
    PRIMARY KEY (server_id, ts)
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS docker_metrics (
    server_id TEXT NOT NULL,
    ts DATETIME NOT NULL,
    container_id TEXT NOT NULL,
    container_name TEXT,
    image TEXT,
    status TEXT,
    cpu_percent REAL,
    mem_used INTEGER, mem_limit INTEGER,
    net_rx_bytes INTEGER, net_tx_bytes INTEGER,
    PRIMARY KEY (server_id, ts, container_id)
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id TEXT NOT NULL,
    ts DATETIME NOT NULL,
    source TEXT,
    level TEXT,
    message TEXT,
    metadata TEXT
);
CREATE INDEX IF NOT EXISTS idx_logs_server_ts ON logs(server_id, ts);
CREATE INDEX IF NOT EXISTS idx_logs_level ON logs(level);

CREATE TABLE IF NOT EXISTS access_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id TEXT NOT NULL,
    ts DATETIME NOT NULL,
    method TEXT,
    path TEXT,
    status_code INTEGER,
    duration_ms REAL,
    ip TEXT,
    user_agent TEXT,
    bytes_sent INTEGER
);
CREATE INDEX IF NOT EXISTS idx_access_logs_server_ts ON access_logs(server_id, ts);
CREATE INDEX IF NOT EXISTS idx_access_logs_status ON access_logs(status_code);

CREATE TABLE IF NOT EXISTS uptime_checks (
    id TEXT PRIMARY KEY,
    name TEXT,
    url TEXT NOT NULL,
    method TEXT DEFAULT 'GET',
    interval_seconds INTEGER DEFAULT 60,
    timeout_seconds INTEGER DEFAULT 10,
    expected_status INTEGER DEFAULT 200,
    enabled INTEGER DEFAULT 1
);

CREATE TABLE IF NOT EXISTS uptime_results (
    check_id TEXT NOT NULL,
    ts DATETIME NOT NULL,
    status_code INTEGER,
    duration_ms REAL,
    error TEXT,
    PRIMARY KEY (check_id, ts)
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS alert_rules (
    id TEXT PRIMARY KEY,
    server_id TEXT,
    metric TEXT NOT NULL,
    operator TEXT NOT NULL,
    threshold REAL NOT NULL,
    duration_seconds INTEGER,
    cooldown_minutes INTEGER DEFAULT 30,
    notify_channels TEXT,
    state TEXT DEFAULT 'idle',
    last_triggered_at DATETIME,
    enabled INTEGER DEFAULT 1
);

CREATE TABLE IF NOT EXISTS alert_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id TEXT NOT NULL,
    server_id TEXT,
    state TEXT NOT NULL,
    message TEXT,
    ts DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS notification_channels (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    config TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS log_sources (
    id TEXT PRIMARY KEY,
    server_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    path TEXT,
    container TEXT,
    format TEXT,
    enabled INTEGER DEFAULT 1
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT
);

INSERT OR IGNORE INTO settings (key, value) VALUES ('retention_days', '30');
INSERT OR IGNORE INTO settings (key, value) VALUES ('collection_interval', '10');
`

func (s *Store) StartRetentionWorker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	s.runRetention()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runRetention()
		}
	}
}

func (s *Store) runRetention() {
	var days int
	err := s.db.QueryRow("SELECT CAST(value AS INTEGER) FROM settings WHERE key = 'retention_days'").Scan(&days)
	if err != nil || days <= 0 {
		days = 30
	}

	cutoff := time.Now().AddDate(0, 0, -days).Format(time.RFC3339)
	tables := []string{"metrics", "docker_metrics", "logs", "access_logs", "uptime_results", "alert_history"}

	for _, table := range tables {
		tsCol := "ts"
		result, err := s.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s < ?", table, tsCol), cutoff)
		if err != nil {
			slog.Error("Retention cleanup failed", "table", table, "error", err)
			continue
		}
		if rows, _ := result.RowsAffected(); rows > 0 {
			slog.Info("Retention cleanup", "table", table, "deleted", rows)
		}
	}
}
