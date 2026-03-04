# SuperSubtitles — Configuration

Configuration is loaded from `config/config.yaml` using Viper. Environment variables are supported with `APP_` prefix, with nested keys mapped by replacing `.` with `_` (for example, `server.address` → `APP_SERVER_ADDRESS`).

## Configuration Fields

| Field                     | Description                           | Default                                                                            | Env Var                        |
| ------------------------- | ------------------------------------- | ---------------------------------------------------------------------------------- | ------------------------------ |
| `proxy_connection_string` | HTTP proxy URL (optional)             | `""`                                                                               | `APP_PROXY_CONNECTION_STRING`  |
| `super_subtitle_domain`   | Base URL for feliratok.eu             | `https://feliratok.eu`                                                             | `APP_SUPER_SUBTITLE_DOMAIN`    |
| `client_timeout`          | HTTP client timeout (Go duration)     | `30s`                                                                              | `APP_CLIENT_TIMEOUT`           |
| `user_agent`              | User-Agent header for HTTP requests   | `Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:147.0) Gecko/20100101 Firefox/147.0` | `APP_USER_AGENT`               |
| `server.port`             | Server listening port                 | `8080`                                                                             | `APP_SERVER_PORT`              |
| `server.address`          | Server listening address              | `localhost`                                                                        | `APP_SERVER_ADDRESS`           |
| `log_level`               | Zerolog level (debug/info/warn/error) | `info`                                                                             | `APP_LOG_LEVEL` or `LOG_LEVEL` |
| `log_format`              | Log output format (console/json); defaults to console for unrecognized values | `console`                                                                          | `APP_LOG_FORMAT` or `LOG_FORMAT` |
| `cache.size`              | Maximum entries in LRU ZIP cache      | `2000`                                                                             | `APP_CACHE_SIZE`               |
| `cache.ttl`               | LRU cache TTL (Go duration)           | `24h`                                                                              | `APP_CACHE_TTL`                |
| `cache.type`              | Cache backend (`memory` or `redis`)   | `memory`                                                                           | `APP_CACHE_TYPE`               |
| `cache.redis.address`     | Redis/Valkey server address           | `localhost:6379`                                                                   | `APP_CACHE_REDIS_ADDRESS`      |
| `cache.redis.password`    | Redis/Valkey password (optional)      | `""`                                                                               | `APP_CACHE_REDIS_PASSWORD`     |
| `cache.redis.db`          | Redis/Valkey database number          | `0`                                                                                | `APP_CACHE_REDIS_DB`           |
| `metrics.enabled`         | Enable Prometheus metrics endpoint    | `true`                                                                             | `APP_METRICS_ENABLED`          |
| `metrics.port`            | Port for the metrics HTTP server      | `9090`                                                                             | `APP_METRICS_PORT`             |
| `retry.max_attempts`      | Total HTTP attempts per request (1 = no retry, 0 uses default 3) | `3`                                                                   | `APP_RETRY_MAX_ATTEMPTS`       |
| `retry.initial_delay`     | Delay before the first retry (exponential back-off base, empty = no delay) | `1s`                                                           | `APP_RETRY_INITIAL_DELAY`      |
| `retry.max_delay`         | Maximum back-off delay cap (empty = use initial_delay as cap) | `10s`                                                                 | `APP_RETRY_MAX_DELAY`          |

## Example Configuration

```yaml
proxy_connection_string: ""
super_subtitle_domain: "https://feliratok.eu"
client_timeout: "30s"
user_agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:147.0) Gecko/20100101 Firefox/147.0"
log_level: "info"
log_format: "console"

server:
  port: 8080
  address: "localhost"

cache:
  type: "memory"  # "memory" (in-process LRU) or "redis" (Redis/Valkey-backed LRU)
  size: 2000
  ttl: "24h"
  redis:
    address: "localhost:6379"
    password: ""
    db: 0

metrics:
  enabled: true
  port: 9090

retry:
  max_attempts: 3      # Total attempts including the initial try (1 = no retry)
  initial_delay: "1s"  # Delay before the first retry (exponential back-off base)
  max_delay: "10s"     # Maximum back-off delay cap
```

## Environment Variables

```bash
export LOG_LEVEL=debug

# Enable JSON logging
export LOG_FORMAT=json

# Override server address
export APP_SERVER_ADDRESS=0.0.0.0

# Override domain
export APP_SUPER_SUBTITLE_DOMAIN=https://feliratok.eu

# Override timeout
export APP_CLIENT_TIMEOUT=60s

# Override metrics port
export APP_METRICS_PORT=9091

# Disable metrics
export APP_METRICS_ENABLED=false

# Disable retries (1 attempt = no retry)
export APP_RETRY_MAX_ATTEMPTS=1

# Tune retry back-off
export APP_RETRY_INITIAL_DELAY=500ms
export APP_RETRY_MAX_DELAY=5s
```
