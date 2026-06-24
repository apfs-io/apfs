package models

import (
	"strconv"
	"strings"
)

// GetVersion returns the workflow version safely.
func (w *Workflow) GetVersion() string {
	if w == nil {
		return ""
	}
	return w.Version
}

// CompareVersion compares two dotted numeric version strings.
// Returns -1 when a < b, 0 when equal, 1 when a > b.
func CompareVersion(a, b string) int {
	if a == b {
		return 0
	}
	pa := parseVersionParts(a)
	pb := parseVersionParts(b)
	maxLen := len(pa)
	if len(pb) > maxLen {
		maxLen = len(pb)
	}
	for i := 0; i < maxLen; i++ {
		va := versionPartAt(pa, i)
		vb := versionPartAt(pb, i)
		if va != vb {
			if va < vb {
				return -1
			}
			return 1
		}
	}
	return strings.Compare(a, b)
}

func parseVersionParts(v string) []string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return strings.Split(v, ".")
}

func versionPartAt(parts []string, idx int) int64 {
	if idx >= len(parts) {
		return 0
	}
	n, err := strconv.ParseInt(strings.TrimSpace(parts[idx]), 10, 64)
	if err != nil {
		return 0
	}
	return n
}
