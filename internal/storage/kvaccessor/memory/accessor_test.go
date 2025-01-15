package memory

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestKVMemory(t *testing.T) {
	kvAccessor := NewKVMemory(time.Second * 100)

	t.Run("set", func(t *testing.T) {
		err := kvAccessor.Set(context.TODO(), "test1", "value1")
		assert.NoError(t, err)
	})

	t.Run("get", func(t *testing.T) {
		val, err := kvAccessor.Get(context.TODO(), "test1")
		assert.NoError(t, err)
		assert.Equal(t, "value1", val)
	})
}
