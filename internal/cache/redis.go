package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheService interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	Incr(ctx context.Context, key string, expiration time.Duration) (int64, error)
	IsAvailable() bool
	Close() error
}

type RedisCache struct {
	client *redis.Client
}

// NewRedisCache initializes a Redis client and checks connectivity.
// If addr is empty or connection fails, it returns a disabled cache that gracefully no-ops.
func NewRedisCache(host, port, password string) *RedisCache {
	if host == "" {
		slog.Info("Redis host not configured, L1 in-memory cache disabled (using PostgreSQL L2 cache)")
		return &RedisCache{client: nil}
	}

	if port == "" {
		port = "6379"
	}

	addr := fmt.Sprintf("%s:%s", host, port)
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           0,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		slog.Warn("Redis server unreachable, continuing with PostgreSQL L2 cache", slog.String("addr", addr), slog.String("error", err.Error()))
		return &RedisCache{client: nil}
	}

	slog.Info("Connected to Redis successfully (L1 cache active)", slog.String("addr", addr))
	return &RedisCache{client: client}
}

func (r *RedisCache) IsAvailable() bool {
	return r != nil && r.client != nil
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	if !r.IsAvailable() {
		return "", fmt.Errorf("redis cache unavailable")
	}
	return r.client.Get(ctx, key).Result()
}

func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if !r.IsAvailable() {
		return nil
	}

	var data interface{}
	switch v := value.(type) {
	case string, []byte, int, int64, float64, bool:
		data = v
	default:
		bytes, err := json.Marshal(v)
		if err != nil {
			return err
		}
		data = bytes
	}

	return r.client.Set(ctx, key, data, expiration).Err()
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	if !r.IsAvailable() {
		return nil
	}
	return r.client.Del(ctx, key).Err()
}

// Incr atomically increments the integer value of a key by one in a single atomic round-trip.
// If the key does not exist, it is initialized to 1 and assigned the provided expiration TTL.
// If the key exists without an expiry, the expiration TTL is guaranteed to be set.
func (r *RedisCache) Incr(ctx context.Context, key string, expiration time.Duration) (int64, error) {
	if !r.IsAvailable() {
		return 0, fmt.Errorf("redis cache unavailable")
	}

	ttlSec := int(expiration.Seconds())
	if ttlSec <= 0 {
		ttlSec = 60
	}

	luaScript := `
		local current = redis.call('INCR', KEYS[1])
		if current == 1 then
			redis.call('EXPIRE', KEYS[1], ARGV[1])
		else
			local ttl = redis.call('TTL', KEYS[1])
			if ttl == -1 then
				redis.call('EXPIRE', KEYS[1], ARGV[1])
			end
		end
		return current
	`

	res, err := r.client.Eval(ctx, luaScript, []string{key}, ttlSec).Result()
	if err != nil {
		return 0, err
	}

	switch v := res.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("unexpected redis eval return type: %T", res)
	}
}

func (r *RedisCache) Close() error {
	if r.IsAvailable() {
		return r.client.Close()
	}
	return nil
}

