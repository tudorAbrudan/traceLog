package ipmatch

import "testing"

func TestMatch(t *testing.T) {
	rules := []string{"192.168.1.10", "10.0.0.0/8", "2001:db8::/32"}
	tests := []struct {
		ip   string
		want bool
	}{
		{"192.168.1.10", true},
		{"192.168.1.11", false},
		{"10.5.5.5", true},
		{"8.8.8.8", false},
	}
	for _, tt := range tests {
		if got := Match(tt.ip, rules); got != tt.want {
			t.Errorf("Match(%q) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}
