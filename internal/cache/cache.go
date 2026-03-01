package cache

// EvictCallback is called when an entry is evicted from the cache.
// Not all providers support eviction callbacks (e.g., Redis relies on server-side eviction).
type EvictCallback func(key string, value []byte)

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
