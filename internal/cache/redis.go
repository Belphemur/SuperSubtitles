package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// defaultKeyPrefix namespaces all cache keys in Redis to avoid collisions.
	defaultKeyPrefix = "sscache:"
)

func init() {
	Register("redis", newRedisCache)
}

// redisCache implements the Cache interface using Redis/Valkey with
// application-level LRU semantics.
//
// Requires Redis 7.4+ or Valkey 8+ for per-field hash TTL (HPEXPIRE command).
// Using an older version will cause Set operations to fail silently (values are stored
// but won't expire automatically).
//
// Data is stored in just 2 Redis keys (regardless of the number of cache entries):
//
//   - {prefix}data — a Hash that stores all cached values (field = user key, value = bytes).
//     Per-field TTL is set via HPEXPIRE (Redis 7.4+ / Valkey 8+), so expired fields are
//     automatically removed by Redis without application-side cleanup.
//   - {prefix}lru  — a Sorted Set that tracks LRU ordering (member = user key,
//     score = last-access µs timestamp).
//
// Lua scripts ensure that Get (touch) and Set (write + evict) are each executed atomically.
// Stale LRU entries (whose hash field has expired) are lazily cleaned during eviction.
type redisCache struct {
	client  *redis.Client
	ttl     time.Duration
	maxSize int
	onEvict EvictCallback
	logger  Logger
	dataKey string // hash key, e.g. "sscache:data"
	lruKey  string // sorted set key, e.g. "sscache:lru"
}

// getAndTouch atomically retrieves a value from the hash and refreshes
// the LRU score when the entry exists.
//
// KEYS[1] = data hash, KEYS[2] = LRU sorted set
// ARGV[1] = current µs timestamp, ARGV[2] = member (user key)
//
// Returns the value on hit, or nil on miss (including expired fields).
var getAndTouch = redis.NewScript(`
local val = redis.call('HGET', KEYS[1], ARGV[2])
if val then
    redis.call('ZADD', KEYS[2], ARGV[1], ARGV[2])
end
return val
`)

// setAndEvict atomically stores a value in the hash, sets per-field TTL via
// HPEXPIRE, updates LRU tracking, and evicts the least-recently-used entries
// when the cache exceeds maxSize. Stale sorted-set members whose hash field
// has already expired are silently cleaned up during eviction.
//
// KEYS[1] = data hash, KEYS[2] = LRU sorted set
// ARGV[1] = value, ARGV[2] = current µs timestamp, ARGV[3] = member (user key),
// ARGV[4] = maxSize, ARGV[5] = TTL in milliseconds
//
// Returns a list of evicted member names (may be empty).
var setAndEvict = redis.NewScript(`
local member  = ARGV[3]
local maxSize = tonumber(ARGV[4])
local ttlMs   = tonumber(ARGV[5])

-- Store value and set per-field TTL
redis.call('HSET', KEYS[1], member, ARGV[1])
redis.call('HPEXPIRE', KEYS[1], ttlMs, 'FIELDS', 1, member)

-- Update LRU score
redis.call('ZADD', KEYS[2], ARGV[2], member)

-- Evict least-recently-used entries if over capacity.
-- If the hash field was already expired by Redis, HDEL is a harmless no-op
-- and we still clean the stale sorted-set member.
local size = redis.call('ZCARD', KEYS[2])
local evicted = {}
while size > maxSize do
    local oldest = redis.call('ZPOPMIN', KEYS[2], 1)
    if #oldest == 0 then break end
    local oldMember = oldest[1]
    redis.call('HDEL', KEYS[1], oldMember)
    table.insert(evicted, oldMember)
    size = size - 1
end

return evicted
`)

func newRedisCache(cfg ProviderConfig) (Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// Verify connectivity.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	prefix := defaultKeyPrefix
	return &redisCache{
		client:  client,
		ttl:     cfg.TTL,
		maxSize: cfg.Size,
		onEvict: cfg.OnEvict,
		logger:  cfg.Logger,
		dataKey: prefix + "data",
		lruKey:  prefix + "lru",
	}, nil
}

func (r *redisCache) keys() []string {
	return []string{r.dataKey, r.lruKey}
}

func (r *redisCache) logError(msg string, err error) {
	if r.logger != nil {
		r.logger.Error(msg, err)
	}
}

func (r *redisCache) Get(key string) ([]byte, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	now := strconv.FormatInt(time.Now().UnixMicro(), 10)
	result, err := getAndTouch.Run(ctx, r.client, r.keys(), now, key).Text()
	if err != nil {
		// redis.Nil means the key doesn't exist — a normal cache miss.
		if !errors.Is(err, redis.Nil) {
			r.logError("redis cache Get failed", err)
		}
		return nil, false
	}
	return []byte(result), true
}

func (r *redisCache) Set(key string, value []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	now := strconv.FormatInt(time.Now().UnixMicro(), 10)
	maxSize := strconv.Itoa(r.maxSize)
	ttlMs := strconv.FormatInt(r.ttl.Milliseconds(), 10)

	evicted, err := setAndEvict.Run(ctx, r.client, r.keys(),
		value, now, key, maxSize, ttlMs,
	).StringSlice()

	if err != nil {
		r.logError("redis cache Set failed", err)
		return
	}

	if len(evicted) == 0 {
		return
	}

	if r.onEvict != nil {
		// Value is nil because retrieving evicted values from Redis would require
		// additional roundtrips. Callers should only rely on the key for bookkeeping.
		for _, evictedKey := range evicted {
			r.onEvict(evictedKey, nil)
		}
	}
}

func (r *redisCache) Contains(key string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	n, err := r.client.HExists(ctx, r.dataKey, key).Result()
	if err != nil {
		r.logError("redis cache Contains failed", err)
	}
	return err == nil && n
}

func (r *redisCache) Len() int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	n, err := r.client.HLen(ctx, r.dataKey).Result()
	if err != nil {
		r.logError("redis cache Len failed", err)
		return 0
	}
	return int(n)
}

func (r *redisCache) Close() error {
	return r.client.Close()
}
