package redis

import (
	// Importing necessary packages for context management, error handling, URL parsing, and Redis client
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/demdxx/gocast/v2"          // Utility for type casting
	goredis "github.com/redis/go-redis/v9" // Redis client library
)

// Error returned when a value is locked and cannot be set
var errValueIsLocked = errors.New(`value is locked`)

// Accessor provides an interface to interact with Redis KV storage
type Accessor struct {
	client   *goredis.Client // Redis client instance
	lifetime time.Duration   // Default expiration time for keys
}

// New creates a new Redis connection using the provided connection string
// connection: tcp://:password@localhost:6379/0
func New(connection string) (*Accessor, error) {
	parsedURL, err := url.Parse(connection) // Parse the connection string
	if err != nil {
		return nil, err
	}

	// Adjust scheme if necessary (e.g., redis -> tcp)
	if parsedURL.Scheme == "redis" {
		parsedURL.Scheme = "tcp"
	}

	// Create a new Redis client with options derived from the connection string
	client := goredis.NewClient(&goredis.Options{
		Network:               parsedURL.Scheme,                                 // Network type (e.g., tcp)
		Addr:                  parsedURL.Host,                                   // Redis server address
		Password:              parsedURL.User.Username(),                        // Password for authentication
		DB:                    gocast.Int(strings.Trim(parsedURL.Path, "/")),    // Database index
		PoolSize:              gocast.Int(parsedURL.Query().Get(`pool`)),        // Connection pool size
		MaxRetries:            gocast.Int(parsedURL.Query().Get(`max_retries`)), // Max retry attempts
		MinIdleConns:          gocast.Int(parsedURL.Query().Get(`idle_cons`)),   // Minimum idle connections
		ContextTimeoutEnabled: true,                                             // Enable context timeout
	})

	// Parse the lifetime parameter for key expiration
	lifetime, _ := time.ParseDuration(parsedURL.Query().Get(`lifetime`))
	if lifetime == 0 {
		lifetime = time.Second // Default to 1 second if not specified
	}

	// Return the initialized Accessor
	return &Accessor{client: client, lifetime: lifetime}, nil
}

// Get retrieves the value associated with the given key from Redis
func (ac *Accessor) Get(ctx context.Context, key string) (string, error) {
	return ac.client.Get(ctx, key).Result()
}

// Set stores a value with the given key in Redis, using the default lifetime
func (ac *Accessor) Set(ctx context.Context, key, value string) error {
	return ac.client.SetNX(ctx, key, value, ac.lifetime).Err()
}

// TrySet attempts to set a value for the given key only if it does not already exist
// Returns an error if the key is locked (already exists)
func (ac *Accessor) TrySet(ctx context.Context, key, value string) error {
	res := ac.client.SetNX(ctx, key, value, ac.lifetime) // Set key if it does not exist
	err := res.Err()
	if err == nil && !res.Val() { // Check if the key was not set
		err = errValueIsLocked // Return locked error
	}
	return err
}

// Close closes the Redis client connection
func (ac *Accessor) Close() error {
	return ac.client.Close()
}
