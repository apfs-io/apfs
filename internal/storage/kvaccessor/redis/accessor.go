package redis

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/demdxx/gocast/v2"
	goredis "github.com/go-redis/redis/v8"
)

var errValueIsLocked = errors.New(`value is locked`)

// Accessor to the redis KV storage
type Accessor struct {
	client   *goredis.Client
	lifetime time.Duration
}

// New connection to redis
// connection: tcp://:password@localhost:6379/0
func New(connection string) (*Accessor, error) {
	parsedURL, err := url.Parse(connection)
	if err != nil {
		return nil, err
	}
	if parsedURL.Scheme == "redis" {
		parsedURL.Scheme = "tcp"
	}
	client := goredis.NewClient(&goredis.Options{
		Network:      parsedURL.Scheme,
		Addr:         parsedURL.Host,
		Password:     parsedURL.User.Username(),
		DB:           gocast.Int(strings.Trim(parsedURL.Path, "/")),
		PoolSize:     gocast.Int(parsedURL.Query().Get(`pool`)),
		MaxRetries:   gocast.Int(parsedURL.Query().Get(`max_retries`)),
		MinIdleConns: gocast.Int(parsedURL.Query().Get(`idle_cons`)),
	})
	lifetime, _ := time.ParseDuration(parsedURL.Query().Get(`lifetime`))
	if lifetime == 0 {
		lifetime = time.Second
	}
	return &Accessor{client: client, lifetime: lifetime}, nil
}

// Get returns value from the key
func (ac *Accessor) Get(ctx context.Context, key string) (string, error) {
	return ac.client.Get(ctx, key).Result()
}

// Set value of the processing
func (ac *Accessor) Set(ctx context.Context, key, value string) error {
	return ac.client.SetNX(ctx, key, value, ac.lifetime).Err()
}

// TrySet value if not exists
func (ac *Accessor) TrySet(ctx context.Context, key, value string) error {
	res := ac.client.SetNX(ctx, key, value, ac.lifetime)
	err := res.Err()
	if err == nil && !res.Val() {
		err = errValueIsLocked
	}
	return err
}
