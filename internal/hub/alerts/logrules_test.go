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
