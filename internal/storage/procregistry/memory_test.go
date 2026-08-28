package procregistry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryRegistry_AddListRemove(t *testing.T) {
	r := NewMemory()
	ctx := context.Background()

	require.NoError(t, r.Add(ctx, "a"))
	require.NoError(t, r.Add(ctx, "b"))

	ids, err := r.ListOlderThan(ctx, 0)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"a", "b"}, ids)

	require.NoError(t, r.Remove(ctx, "a"))
	ids, err = r.ListOlderThan(ctx, 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"b"}, ids)
}

func TestMemoryRegistry_ListOlderThanSkipsFresh(t *testing.T) {
	r := NewMemory()
	ctx := context.Background()
	require.NoError(t, r.Add(ctx, "fresh"))

	ids, err := r.ListOlderThan(ctx, time.Hour)
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestMemoryRegistry_TryBegin(t *testing.T) {
	r := NewMemory()
	assert.True(t, r.TryBegin("obj"))
	assert.False(t, r.TryBegin("obj"))
	r.End("obj")
	assert.True(t, r.TryBegin("obj"))
}

func TestMemoryRegistry_IsBusy(t *testing.T) {
	r := NewMemory()
	assert.False(t, r.IsBusy("obj"))
	assert.True(t, r.TryBegin("obj"))
	assert.True(t, r.IsBusy("obj"))
	r.End("obj")
	assert.False(t, r.IsBusy("obj"))
}
