package memory

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// Error list...
var (
	ErrUndefinedKey = errors.New(`undefined key`)
)

type item struct {
	value     string
	expiredAt time.Time
}

// KVMemory accessor to the string `value` by string `key`
type KVMemory struct {
	mx       sync.Mutex
	lifetime time.Duration
	data     map[string]item
}

// NewKVMemory object
func NewKVMemory(lifetime time.Duration) *KVMemory {
	if lifetime == 0 {
		lifetime = time.Minute
	}
	return &KVMemory{
		lifetime: lifetime,
		data:     map[string]item{},
	}
}

// Get returns string value of error
func (kv *KVMemory) Get(ctx context.Context, key string) (string, error) {
	kv.mx.Lock()
	defer kv.mx.Unlock()
	if kv.data == nil {
		return "", errors.Wrap(ErrUndefinedKey, key)
	}
	val, ok := kv.data[key]
	if ok {
		ok = !val.expiredAt.Before(time.Now())
		delete(kv.data, key)
	}
	if !ok {
		return "", errors.Wrap(ErrUndefinedKey, key)
	}
	return val.value, nil
}

// Set `value` of `key`
func (kv *KVMemory) Set(ctx context.Context, key, value string) error {
	kv.mx.Lock()
	defer kv.mx.Unlock()
	if kv.data == nil {
		kv.data = map[string]item{}
	}
	kv.data[key] = item{value: value, expiredAt: time.Now().Add(kv.lifetime)}
	return nil
}

// TrySet `value` of `key`
func (kv *KVMemory) TrySet(ctx context.Context, key, value string) error {
	return kv.Set(ctx, key, value)
}
