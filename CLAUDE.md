# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is a customized fork of Go Micro v5 - a microservices framework providing distributed systems development capabilities including RPC, service discovery, async messaging, and pluggable architecture. The codebase includes custom integrations with private `zigo2048/mcbeam-common-lib` packages and maintains backward compatibility during service migration.

### Related Repositories

**Deployment Scripts Repository**: `go-micro-deploy`
- Location: `/home/hello/go/src/github.com/zigo2048/go-micro-deploy`
- Purpose: Contains deployment configurations, scripts, and infrastructure-as-code
- Key Components:
  - Fly.io deployment configurations
  - Kubernetes manifests 
  - Docker configurations
  - Infrastructure automation scripts
  - Configuration migration tools

**Framework Extensions Repository**: `mcbeam-common-lib`
- Location: `/home/hello/go/src/github.com/zigo2048/mcbeam-common-lib`
- Purpose: Private library containing custom plugins and extensions for go-micro
- Key Components:
  - Configuration plugins (Apollo, Consul)
  - Authentication and authorization modules
  - Custom middleware and wrappers
  - Utility functions and common interfaces
  - Metrics and observability components

## Essential Development Commands

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...

# Test specific packages
go test ./client/...
go test ./server/...
go test ./broker/...
go test ./transport/...

# Run specific test function
go test -run TestBroker ./broker
go test -run TestGRPCTransport ./transport/grpc

# Run benchmarks
go test -bench=. ./...

# Check for race conditions
go test -race ./...
```

### Building and Installing
```bash
# Build all binaries
go build ./...

# Build protoc plugin
go build ./cmd/protoc-gen-micro

# Install protoc plugin
go install ./cmd/protoc-gen-micro

# Verify dependencies
go mod verify

# Download and tidy dependencies
go mod download && go mod tidy

# Update all dependencies
go get -u ./...
```

### Protocol Buffer Generation
```bash
# Generate protobuf code (requires protoc, protoc-gen-go, protoc-gen-micro)
protoc --proto_path=. --micro_out=. --go_out=. path/to/file.proto

# Generate with grpc support
protoc --proto_path=. --micro_out=. --go_out=. --go-grpc_out=. path/to/file.proto

