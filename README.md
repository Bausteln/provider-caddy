# provider-caddy

`provider-caddy` is a [Crossplane](https://crossplane.io/) Provider for [Caddy Server](https://caddyserver.com/), enabling declarative management of Caddy reverse proxy configurations through Kubernetes.

## Overview

This provider allows you to configure Caddy reverse proxy rules and other settings via the Caddy Admin API using Kubernetes Custom Resources. It provides full API compatibility with Caddy's configuration system.

## Features

- **ProxyRoute Resource**: Configure Caddy reverse proxy routes declaratively
- **Full Caddy API Support**: Direct integration with Caddy's Admin API
- **Advanced Routing**: Support for host, path, method, and header-based routing
- **Load Balancing**: Multiple load balancing policies (round_robin, least_conn, ip_hash, etc.)
- **Health Checks**: Active and passive health checking for upstreams
- **Header Manipulation**: Set, add, or delete request and response headers
- **TLS Support**: Configure TLS for upstream connections

## Installation

### Prerequisites

- Kubernetes cluster
- [Crossplane](https://crossplane.io/) installed (v1.14.0+)
- Caddy server with Admin API enabled

### Install the Provider

Using a Provider manifest (recommended):

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-caddy
spec:
  package: ghcr.io/bausteln/provider-caddy:v0.1.0
```

Apply it:

```bash
kubectl apply -f provider.yaml
```

Or install the latest version:

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-caddy
spec:
  package: ghcr.io/bausteln/provider-caddy:latest
```

### Verify Installation

```bash
# Check provider is installed and healthy
kubectl get providers

# Check provider pod is running
kubectl get pods -n crossplane-system

# View provider logs
kubectl logs -n crossplane-system -l pkg.crossplane.io/provider=provider-caddy
```

## Quick Start

### 1. Create a Simple Proxy Route

```yaml
apiVersion: config.caddy.crossplane.io/v1alpha1
kind: ProxyRoute
metadata:
  name: simple-proxy
spec:
  forProvider:
    # Caddy admin API endpoint
    caddyEndpoint: http://caddy-server:2019

    # Route matching
    match:
      host:
        - api.example.com

    # Backend upstream
    upstreams:
      - dial: backend:8080
```

### 2. Advanced Configuration Example

```yaml
apiVersion: config.caddy.crossplane.io/v1alpha1
kind: ProxyRoute
metadata:
  name: advanced-proxy
spec:
  forProvider:
    caddyEndpoint: http://caddy-server:2019

    # Match conditions
    match:
      host:
        - example.com
        - www.example.com
      path:
        - /api/*
      method:
        - GET
        - POST

    # Multiple upstreams
    upstreams:
      - dial: backend1:8080
      - dial: backend2:8080
      - dial: backend3:8080

    # Load balancing
    loadBalancing:
      policy: round_robin
      tryDuration: 30s
      tryInterval: 250ms

    # Health checks
    healthChecks:
      active:
        path: /health
        interval: 30s
        timeout: 5s
      passive:
        maxFails: 3
        unhealthyLatency: 3s

    # Header manipulation
    headers:
      request:
        set:
          X-Forwarded-Proto:
            - https
      response:
        set:
          X-Served-By:
            - caddy-proxy

    # TLS for upstream connections
    tls:
      enabled: true
      serverName: backend.internal
```

## ProxyRoute Specification

### Core Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `caddyEndpoint` | string | Yes | Caddy Admin API endpoint (e.g., `http://localhost:2019`) |
| `serverName` | string | No | Caddy server name (default: `srv0`) |
| `upstreams` | array | Yes | List of backend servers |
| `match` | object | No | Route matching conditions |
| `loadBalancing` | object | No | Load balancing configuration |
| `headers` | object | No | Header manipulation rules |
| `healthChecks` | object | No | Health check configuration |
| `tls` | object | No | TLS settings for upstream connections |

### Match Conditions

```yaml
match:
  host:          # List of host names to match
    - example.com
  path:          # List of paths to match (supports wildcards)
    - /api/*
  method:        # HTTP methods to match
    - GET
    - POST
  headers:       # Header-based matching
    X-Custom-Header:
      - value1
```

### Upstreams

```yaml
upstreams:
  - dial: backend1:8080
    maxRequests: 100  # Optional: max concurrent requests
  - dial: backend2:8080
```

### Load Balancing Policies

Supported policies:
- `random` - Random selection
- `round_robin` - Round-robin distribution
- `least_conn` - Least connections
- `ip_hash` - IP-based hashing
- `header` - Header-based selection
- `cookie` - Cookie-based selection

```yaml
loadBalancing:
  policy: round_robin
  tryDuration: 30s
  tryInterval: 250ms
```

### Health Checks

```yaml
healthChecks:
  active:
    path: /health
    interval: 30s
    timeout: 5s
  passive:
    maxFails: 3
    unhealthyLatency: 3s
```

### Header Manipulation

```yaml
headers:
  request:
    set:
      X-Forwarded-Proto: [https]
    add:
      X-Custom-Header: [value1]
    delete:
      - X-Unwanted-Header
  response:
    set:
      X-Served-By: [caddy]
```

## Architecture

The provider follows the standard Crossplane provider pattern:

```
ProxyRoute CR → Controller → Caddy Client → Caddy Admin API
```

1. User creates a ProxyRoute custom resource
2. Provider controller watches for changes
3. Caddy client translates CR to Caddy API calls
4. Caddy Admin API applies the configuration

## Development

### Building the Provider

```bash
# Initialize submodules
make submodules

# Build the provider
make build

# Run tests
make test

# Generate CRDs and code
go generate ./...
make generate
```

### Project Structure

```
provider-caddy/
├── apis/                      # API definitions
│   ├── config/v1alpha1/      # ProxyRoute CRD
│   └── v1alpha1/             # ProviderConfig CRD
├── internal/
│   ├── clients/caddy/        # Caddy API client
│   └── controller/           # Controllers
│       ├── proxyroute/       # ProxyRoute controller
│       └── config/           # ProviderConfig controller
├── examples/                  # Example configurations
└── package/crds/             # Generated CRD manifests
```

### Adding New Resource Types

The provider can be extended to support additional Caddy features:

1. Create new CRD in `apis/config/v1alpha1/`
2. Extend the Caddy client in `internal/clients/caddy/`
3. Create controller in `internal/controller/`
4. Register controller in `internal/controller/register.go`
5. Generate code with `go generate ./...`

## Roadmap

Future enhancements:
- [ ] **Server** resource for full server configuration
- [ ] **App** resource for managing Caddy apps (HTTP, TLS, PKI)
- [ ] **Config** resource for complete Caddy configuration management
- [ ] **TLS** resource for certificate management
- [ ] Support for Caddy modules and plugins
- [ ] Metrics and observability integration
- [ ] Multi-cluster Caddy coordination

## Contributing

Contributions are welcome! Please refer to Crossplane's [CONTRIBUTING.md](https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md) and the [Provider Development Guide](https://github.com/crossplane/crossplane/blob/master/contributing/guide-provider-development.md).

## License

provider-caddy is under the Apache 2.0 license.

## Resources

- [Caddy Documentation](https://caddyserver.com/docs/)
- [Caddy Admin API](https://caddyserver.com/docs/api)
- [Crossplane Documentation](https://crossplane.io/docs/)
- [Provider Development Guide](https://github.com/crossplane/crossplane/blob/master/contributing/guide-provider-development.md)
