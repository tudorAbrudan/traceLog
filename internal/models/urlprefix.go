package models

import "strings"

// NormalizeURLPathPrefix returns "" for root, or a path like "/tracelog" (leading slash, no trailing slash).
func NormalizeURLPathPrefix(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "/" {
		return ""
	}
	if !strings.HasPrefix(s, "/") {
		s = "/" + s
	}
	return strings.TrimSuffix(s, "/")
}
