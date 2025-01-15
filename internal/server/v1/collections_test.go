package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollections(t *testing.T) {
	accessor, err := newKVAccessor("memory")
	assert.NoError(t, err)
	assert.NotNil(t, accessor)
}
