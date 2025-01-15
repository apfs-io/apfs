package kvaccessor

import (
	"context"
)

// KVAccessor interface describes the way of manipulation
// of string values by key
type KVAccessor interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	TrySet(ctx context.Context, key, value string) error
}
