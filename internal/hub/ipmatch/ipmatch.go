// Package ipmatch matches client IPs against a list of exact addresses or CIDR rules.
package ipmatch

import (
	"encoding/json"
	"net"
	"strings"
)

// Match reports whether ipStr matches any rule. Rules are exact IPs (v4/v6) or CIDRs like 10.0.0.0/8.
// Malformed rules are skipped.
func Match(ipStr string, rules []string) bool {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return false
	}
	for _, r := range rules {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		if strings.Contains(r, "/") {
			_, cidr, err := net.ParseCIDR(r)
			if err != nil {
				continue
			}
			if cidr.Contains(ip) {
				return true
			}
			continue
		}
		x := net.ParseIP(r)
		if x != nil && x.Equal(ip) {
			return true
		}
	}
	return false
}

// ParseJSONArray decodes a JSON array of strings (from settings).
func ParseJSONArray(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	var cleaned []string
	for _, s := range out {
		s = strings.TrimSpace(s)
		if s != "" {
			cleaned = append(cleaned, s)
		}
	}
	return cleaned
}

// ToJSONArray serializes rules for storage.
func ToJSONArray(rules []string) (string, error) {
	var cleaned []string
	for _, s := range rules {
		s = strings.TrimSpace(s)
		if s != "" {
			cleaned = append(cleaned, s)
		}
	}
	b, err := json.Marshal(cleaned)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
