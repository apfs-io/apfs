package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandStr(t *testing.T) {
	assert.Equal(t, 10, len(RandStr(10)))
}
