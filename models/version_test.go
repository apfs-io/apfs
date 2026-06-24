package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompareVersion(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"2", "2", 0},
		{"2", "3", -1},
		{"3", "2", 1},
		{"2.1", "2.0", 1},
		{"2.0.1", "2.0", 1},
		{"1.10", "1.9", 1},
		{"", "1", -1},
		{"1", "", 1},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, CompareVersion(tc.a, tc.b), "%q vs %q", tc.a, tc.b)
	}
}

func TestWorkflow_GetVersion(t *testing.T) {
	assert.Equal(t, "", (*Workflow)(nil).GetVersion())
	assert.Equal(t, "2", (&Workflow{Version: "2"}).GetVersion())
}
