package models

import "testing"

func TestNormalizeURLPathPrefix(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"/", ""},
		{"tracelog", "/tracelog"},
		{"/tracelog", "/tracelog"},
		{"/tracelog/", "/tracelog"},
		{"  /foo/bar/  ", "/foo/bar"},
	}
	for _, tc := range tests {
		if got := NormalizeURLPathPrefix(tc.in); got != tc.want {
			t.Errorf("NormalizeURLPathPrefix(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
