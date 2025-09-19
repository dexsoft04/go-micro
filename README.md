# Go Micro [![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/go-micro.dev/v5?tab=doc) [![Go Report Card](https://goreportcard.com/badge/github.com/go-micro/go-micro)](https://goreportcard.com/report/github.com/go-micro/go-micro) 

Go Micro is a framework for distributed systems development.

## Overview

Go Micro provides the core requirements for distributed systems development including RPC and Event driven communication.
The Go Micro philosophy is sane defaults with a pluggable architecture. We provide defaults to get you started quickly
but everything can be easily swapped out.

## Features

Go Micro abstracts away the details of distributed systems. Here are the main features.

- **Authentication** - Auth is built in as a first class citizen. Authentication and authorization enable secure
  zero trust networking by providing every service an identity and certificates. This additionally includes rule
  based access control.

- **Dynamic Config** - Load and hot reload dynamic config from anywhere. The config interface provides a way to load application
  level config from any source such as env vars, file, etcd. You can merge the sources and even define fallbacks.

- **Data Storage** - A simple data store interface to read, write and delete records. It includes support for many storage backends
in the plugins repo. State and persistence becomes a core requirement beyond prototyping and Micro looks to build that into the framework.

- **Service Discovery** - Automatic service registration and name resolution. Service discovery is at the core of micro service
  development. When service A needs to speak to service B it needs the location of that service. The default discovery mechanism is
  multicast DNS (mdns), a zeroconf system.

- **Load Balancing** - Client side load balancing built on service discovery. Once we have the addresses of any number of instances
  of a service we now need a way to decide which node to route to. We use random hashed load balancing to provide even distribution
  across the services and retry a different node if there's a problem.

- **Message Encoding** - Dynamic message encoding based on content-type. The client and server will use codecs along with content-type
  to seamlessly encode and decode Go types for you. Any variety of messages could be encoded and sent from different clients. The client
  and server handle this by default. This includes protobuf and json by default.

- **RPC Client/Server** - RPC based request/response with support for bidirectional streaming. We provide an abstraction for synchronous
  communication. A request made to a service will be automatically resolved, load balanced, dialled and streamed.

- **Async Messaging** - PubSub is built in as a first class citizen for asynchronous communication and event driven architectures.
  Event notifications are a core pattern in micro service development. The default messaging system is a HTTP event message broker.

- **Pluggable Interfaces** - Go Micro makes use of Go interfaces for each distributed system abstraction. Because of this these interfaces
  are pluggable and allows Go Micro to be runtime agnostic. You can plugin any underlying technology.

## Getting Started

To make use of Go Micro 

```golang
go get "go-micro.dev/v5"
```

Create a service and register a handler

```golang
package main

import (
        "go-micro.dev/v5"
)

type Request struct {
        Name string `json:"name"`
}

type Response struct {
        Message string `json:"message"`
}

type Say struct{}

func (h *Say) Hello(ctx context.Context, req *Request, rsp *Response) error {
        rsp.Message = "Hello " + req.Name
        return nil
}

func main() {
        // create the service
        service := micro.New("helloworld")

        // register handler
        service.Handle(new(Say))

        // run the service
        service.Run()
}
```

Set a fixed address

```golang
service := micro.NewService(
    micro.Name("helloworld"),
    micro.Address(":8080"),
)
```

Call it via curl

```
curl -XPOST \
     -H 'Content-Type: application/json' \
     -H 'Micro-Endpoint: Say.Hello' \
     -d '{"name": "alice"}' \
      http://localhost:8080
```

## Toolkit

Once you've written a service you'll want to run, query and manage it. This is where the [micro](https://github.com/micro/micro) CLI can offer some value. Check it out.

## Backward Compatibility Notice

**⚠️ IMPORTANT: This version includes compatibility code for gradual service migration.**

### gRPC Compatibility

This version maintains backward compatibility with services configured using `MICRO_TRANSPORT=grpc`. The following compatibility features are implemented:

1. **Automatic Protocol Detection**: Services can automatically detect whether to use HTTP or gRPC based on:
   - Service metadata (`transport` field)
   - Cached transport information
   - Automatic fallback on HTTP connection errors

2. **Mixed Environment Support**: You can safely run both old and new services in the same environment:
   - Old services: Use `MICRO_TRANSPORT=grpc`
   - New services: Use `MICRO_CLIENT=grpc` and `MICRO_SERVER=grpc` (recommended)

3. **Gradual Migration Path**: Update services one by one without breaking the entire system.

### Migration Timeline

- **Phase 1**: Use compatibility mode to ensure all services work together
- **Phase 2**: Gradually update services to use upstream gRPC implementations
- **Phase 3**: Remove compatibility code after all services are updated

### Compatibility Code Locations

The following files contain compatibility code (marked with `===== COMPATIBILITY:`):
- `client/rpc_client.go`: gRPC connection pooling and automatic protocol detection
- `transport/grpc/grpc.go`: DefaultGrpcTransport initialization

**TODO**: Remove all compatibility code after complete migration to maintain clean codebase.

## 相关文档

### 开发运维指南
- [SSH 隧道完全指南](docs/ssh-tunnel-guide.md) - SSH 隧道基础概念、实战案例、工具集成、故障排查等完整知识体系

### 性能测试与压测
- [压测基础概念指南](docs/load-testing-guide.md) - 压测类型、关键指标、微服务压测注意事项等基础知识
- [流量捕获与回放指南](docs/traffic-capture-guide.md) - 使用 HAR、mitmproxy、tshark 等工具捕获真实流量并进行回放测试
- [K6 压测实战指南](docs/k6-testing-guide.md) - 使用 K6 进行 HTTP API 和 WebSocket 压测的完整指南
- [gRPC 服务压测指南](docs/grpc-testing-guide.md) - 使用 ghz 工具对 gRPC 微服务进行性能测试
- [性能监控与分析指南](docs/monitoring-analysis-guide.md) - Prometheus + Grafana 监控搭建、pprof 性能分析、链路追踪等
- [压测脚本工具集](docs/testing-scripts/README.md) - 开箱即用的压测脚本，包括 K6、ghz、流量回放等工具

### 性能优化指南
- [高并发优化计划](docs/high-concurrency-optimization-plan.md) - 详细的性能优化实施计划，包括连接池优化、内存管理、服务发现缓存等改进方案

### 版本兼容性测试
- [快速开始](tests/integration/QUICK_START.md) - 5分钟快速验证新旧版本兼容性
- [完整测试指南](tests/integration/TESTING_GUIDE.md) - 详细的兼容性测试文档，包含环境准备、测试执行、结果分析等完整流程
- [集成测试套件](tests/integration/README.md) - 自动化测试脚本说明，涵盖 RPC、事件发布订阅、性能和错误恢复等测试场景
- [Postman API 测试](tests/integration/postman/README.md) - 使用 Postman 进行 API 兼容性测试的配置和使用指南

## Experimental

There's a new `genai` package for generative AI capabilities. This is an evolving feature which may change over time as we think about how Go Micro plays the right role in the developers workflow. We'll also be looking at agentic features and a2a/mcp protocol integration.

## Adopters

- [Sourse](https://sourse.eu) - Work in the field of earth observation, including embedded Kubernetes running onboard aircraft, and we’ve built a mission management SaaS platform using Go Micro.
