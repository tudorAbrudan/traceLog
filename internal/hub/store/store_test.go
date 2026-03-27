package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/models"
)

func setupTestStore(t *testing.T) (*Store, func()) {
	t.Helper()
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatal("failed to create store:", err)
	}
	return s, func() {
		s.Close()
		os.RemoveAll(dir)
	}
}

func TestMigration(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	var ver int
	err := s.db.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&ver)
	if err != nil {
		t.Fatal("failed to query version:", err)
	}
	if ver < 1 {
		t.Errorf("expected schema version >= 1, got %d", ver)
	}
}

func TestCreateUser(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	user, err := s.CreateUser(ctx, "testuser", "testpass123")
	if err != nil {
		t.Fatal("CreateUser failed:", err)
	}
	if user.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", user.Username)
	}
	if user.ID == "" {
		t.Error("expected non-empty user ID")
	}

	count, _ := s.UserCount(ctx)
	if count != 1 {
		t.Errorf("expected user count 1, got %d", count)
	}

	fetched, err := s.GetUserByUsername(ctx, "testuser")
	if err != nil {
		t.Fatal("GetUserByUsername failed:", err)
	}
	if fetched.ID != user.ID {
		t.Errorf("user IDs don't match: %q vs %q", fetched.ID, user.ID)
	}
}

func TestCreateUserDuplicate(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	_, err := s.CreateUser(ctx, "admin", "pass1234")
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateUser(ctx, "admin", "pass5678")
	if err == nil {
		t.Error("expected error for duplicate username, got nil")
	}
}

func TestSession(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	user, _ := s.CreateUser(ctx, "sessionuser", "pass1234")

	token := "test-token-123"
	expires := time.Now().Add(time.Hour)
	if err := s.CreateSession(ctx, token, user.ID, expires); err != nil {
		t.Fatal("CreateSession failed:", err)
	}

	fetched, err := s.GetUserBySession(ctx, token)
	if err != nil {
		t.Fatal("GetUserBySession failed:", err)
	}
	if fetched.ID != user.ID {
		t.Errorf("session returned wrong user: %q vs %q", fetched.ID, user.ID)
	}

	if err := s.DeleteSession(ctx, token); err != nil {
		t.Fatal("DeleteSession failed:", err)
	}
	_, err = s.GetUserBySession(ctx, token)
	if err == nil {
		t.Error("expected error after deleting session, got nil")
	}
}

func TestExpiredSession(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	user, _ := s.CreateUser(ctx, "expuser", "pass1234")
	token := "expired-token"
	expires := time.Now().Add(-time.Hour)
	if err := s.CreateSession(ctx, token, user.ID, expires); err != nil {
		t.Fatal(err)
	}

	_, err := s.GetUserBySession(ctx, token)
	if err == nil {
		t.Error("expected error for expired session, got nil")
	}
}

func TestServerCRUD(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	srv, err := s.CreateServer(ctx, "test-server", "192.168.1.1")
	if err != nil {
		t.Fatal("CreateServer failed:", err)
	}
	if srv.Name != "test-server" {
		t.Errorf("expected name 'test-server', got %q", srv.Name)
	}
	if srv.APIKey == "" {
		t.Error("expected non-empty API key")
	}

	fetched, err := s.GetServer(ctx, srv.ID)
	if err != nil {
		t.Fatal("GetServer failed:", err)
	}
	if fetched.Name != "test-server" {
		t.Errorf("fetched wrong server name: %q", fetched.Name)
	}

	byKey, err := s.GetServerByAPIKey(ctx, srv.APIKey)
	if err != nil {
		t.Fatal("GetServerByAPIKey failed:", err)
	}
	if byKey.ID != srv.ID {
		t.Error("GetServerByAPIKey returned wrong server")
	}

	servers, err := s.ListServers(ctx)
	if err != nil {
		t.Fatal("ListServers failed:", err)
	}
	if len(servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(servers))
	}

	if err := s.DeleteServer(ctx, srv.ID); err != nil {
		t.Fatal("DeleteServer failed:", err)
	}
	servers, _ = s.ListServers(ctx)
	if len(servers) != 0 {
		t.Errorf("expected 0 servers after delete, got %d", len(servers))
	}
}

