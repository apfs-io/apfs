package converters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPtr(t *testing.T) {
	var st struct{}
	assert.Equal(t, uintptr(0), ptr(nil))
	assert.Equal(t, uintptr(0), ptr(any(nil)))
	assert.Equal(t, uintptr(0), ptr(st))
	assert.NotEqual(t, uintptr(0), ptr(&st))
}
