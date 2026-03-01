package cache

// EvictCallback is called when an entry is evicted from the cache.
// Support for eviction callbacks is provider-specific. For example, the Redis/Valkey
// provider performs application-level LRU eviction and can invoke this callback.
type EvictCallback func(key string, value []byte)

// Logger is a minimal logging interface for cache providers to report errors.
// This avoids coupling the cache package to a specific logging framework.
type Logger interface {
	// Error logs a message at error level with an associated error value.
	Error(msg string, err error, keysAndValues ...any)
}

// Cache defines the interface for key-value caching with LRU semantics.
// Implementations may use in-memory storage or external backends like Redis/Valkey.
type Cache interface {
	// Get retrieves a value by key. Returns the value and true if found, or nil and false if not.
	Get(key string) ([]byte, bool)

	// Set stores a value with the given key. If the key already exists, it is overwritten.
	Set(key string, value []byte)

	// Contains checks whether a key exists in the cache without affecting LRU ordering.
	Contains(key string) bool

	// Len returns the number of entries currently in the cache.
	// For external backends like Redis, this may reflect the total key count in the configured database.
	Len() int

	// Close releases any resources held by the cache (e.g., network connections).
	// For in-memory caches, this is a no-op.
	Close() error
}
