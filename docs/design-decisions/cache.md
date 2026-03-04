# Design Decisions — Cache

## Cache-Layer Metrics with Group Label

**Decision**: Prometheus metrics for cache operations (`cache_hits_total`, `cache_misses_total`, `cache_evictions_total`, `cache_entries`) are owned and emitted by the `internal/cache` package, not by callers. All four metrics carry a `cache` label whose value is the `Group` set in `ProviderConfig`. The entry-count gauge uses lazy evaluation (`GaugeFunc`-style `cacheEntriesCollector`) that calls `Len()` at scrape time rather than maintaining an in-process counter.

**Rationale**:

- **Correctness with Redis TTL**: Redis removes expired entries automatically, so an in-process `Inc`/`Dec` counter inevitably drifts. Querying `Len()` at scrape time always reflects the true count.
- **Labels over name prefixes**: Using a `cache` label (e.g., `cache="zip"`) instead of a per-service metric prefix (`subtitle_cache_hits_total`) keeps metric names generic and allows the same cache infrastructure to be reused for other purposes (different groups) without renaming metrics or adding new registrations.
- **Separation of concerns**: Callers create a cache with a `Group` name; all instrumentation is handled transparently by the cache layer via `instrumentedCache`. No metric code leaks into service or downloader layers.

**Implementation**: `internal/cache/metrics.go` (CounterVec definitions + `cacheEntriesCollector`), `internal/cache/instrumented.go` (`instrumentedCache` wrapper), `internal/cache/factory.go` (`New()` wraps the result and injects the eviction counter hook when `Group != ""`).

## Pluggable Cache with Factory Pattern

**Decision**: Abstract the ZIP file cache behind an interface (`cache.Cache`) with a provider registry, allowing the cache backend to be selected via configuration (`cache.type`). Ship two built-in providers: `memory` (in-process LRU) and `redis` (Redis/Valkey-backed LRU).

**Rationale**:

- Decouples the `SubtitleDownloader` from a specific cache implementation
- Enables sharing cache across multiple application instances via Redis/Valkey
- Factory + provider registry pattern makes it easy to add new backends without modifying existing code
- Falls back gracefully to memory if the configured backend fails to initialize

**Redis/Valkey LRU Architecture**:

- Only **2 Redis keys** are used regardless of cache size (not N+1 individual keys):
  - A **Hash** (`sscache:data`) stores all cached values as fields, with per-field TTL via `HPEXPIRE` (Redis 7.4+ / Valkey 8+)
  - A **Sorted Set** (`sscache:lru`) tracks LRU ordering (member = key, score = last-access microsecond timestamp)
- **Atomic Lua scripts** ensure consistency:
  - `getAndTouch`: retrieves a value and refreshes the LRU score in one atomic operation
  - `setAndEvict`: stores a value, sets per-field TTL, updates LRU, and evicts the oldest entries when over capacity
- Expired hash fields are automatically removed by Redis; stale sorted-set members are lazily cleaned during eviction

**Implementation**:

- `internal/cache/cache.go` — `Cache` interface with `Get`, `Set`, `Contains`, `Len`, `Close`
- `internal/cache/factory.go` — Provider registry with `Register`, `New`, `RegisteredProviders`
- `internal/cache/memory.go` — In-memory provider wrapping `hashicorp/golang-lru/v2/expirable`
- `internal/cache/redis.go` — Redis/Valkey provider with Lua scripts for atomic LRU operations
- `internal/services/subtitle_downloader_impl.go` — Uses `cache.Cache` interface; selects backend via `cache.New(cacheType, ...)`
