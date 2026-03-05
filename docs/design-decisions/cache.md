# Design Decisions — Cache

## Cache-Layer Metrics with Group Label

**Decision**: Prometheus metrics for cache operations (hits, misses, evictions, entries) are owned and emitted by the cache package, not by callers. All metrics carry a `cache` label whose value is a group name provided at creation time. The entry-count gauge uses lazy evaluation that queries the actual count at scrape time rather than maintaining an in-process counter.

**Rationale**:

- **Correctness with Redis TTL**: Redis removes expired entries automatically, so an in-process counter inevitably drifts. Querying at scrape time always reflects the true count.
- **Labels over name prefixes**: Using a label (e.g., `cache="zip"`) instead of a per-service metric prefix keeps metric names generic and allows the same infrastructure to be reused for different cache groups without renaming metrics.
- **Separation of concerns**: Callers create a cache with a group name; all instrumentation is handled transparently by a wrapper. No metric code leaks into service layers.

**Implementation**: `internal/cache/metrics.go` (CounterVec definitions + `cacheEntriesCollector`), `internal/cache/instrumented.go` (`instrumentedCache` wrapper), `internal/cache/factory.go` (`New()` wraps the result and injects the eviction counter hook when `Group != ""`).

## Pluggable Cache with Factory Pattern

**Decision**: Abstract the ZIP file cache behind an interface with a provider registry, allowing the backend to be selected via configuration. Two built-in providers ship: `memory` (in-process LRU) and `redis` (Redis/Valkey-backed LRU).

**Rationale**:

- Decouples the download service from a specific cache implementation
- Enables sharing cache across multiple application instances via Redis/Valkey
- Factory + provider registry makes it easy to add new backends without modifying existing code
- Falls back gracefully to memory if the configured backend fails to initialize

**Redis/Valkey LRU Architecture**:

- Only **2 Redis keys** are used regardless of cache size:
  - A **Hash** stores all cached values as fields, with per-field TTL (requires Redis 7.4+ / Valkey 8+)
  - A **Sorted Set** tracks LRU ordering (score = last-access timestamp)
- **Atomic Lua scripts** ensure consistency for get-and-touch and set-and-evict operations
- Expired hash fields are automatically removed by Redis; stale sorted-set members are lazily cleaned during eviction

**Implementation**:

- `internal/cache/cache.go` — `Cache` interface with `Get`, `Set`, `Contains`, `Len`, `Close`
- `internal/cache/factory.go` — Provider registry with `Register`, `New`, `RegisteredProviders`
- `internal/cache/memory.go` — In-memory provider wrapping `hashicorp/golang-lru/v2/expirable`
- `internal/cache/redis.go` — Redis/Valkey provider with Lua scripts for atomic LRU operations
- `internal/services/subtitle_downloader_impl.go` — Uses `cache.Cache` interface; selects backend via `cache.New(cacheType, ...)`
