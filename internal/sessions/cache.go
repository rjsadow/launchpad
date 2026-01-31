// Package sessions provides session management with horizontal scalability support.
package sessions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rjsadow/launchpad/internal/db"
)

// SessionCache defines the interface for session caching.
// This allows for different cache implementations (in-memory, Redis, etc.)
type SessionCache interface {
	// Get retrieves a session from the cache
	Get(ctx context.Context, sessionID string) (*db.Session, error)

	// Set stores a session in the cache with optional TTL
	Set(ctx context.Context, session *db.Session, ttl time.Duration) error

	// Delete removes a session from the cache
	Delete(ctx context.Context, sessionID string) error

	// Close cleans up cache resources
	Close() error
}

// ErrCacheMiss indicates the session was not found in cache
var ErrCacheMiss = errors.New("session not found in cache")

// InMemoryCache provides a local in-memory session cache.
// Suitable for single-instance deployments or development.
type InMemoryCache struct {
	mu       sync.RWMutex
	sessions map[string]*db.Session
}

// NewInMemoryCache creates a new in-memory session cache.
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		sessions: make(map[string]*db.Session),
	}
}

// Get retrieves a session from the in-memory cache.
func (c *InMemoryCache) Get(ctx context.Context, sessionID string) (*db.Session, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	session, ok := c.sessions[sessionID]
	if !ok {
		return nil, ErrCacheMiss
	}
	return session, nil
}

// Set stores a session in the in-memory cache.
// TTL is ignored for in-memory cache (cleanup handled by session manager).
func (c *InMemoryCache) Set(ctx context.Context, session *db.Session, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sessions[session.ID] = session
	return nil
}

// Delete removes a session from the in-memory cache.
func (c *InMemoryCache) Delete(ctx context.Context, sessionID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.sessions, sessionID)
	return nil
}

// Close is a no-op for in-memory cache.
func (c *InMemoryCache) Close() error {
	return nil
}

// RedisCache provides a distributed session cache using Redis.
// Suitable for horizontally scaled deployments.
type RedisCache struct {
	client    *redis.Client
	keyPrefix string
}

// RedisCacheConfig holds configuration for Redis cache.
type RedisCacheConfig struct {
	// Address is the Redis server address (host:port)
	Address string

	// Password for Redis authentication (optional)
	Password string

	// DB is the Redis database number
	DB int

	// KeyPrefix is prepended to all session keys (default: "launchpad:session:")
	KeyPrefix string

	// PoolSize is the maximum number of connections (default: 10)
	PoolSize int

	// MinIdleConns is the minimum number of idle connections (default: 2)
	MinIdleConns int

	// ConnectTimeout is the timeout for establishing connections (default: 5s)
	ConnectTimeout time.Duration

	// ReadTimeout is the timeout for read operations (default: 3s)
	ReadTimeout time.Duration

	// WriteTimeout is the timeout for write operations (default: 3s)
	WriteTimeout time.Duration
}

// DefaultRedisCacheConfig returns a RedisCacheConfig with sensible defaults.
func DefaultRedisCacheConfig() RedisCacheConfig {
	return RedisCacheConfig{
		Address:        "localhost:6379",
		KeyPrefix:      "launchpad:session:",
		PoolSize:       10,
		MinIdleConns:   2,
		ConnectTimeout: 5 * time.Second,
		ReadTimeout:    3 * time.Second,
		WriteTimeout:   3 * time.Second,
	}
}

// NewRedisCache creates a new Redis-backed session cache.
func NewRedisCache(cfg RedisCacheConfig) (*RedisCache, error) {
	// Apply defaults for zero values
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "launchpad:session:"
	}
	if cfg.PoolSize == 0 {
		cfg.PoolSize = 10
	}
	if cfg.MinIdleConns == 0 {
		cfg.MinIdleConns = 2
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 5 * time.Second
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 3 * time.Second
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 3 * time.Second
	}

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Address,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.ConnectTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", cfg.Address, err)
	}

	return &RedisCache{
		client:    client,
		keyPrefix: cfg.KeyPrefix,
	}, nil
}

// Get retrieves a session from Redis.
func (c *RedisCache) Get(ctx context.Context, sessionID string) (*db.Session, error) {
	key := c.keyPrefix + sessionID

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrCacheMiss
		}
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	var session db.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// Set stores a session in Redis with the specified TTL.
func (c *RedisCache) Set(ctx context.Context, session *db.Session, ttl time.Duration) error {
	key := c.keyPrefix + session.ID

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	return nil
}

// Delete removes a session from Redis.
func (c *RedisCache) Delete(ctx context.Context, sessionID string) error {
	key := c.keyPrefix + sessionID

	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis delete failed: %w", err)
	}

	return nil
}

// Close closes the Redis connection.
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// Ping tests the Redis connection.
func (c *RedisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
