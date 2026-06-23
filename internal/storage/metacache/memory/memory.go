// Package memory provides an in-process MetaCache backed by sync.Map with TTL.
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/apfs-io/apfs/internal/storage/metacache"
	"github.com/apfs-io/apfs/models"
)

type entry struct {
	meta      *models.Meta
	expiresAt time.Time
}

// Cache is a goroutine-safe in-memory MetaCache.
type Cache struct {
	mu         sync.RWMutex
	data       map[string]*entry
	defaultTTL time.Duration
}

// New creates an in-memory MetaCache with the given default TTL.
// If ttl is zero DefaultTTL is used.
func New(defaultTTL time.Duration) *Cache {
	if defaultTTL <= 0 {
		defaultTTL = metacache.DefaultTTL
	}
	c := &Cache{
		data:       make(map[string]*entry),
		defaultTTL: defaultTTL,
	}
	return c
}

// Get implements MetaCache.
func (c *Cache) Get(_ context.Context, id string) (*models.Meta, error) {
	c.mu.RLock()
	e, ok := c.data[id]
	c.mu.RUnlock()
	if !ok || e == nil || time.Now().After(e.expiresAt) {
		return nil, nil
	}
	return e.meta, nil
}

// Set implements MetaCache.
func (c *Cache) Set(_ context.Context, id string, meta *models.Meta, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = c.defaultTTL
	}
	c.mu.Lock()
	c.data[id] = &entry{meta: meta, expiresAt: time.Now().Add(ttl)}
	c.mu.Unlock()
	return nil
}

// Invalidate implements MetaCache.
func (c *Cache) Invalidate(_ context.Context, id string) error {
	c.mu.Lock()
	delete(c.data, id)
	c.mu.Unlock()
	return nil
}

var _ metacache.MetaCache = (*Cache)(nil)