func TestServerUpdateAndLogSourceLookup(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	srv, err := s.CreateServer(ctx, "a", "localhost")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateServer(ctx, srv.ID, "renamed", "10.0.0.1", "prod note"); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetServer(ctx, srv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "renamed" || got.Host != "10.0.0.1" || got.Notes != "prod note" {
		t.Fatalf("update mismatch: %+v", got)
	}

	err = s.CreateLogSource(ctx, &LogSourceRecord{
		ServerID: srv.ID,
		Name:     "auth",
		Type:     "file",
		Path:     "/var/log/auth.log",
		Format:   "plain",
		Enabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	path, container, typ, ok := s.LookupLogSourceByName(ctx, srv.ID, "auth")
	if !ok || path != "/var/log/auth.log" || container != "" || typ != "file" {
		t.Fatalf("lookup: ok=%v path=%q container=%q type=%q", ok, path, container, typ)
	}
	_, _, _, ok = s.LookupLogSourceByName(ctx, srv.ID, "missing")
	if ok {
		t.Fatal("expected miss")
	}
}

func TestMetricsInsertAndQuery(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	srv, _ := s.CreateServer(ctx, "metrics-server", "")

	now := time.Now()
	m := &models.SystemMetrics{
		ServerID:   srv.ID,
		Ts:         now,
		CPUPercent: 45.2,
		MemUsed:    1024 * 1024 * 512,
		MemTotal:   1024 * 1024 * 1024,
		DiskUsed:   50 * 1024 * 1024 * 1024,
		DiskTotal:  100 * 1024 * 1024 * 1024,
		Load1:      1.5,
		Load5:      2.0,
		Load15:     1.8,
	}

	if err := s.InsertMetrics(ctx, m); err != nil {
		t.Fatal("InsertMetrics failed:", err)
	}

	results, err := s.QueryMetrics(ctx, srv.ID, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatal("QueryMetrics failed:", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(results))
	}
	if results[0].CPUPercent != 45.2 {
		t.Errorf("expected CPU 45.2, got %f", results[0].CPUPercent)
	}
}

func TestSettings(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	val, err := s.GetSetting(ctx, "retention_days")
	if err != nil {
		t.Fatal("GetSetting failed:", err)
	}
	if val != "30" {
		t.Errorf("expected default retention_days='30', got %q", val)
	}

	if err := s.SetSetting(ctx, "retention_days", "14"); err != nil {
		t.Fatal("SetSetting failed:", err)
	}
	val, _ = s.GetSetting(ctx, "retention_days")
	if val != "14" {
		t.Errorf("expected retention_days='14', got %q", val)
	}
}

func TestBackup(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	if _, err := s.CreateUser(ctx, "backup-test", "pass1234"); err != nil {
		t.Fatal(err)
	}

	backupPath := filepath.Join(t.TempDir(), "test-backup.db")
	if err := s.Backup(ctx, backupPath); err != nil {
		t.Fatal("Backup failed:", err)
	}

	info, err := os.Stat(backupPath)
	if err != nil {
		t.Fatal("Backup file not found:", err)
	}
	if info.Size() == 0 {
		t.Error("Backup file is empty")
	}
}

func TestEnsureLocalServer(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	srv1, err := s.EnsureLocalServer(ctx)
	if err != nil {
		t.Fatal("EnsureLocalServer failed:", err)
	}
	if srv1.ID == "" {
		t.Error("expected non-empty server ID")
	}

	srv2, err := s.EnsureLocalServer(ctx)
	if err != nil {
		t.Fatal("second EnsureLocalServer failed:", err)
	}
	if srv2.ID != srv1.ID {
		t.Error("EnsureLocalServer should return the same server on second call")
	}
}

func TestLogInsertAndQuery(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	srv, _ := s.CreateServer(ctx, "log-server", "")
	entry := &models.LogEntry{
		ServerID: srv.ID,
		Ts:       time.Now(),
		Source:   "test.log",
		Level:    "error",
		Message:  "something went wrong",
	}
	if err := s.InsertLog(ctx, entry); err != nil {
		t.Fatal("InsertLog failed:", err)
	}

	logs, err := s.QueryLogs(ctx, srv.ID, LogQueryOpts{Level: "error", Limit: 100})
	if err != nil {
		t.Fatal("QueryLogs failed:", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].Message != "something went wrong" {
		t.Errorf("wrong log message: %q", logs[0].Message)
	}
}

func TestQueryLogsSeverityMin(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	srv, _ := s.CreateServer(ctx, "sev", "")
	base := time.Now().UTC()
	for i, row := range []struct {
		level, msg string
	}{
		{"critical", "c"},
		{"error", "e"},
		{"warn", "w"},
		{"info", "i"},
		{"debug", "d"},
	} {
		if err := s.InsertLog(ctx, &models.LogEntry{
			ServerID: srv.ID,
			Ts:       base.Add(time.Duration(i) * time.Millisecond),
			Source:   "t",
			Level:    row.level,
			Message:  row.msg,
		}); err != nil {
			t.Fatal(err)
		}
	}

	q := func(min string) int {
		logs, err := s.QueryLogs(ctx, srv.ID, LogQueryOpts{SeverityMin: min, Limit: 20})
		if err != nil {
			t.Fatal(err)
		}
		return len(logs)
	}
	if n := q("error"); n != 2 {
		t.Fatalf("severity_min=error want 2 logs, got %d", n)
	}
	if n := q("warn"); n != 3 {
		t.Fatalf("severity_min=warn want 3 logs, got %d", n)
	}
	if n := q("info"); n != 4 {
		t.Fatalf("severity_min=info want 4 logs, got %d", n)
	}
	if n := q("debug"); n != 5 {
		t.Fatalf("severity_min=debug want 5 logs, got %d", n)
	}
}

func TestDeleteIngestedLogs(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	srv, _ := s.CreateServer(ctx, "p", "")
	old := time.Now().Add(-48 * time.Hour)
	newTs := time.Now().Add(-1 * time.Hour)
	for _, ts := range []time.Time{old, newTs} {
		if err := s.InsertLog(ctx, &models.LogEntry{
			ServerID: srv.ID,
			Ts:       ts,
			Source:   "app",
			Level:    "info",
			Message:  "line",
		}); err != nil {
			t.Fatal(err)
		}
	}

	n, err := s.DeleteIngestedLogs(ctx, srv.ID, "", time.Now().Add(-36*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 deleted, got %d", n)
	}
	left, _ := s.QueryLogs(ctx, srv.ID, LogQueryOpts{Limit: 10})
	if len(left) != 1 {
		t.Fatalf("expected 1 log left, got %d", len(left))
	}

	n2, err := s.DeleteIngestedLogs(ctx, srv.ID, "", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if n2 != 1 {
		t.Fatalf("expected 1 deleted (all), got %d", n2)
	}
}

func TestListLogSourcesForAgentServer(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	srv, err := s.CreateServer(ctx, "agent-host", "")
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.CreateServer(ctx, "other", "")
	if err != nil {
		t.Fatal(err)
	}

	f := filepath.Join(t.TempDir(), "app.log")
	if err := os.WriteFile(f, []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	lsA := &LogSourceRecord{ServerID: srv.ID, Name: "a", Type: "file", Path: f, Format: "plain", Enabled: true}
	lsB := &LogSourceRecord{ServerID: other.ID, Name: "b", Type: "file", Path: f, Format: "plain", Enabled: true}
	if err := s.CreateLogSource(ctx, lsA); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateLogSource(ctx, lsB); err != nil {
		t.Fatal(err)
	}

	out, err := s.ListLogSourcesForAgentServer(ctx, srv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Name != "a" {
		t.Fatalf("want one source 'a', got %+v", out)
	}

	empty, err := s.ListLogSourcesForAgentServer(ctx, "")
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty server id: %+v err %v", empty, err)
	}
}

func TestQueryAccessSlowRequests(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()
	srv, err := s.CreateServer(ctx, "web", "")
	if err != nil {
		t.Fatal(err)
	}
	ts := time.Now().UTC().Add(-time.Minute)
	fast := &models.AccessLogEntry{
		ServerID: srv.ID, Ts: ts, Method: "GET", Path: "/fast", StatusCode: 200,
		DurationMs: 10, IP: "1.1.1.1", UserAgent: "Mozilla/5.0", BytesSent: 100,
	}
	if err := s.InsertAccessLog(ctx, fast); err != nil {
		t.Fatal(err)
	}
	slow := &models.AccessLogEntry{
		ServerID: srv.ID, Ts: ts, Method: "GET", Path: "/slow", StatusCode: 200,
		DurationMs: 2000, IP: "1.1.1.2", UserAgent: "Mozilla/5.0", BytesSent: 200,
	}
	if err := s.InsertAccessLog(ctx, slow); err != nil {
		t.Fatal(err)
	}
	border := &models.AccessLogEntry{
		ServerID: srv.ID, Ts: ts, Method: "GET", Path: "/border", StatusCode: 200,
		DurationMs: 500, IP: "1.1.1.3", UserAgent: "Mozilla/5.0", BytesSent: 150,
	}
	if err := s.InsertAccessLog(ctx, border); err != nil {
		t.Fatal(err)
	}
	out, err := s.QueryAccessSlowRequests(ctx, srv.ID, ts.Add(-time.Hour), 500, 10, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("want 2 rows with duration >= 500ms, got %d", len(out))
	}
	if out[0].DurationMs < out[1].DurationMs {
		t.Fatalf("want ORDER BY duration_ms DESC, got %.0f then %.0f", out[0].DurationMs, out[1].DurationMs)
	}
	if out[0].Path != "/slow" {
		t.Fatalf("expected slowest /slow first, got %q", out[0].Path)
	}
}

func TestAlertHistoryInsertAndList(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	if err := s.InsertAlertHistory(ctx, "rule1", "srv1", "fired", "hello"); err != nil {
		t.Fatal(err)
	}
	rows, err := s.ListAlertHistoryRecent(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].RuleID != "rule1" || rows[0].ServerID != "srv1" || rows[0].Message != "hello" {
		t.Fatalf("unexpected row: %+v", rows[0])
	}
}