# Regenerate existing proto files
protoc --proto_path=. --micro_out=. --go_out=. transport/grpc/proto/transport.proto
protoc --proto_path=. --micro_out=. --go_out=. debug/proto/debug.proto
protoc --proto_path=. --micro_out=. --go_out=. server/proto/server.proto
protoc --proto_path=. --micro_out=. --go_out=. errors/errors.proto
protoc --proto_path=. --micro_out=. --go_out=. codec/protorpc/envelope.proto
```

### Sync to Remote (Custom)
```bash
# Sync to remote server (excludes git and IDE files)
make sync
```

## Architecture Overview

### Core Framework Components

**Service Layer** (`micro.go`, `service.go`):
- Main service interface with lifecycle management (Init, Start, Stop, Run)
- Integration hub for all distributed system components
- Service wrapping and middleware chaining

**Client/Server** (`client/`, `server/`):
- RPC client with automatic retry, backoff, and connection pooling
- Server with request handling, streaming support, and middleware
- Both support wrapper/middleware pattern for cross-cutting concerns
- Codec negotiation and content-type based serialization

**Multi-tier Connection Pooling**:
- `util/pool/` - Generic connection pool interface for standard transports
- `client/grpc/grpc_pool.go` - Specialized gRPC pool with stream multiplexing
- `util/socket/pool.go` - WebSocket connection management
- `client/rpc_client.go` - Implements dual-pool strategy for backward compatibility

**Service Discovery & Registry** (`registry/`):
- Default mDNS (multicast DNS) for local development
- Built-in etcd support at `registry/etcd/` for production
- Consul support at `registry/consul/` 
- Memory and cache implementations for testing
- Watch functionality for service changes

**Message Broker** (`broker/`):
- HTTP broker for simple pub/sub
- NATS broker at `broker/nats/` for production messaging
- Memory broker for testing
- Support for queue groups and message acknowledgment

**Transport Layer** (`transport/`):
- HTTP transport with connection pooling
- gRPC transport at `transport/grpc/` with stream support
- Memory transport for unit testing
- Headers package for standard header management

**Config Management** (`config/`):
- Dynamic configuration with hot-reload capability
- Multiple sources: files, environment, CLI flags, memory
- Apollo configuration center integration via mcbeam-common-lib (legacy)
- Consul configuration center via mcbeam-common-lib (recommended for Fly.io)
- Configuration migration tools available in `go-micro-deploy/fly.io/config-migration/`
- Encoder support for JSON, YAML, TOML

### Plugin Architecture & Custom Integrations

This fork integrates with private `zigo2048/mcbeam-common-lib` packages:
- **Apollo Config**: `philchia/agollo/v4` + custom Apollo plugin at `mcbeam-common-lib/plugins/config/apollo/`
- **Consul Config**: Custom Consul adapter at `mcbeam-common-lib/plugins/config/consul/` (Apollo replacement)
- **NATS Broker**: Built-in at `broker/nats/` with connection management
- **etcd Registry**: Built-in at `registry/etcd/` with TTL and heartbeat
- **OpenTelemetry**: Distributed tracing at `wrapper/trace/opentelemetry/`
- **JWT Auth**: Token support at `auth/jwt/` with claims validation from `mcbeam-common-lib/auth/`
- **API Headers**: Custom wrapper at `mcbeam-common-lib/common/wrapper/apiheader/`
- **Metrics**: Prometheus integration via `mcbeam-common-lib/common/metrics/`

### Deployment Infrastructure

Deployment configurations and automation are maintained in the `go-micro-deploy` repository:
- **Fly.io Configurations**: Complete Fly.io deployment setup in `go-micro-deploy/fly.io/`
  - Infrastructure services (etcd, NATS, Consul) deployment scripts
  - Application service deployment configurations
  - Configuration migration tools for Apollo → Consul transition
  - Monitoring and validation scripts
- **Legacy Kubernetes**: K8s manifests in `go-micro-deploy/` root directory
- **Docker**: Multi-stage builds and container configurations

### Backward Compatibility Architecture

**Critical**: This version includes temporary compatibility code for gradual migration from older Go Micro versions.

1. **Dual Pool Strategy** (`client/rpc_client.go`):
   - Maintains two connection pools for protocol compatibility
   - Standard pool for HTTP transport
   - gRPC pool for gRPC transport compatibility

2. **Transport Protocol Detection**:
   - Automatic detection via service metadata (`transport` field)
   - Environment variable support: `MICRO_TRANSPORT=grpc` (old) and `MICRO_CLIENT=grpc` (new)
   - Fallback mechanisms for mixed version environments
   - Protocol negotiation on connection establishment

3. **Compatibility Code Locations**:
   - `client/rpc_client.go` - Dual pooling and protocol detection
   - `transport/grpc/grpc.go` - DefaultGrpcTransport initialization
   - `broker/nats/nats.go` - Message format compatibility
   - Files marked with `===== COMPATIBILITY:` comments indicate temporary code

4. **Migration Path**:
   - Phase 1: Compatibility mode with dual protocol support (current)
   - Phase 2: Gradual service-by-service updates
   - Phase 3: Remove compatibility code once all services migrated

## Key Development Patterns

**Options Pattern**: All components use functional options for configuration
```go
service := micro.NewService(
    micro.Name("service-name"),
    micro.Version("v1.0.0"),
    micro.Address(":8080"),
    micro.RegisterTTL(time.Second*30),
    micro.RegisterInterval(time.Second*10),
)
```

**Interface-based Design**: Core abstractions are interfaces enabling runtime pluggability

**Context Propagation**: Context flows through all layers carrying deadlines, cancellation signals, and metadata

**Wrapper/Middleware Pattern**: Cross-cutting concerns implemented as wrappers
```go
service.Server().Handle(
    service.Server().NewHandler(&Handler{}),
    server.InternalHandler(true),
)
```

**Codec System**: Dynamic message encoding based on content-type (application/json, application/protobuf, etc.)

## Important Implementation Notes

1. **gRPC Pool Optimization** (`client/grpc/grpc_pool.go`):
   - Stream multiplexing with configurable maxStreams per connection
   - Connection state monitoring (Ready/Idle/Connecting/TransientFailure/Shutdown)
   - Intelligent idle/busy connection management with TTL
   - Automatic reconnection with exponential backoff

2. **Service Discovery Defaults**:
   - Development: mDNS with automatic service announcement
   - Production: etcd with lease management and health checks
   - Service metadata includes version, endpoints, and transport protocol

3. **Error Handling**:
   - Use `errors` package for structured errors with status codes
   - Error types: BadRequest, Unauthorized, Forbidden, NotFound, Timeout, Conflict, InternalServerError
   - Errors are serialized with code, detail, and status for cross-service propagation

4. **Testing Infrastructure**:
   - Memory implementations for all components (`memory.go`, `*_memory.go` files)
   - Mock clients and servers in `client/mock/` and `server/mock/`
   - Test utilities in `util/test/`

5. **Performance Considerations**:
   - Connection pooling with configurable size and TTL
   - Client-side load balancing with multiple strategies
   - Request/response caching capabilities
   - Streaming support for large payloads

6. **Security Features**:
   - JWT authentication with RSA/HMAC support
   - TLS configuration for transport encryption
   - Service-to-service authentication via certificates
   - Rule-based access control in auth package

## Required Dependencies

- Go 1.23.10 or higher
- Access to private `zigo2048/mcbeam-common-lib` packages
- Protocol buffer compiler (protoc) version 3.x
- protoc-gen-go plugin
- protoc-gen-micro plugin (built from this repo)

## Environment Variables

Common configuration environment variables:
- `MICRO_REGISTRY`: Registry backend (mdns, etcd, consul, memory)
- `MICRO_BROKER`: Broker backend (http, nats, memory)
- `MICRO_TRANSPORT`: Transport protocol (http, grpc) - legacy
- `MICRO_CLIENT`: Client implementation (rpc, grpc) - new
- `MICRO_SERVER`: Server implementation (rpc, grpc) - new
- `MICRO_SELECTOR`: Load balancer selector (random, roundrobin, shard)
- `MICRO_CONFIG`: Config source (file, env, consul, etcd)

## Comments and Documentation

- All comments must be in English regardless of other language preferences
- Follow Go documentation conventions with package-level comments
- Protocol buffer definitions include comprehensive field documentation
- Interfaces should have detailed godoc explaining implementation requirements