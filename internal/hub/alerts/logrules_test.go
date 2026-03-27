package alerts

import "testing"

func TestLogLevelMatches(t *testing.T) {
	if !LogLevelMatches("log_error", "error") {
		t.Fatal("log_error should match error")
	}
	if !LogLevelMatches("log_error", "critical") {
		t.Fatal("log_error should match critical")
	}
	if LogLevelMatches("log_error", "warn") {
		t.Fatal("log_error should not match warn")
	}
	if !LogLevelMatches("log_warn", "warn") {
		t.Fatal("log_warn should match warn")
	}
	if !LogLevelMatches("log_critical", "critical") {
		t.Fatal("log_critical should match critical")
	}
	if LogLevelMatches("log_critical", "error") {
		t.Fatal("log_critical should not match error alone")
	}
}

func TestAlertNotificationKind(t *testing.T) {
	k, h := AlertNotificationKind("log_warn")
	if k == "" || h == "" {
		t.Fatalf("expected non-empty kind and hint for log_warn, got %q / %q", k, h)
	}
	k, h = AlertNotificationKind("docker_mem_pct")
	if k == "" || h == "" {
		t.Fatalf("expected non-empty for docker, got %q / %q", k, h)
	}
	k, h = AlertNotificationKind("cpu_percent")
	if k == "" || h == "" {
		t.Fatalf("expected non-empty for host metric, got %q / %q", k, h)
	}
}
