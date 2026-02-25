# Ultra-Lean Reverse Proxy

A memory-efficient reverse proxy built with Go, designed to run in ~8-10 MB of RAM.

## Features

- **Extremely low memory footprint**: ~6 MB idle, ~8-10 MB under load
- **Minimal dependencies**: Uses standard Go libraries and `gopkg.in/yaml.v3`
- **Streaming proxy**: Request and response bodies are streamed directly without full buffering
- **CORS support**: Fully configurable via the configuration file
- **Docker-ready**: Designed to run in a minimal scratch-based image
- **System Tuning**: Automatically optimizes GOMAXPROCS and GC settings for low-memory environments

## Configuration

The proxy is configured via a YAML file.

### Sample `config.yaml`

```yaml
listen: :80

routes:
  - path: /api/
    target: http://backend-service:8080/

cors:
  allowed_origins:
    - "https://example.com"
    - "https://app.example.com"

transport:
  dial_timeout: 10
  dial_keep_alive: 30
  max_idle_conns: 10
  max_idle_conns_per_host: 2
  max_conns_per_host: 20
  idle_conn_timeout: 30
  response_header_timeout: 30
  read_buffer_size: 4096
  write_buffer_size: 4096

proxy:
  max_concurrent_requests: 100
```

### CORS Configuration

CORS is now configured in the `cors` section of the `config.yaml`:

- `allowed_origins`: A list of strings. Use `["*"]` to allow all origins.

## Usage

### Run Locally

The proxy requires a path to the configuration file as the first positional argument.

```bash
go run main.go config.yaml
```
### Build and Run with Docker

```bash
docker build -t proxy:go .

# Run with custom config
docker run --rm -p 8080:80 --memory=10m \
  -v ./config.yaml:/config.yaml \
  proxy:go /config.yaml
```

### Using Pre-built Docker Image

You can use the official pre-built image from GitHub Container Registry. You only need to provide your `config.yaml` file via a volume mount.

```bash
docker run --rm -p 8080:80 \
  -v ./your-config.yaml:/config.yaml \
  ghcr.io/pecatatoshev/go-proxy:sha-d0bca50
```

*Note: The image is configured to look for the configuration at `/config.yaml` by default.*

## System Tuning & Environment Variables

The proxy includes built-in optimizations for low-memory environments. These are applied automatically but can be overridden by standard Go environment variables:

- **`GOMAXPROCS`**: Defaults to `2` (limits OS threads to save memory).
- **`GOGC`**: Defaults to `20` (aggressive garbage collection).
- **`GOMEMLIMIT`**: Defaults to `8MiB` (soft memory ceiling).

To override these, simply set the environment variables when running:

```bash
GOMEMLIMIT=16MiB ./proxy config.yaml
```

## Health Check

The proxy provides a basic health check endpoint at `/health`.

```bash
curl http://localhost:8080/health
# ok
```

## Architecture

The project is organized into internal packages for better maintainability:

- `internal/sys`: System tuning and memory management (GC, GOMAXPROCS).
- `internal/proxy`: Core proxy logic, streaming handlers, and middleware.
- `config`: Configuration loading and YAML mapping.
- `health`: Health check handlers.

## Memory Optimizations

1. **`GOMAXPROCS=2`**: Limits OS threads (each has ~1 MB stack).
2. **`GOGC=20`**: Triggers GC at 20% heap growth.
3. **`GOMEMLIMIT=8MiB`**: Signals the runtime to stay under this limit.
4. **Streaming I/O**: Uses pooled 4 KB buffers to avoid loading large bodies into memory.
5. **Periodic GC**: A background routine triggers manual GC and returns memory to the OS every 30s during idle periods.
