package procregistry

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/demdxx/gocast/v2"
	goredis "github.com/redis/go-redis/v9"
)

type redisRegistry struct {
	client *goredis.Client
	busy   sync.Map
}

// NewRedis builds a Registry backed by a Redis HASH at inflightKey.
func NewRedis(connection string) (Registry, error) {
	parsedURL, err := url.Parse(connection)
	if err != nil {
		return nil, err
	}
	if parsedURL.Scheme == "redis" {
		parsedURL.Scheme = "tcp"
	}
	password := ""
	if parsedURL.User != nil {
		password, _ = parsedURL.User.Password()
		if password == "" {
			password = parsedURL.User.Username()
		}
	}
	client := goredis.NewClient(&goredis.Options{
		Network:               parsedURL.Scheme,
		Addr:                  parsedURL.Host,
		Password:              password,
		DB:                    gocast.Int(strings.Trim(parsedURL.Path, "/")),
		ContextTimeoutEnabled: true,
	})
	return &redisRegistry{client: client}, nil
}

func (r *redisRegistry) Add(ctx context.Context, objectID string) error {
	if objectID == "" {
		return nil
	}
	return r.client.HSet(ctx, inflightKey, objectID, strconv.FormatInt(time.Now().UnixNano(), 10)).Err()
}

func (r *redisRegistry) Remove(ctx context.Context, objectID string) error {
	r.busy.Delete(objectID)
	return r.client.HDel(ctx, inflightKey, objectID).Err()
}

func (r *redisRegistry) ListOlderThan(ctx context.Context, age time.Duration) ([]string, error) {
	all, err := r.client.HGetAll(ctx, inflightKey).Result()
	if err != nil {
		return nil, err
	}
	cutoff := time.Now().Add(-age).UnixNano()
	out := make([]string, 0, len(all))
	for id, raw := range all {
		ts, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			out = append(out, id)
			continue
		}
		if ts <= cutoff {
			out = append(out, id)
		}
	}
	return out, nil
}

func (r *redisRegistry) TryBegin(objectID string) bool {
	if objectID == "" {
		return false
	}
	_, loaded := r.busy.LoadOrStore(objectID, struct{}{})
	return !loaded
}

func (r *redisRegistry) End(objectID string) {
	r.busy.Delete(objectID)
}

func (r *redisRegistry) IsBusy(objectID string) bool {
	_, ok := r.busy.Load(objectID)
	return ok
}
