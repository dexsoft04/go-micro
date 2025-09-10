# Go-Micro 测试套件

## 概述

本测试套件包含两个主要部分：
1. **跨版本兼容性测试** - 验证 go-micro 框架在 Kubernetes 环境下的跨版本兼容性
2. **RPC 耗时分析工具** - 基于 OpenTelemetry 的 RPC 请求性能监控和分析

## 测试文件结构

```
tests/
├── README.md                                          # 本文档
├── IMPLEMENTATION_SUMMARY.md                         # 实现总结
├── nats_message_test.go                              # 原有的 NATS 消息测试
├── basic_compatibility_test.go                       # 基础兼容性测试
├── compatibility_test.go                             # 主兼容性测试套件
├── cross_version_rpc_test.go                         # 跨版本 RPC 调用测试
├── cross_version_pubsub_test.go                      # 跨版本消息发布订阅测试
├── version_detection_test.go                         # 版本检测和协议协商测试
├── config_init_test.go                               # 配置初始化测试
├── helpers_test.go                                   # 测试辅助函数和工具
├── rpc_timing_test.go                                # RPC 耗时测试
├── timing_example.go                                 # 耗时监控配置示例
├── analyze_timing.sh                                 # 耗时日志分析脚本
├── timing_config.yaml                               # 耗时监控配置文件
└── postman/                                          # Postman 测试配置
    ├── README.md                                     # Postman 使用指南
    ├── go-micro-compatibility-tests.postman_collection.json    # 测试集合
    └── go-micro-compatibility-tests.postman_environment.json   # 环境配置
```

## 跨版本兼容性测试

主要测试新版本服务（mcbeam-overall-control-srv，使用本地修改的 go-micro）和老版本服务（mcbeam-game-center-srv，使用 v5.6.0-beta）之间的互操作性。

## 测试覆盖的关键场景

### 1. 基础兼容性测试 (`basic_compatibility_test.go`)

- **TestBasicCompatibility**: 测试不同环境变量配置下的服务创建
  - 旧版本配置：`MICRO_TRANSPORT=grpc`
  - 新版本配置：`MICRO_CLIENT=grpc`, `MICRO_SERVER=grpc`
  - 混合配置：新配置覆盖旧配置

- **TestVersionMetadata**: 验证版本元数据的正确设置和传递

- **TestConfigurationPriority**: 测试配置优先级处理

- **TestK8sEnvironmentHandling**: 验证 Kubernetes 环境变量处理

### 2. 服务发现兼容性测试 (`compatibility_test.go`)

- **TestServiceDiscoveryCompatibility**: 测试新旧版本服务相互发现
- **TestBrokerCompatibility**: 测试 NATS broker 在不同版本间的兼容性
- **TestK8sEnvironmentVariables**: 测试 Kubernetes 特定环境变量

### 3. RPC 调用兼容性测试 (`cross_version_rpc_test.go`)

- **TestCrossVersionRPCCall**: 测试跨版本 RPC 调用
  - 新客户端调用老服务
  - 老客户端调用新服务
- **TestProtocolNegotiation**: 测试协议自动协商
- **TestRPCWithMetadata**: 测试元数据传递
- **TestRetryAndBackoff**: 测试重试和退避机制
- **TestErrorPropagation**: 测试错误传递兼容性

### 4. 消息发布订阅测试 (`cross_version_pubsub_test.go`)

- **TestCrossVersionPubSub**: 老版本发布 → 新版本订阅
- **TestReverseVersionPubSub**: 新版本发布 → 老版本订阅
- **TestMessageFormatCompatibility**: 测试不同消息格式兼容性
- **TestQueueGroupCompatibility**: 测试队列组负载均衡
- **TestEventMetadataPropagation**: 测试事件元数据传递

### 5. 版本检测和协议协商 (`version_detection_test.go`)

- **TestVersionDetectionFromMetadata**: 从服务元数据检测版本
- **TestProtocolDetectionFromEnvironment**: 从环境变量检测协议
- **TestServiceCompatibilityMatrix**: 服务版本兼容性矩阵
- **TestVersionNegotiation**: 版本协商机制
- **TestBackwardCompatibilityFlags**: 向后兼容性标志

## 运行测试

### 运行所有测试
```bash
go test -v ./tests/
```

### 运行特定测试
```bash
# 基础兼容性测试
go test -v ./tests/ -run TestBasicCompatibility

# 版本元数据测试
go test -v ./tests/ -run TestVersionMetadata

# NATS 消息测试（需要 NATS 服务器）
NATS_URL=nats://localhost:4222 go test -v ./tests/ -run TestNATSMessage

# 跨版本 RPC 测试
go test -v ./tests/ -run TestCrossVersionRPC
```

### 带外部依赖的测试
```bash
# 需要 NATS 服务器
NATS_URL=nats://localhost:4222 go test -v ./tests/

# 需要 etcd
ETCD_ENDPOINTS=http://localhost:2379 go test -v ./tests/
```

## 环境变量配置

### 必需环境变量（测试会自动设置默认值）
- `MICRO_REGISTRY=memory` - 使用内存注册表（测试环境）
- `MICRO_BROKER=memory` - 使用内存消息代理（测试环境）

### 可选环境变量（连接外部服务）
- `NATS_URL` - NATS 服务器地址（如 nats://localhost:4222）
- `ETCD_ENDPOINTS` - etcd 服务器地址（如 http://localhost:2379）

### Kubernetes 环境变量
- `KUBERNETES_SERVICE_HOST` - K8s API 服务器地址
- `KUBERNETES_SERVICE_PORT` - K8s API 服务器端口
- `POD_NAME` - Pod 名称
- `POD_NAMESPACE` - Pod 命名空间
- `HOSTNAME` - 主机名

## 兼容性测试矩阵

| 场景 | 老版本 (v5.6.0-beta) | 新版本 (v5.6.0-local) | 状态 |
|------|---------------------|----------------------|------|
| 服务发现 | HTTP + mDNS/etcd | gRPC + etcd | ✅ 兼容 |
| RPC 调用 | HTTP 传输 | HTTP/gRPC 自动检测 | ✅ 兼容 |
| 消息传递 | NATS + JSON | NATS + JSON/Protobuf | ✅ 兼容 |
| 元数据 | 基础元数据 | 扩展元数据 | ✅ 兼容 |
| 配置 | MICRO_TRANSPORT | MICRO_CLIENT/SERVER | ✅ 兼容 |

## 测试辅助工具

### MockServiceVersion (`helpers_test.go`)
用于创建具有特定版本特征的模拟服务：
```go
oldService := CreateMockService(
    "test-service", "v1.0.0-beta", "v5.6.0-beta", "http",
    map[string]string{"generation": "old"},
)
```

### MessageCollector
用于收集和验证消息：
```go
collector := NewMessageCollector(10)
messages, err := collector.WaitForMessages(1, 5*time.Second)
```

### VersionCompatibilityTester
用于测试版本兼容性：
```go
tester := NewVersionCompatibilityTester("old-service", "new-service")
err := tester.TestCrossVersionMessage(topic, message)
```

## 性能基准

测试套件包含基础的性能验证：
- 服务创建性能
- 跨版本调用延迟
- 消息传递吞吐量

## 故障场景测试

- 网络分区恢复
- 服务重启
- 配置热更新
- 协议降级

## 生产环境建议

1. **渐进式升级**: 先升级一个服务实例，验证兼容性
2. **监控关键指标**: RPC 调用成功率、消息传递延迟
3. **回滚准备**: 保持老版本镜像以便快速回滚
4. **配置验证**: 确保新旧配置都能正确解析
5. **日志监控**: 关注兼容性相关的错误日志

## 注意事项

1. 测试默认使用内存实现（memory registry/broker），避免外部依赖
2. 需要外部服务的测试会自动跳过（如 NATS 未配置）
3. 某些测试可能需要特定的环境变量设置
4. K8s 相关测试会模拟 K8s 环境变量

## 扩展测试

要添加新的兼容性测试：

1. 在相应的测试文件中添加测试函数
2. 使用 helpers_test.go 中的辅助函数
3. 确保测试能处理外部服务不可用的情况
4. 更新本文档的测试覆盖说明

## 持续集成

建议在 CI/CD 流水线中：
1. 基础测试：每次提交都运行
2. 集成测试：需要外部服务的测试在专门的环境运行
3. 性能测试：定期运行，监控性能回归
4. 端到端测试：在实际 K8s 集群中验证

## RPC 耗时分析工具

基于 OpenTelemetry 集成的 RPC 请求性能监控解决方案，提供详细的耗时分析和性能优化建议。

### 核心实现
- `../wrapper/trace/opentelemetry/wrapper.go` - 增强的 OpenTelemetry wrappers，添加详细耗时日志
- `../client/rpc_client.go` - 客户端增加连接、网络传输等阶段性耗时记录

### 快速开始

#### 1. 配置应用程序
```go
import (
    "go-micro.dev/v5"
    log "go-micro.dev/v5/logger"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/stdout/stdouttrace" 
    "go.opentelemetry.io/otel/sdk/trace"
    otWrapper "go-micro.dev/v5/wrapper/trace/opentelemetry"
)

func main() {
    // 配置 OpenTelemetry
    exporter, _ := stdouttrace.New(stdouttrace.WithPrettyPrint())
    tp := trace.NewTracerProvider(trace.WithBatcher(exporter))
    otel.SetTracerProvider(tp)
    
    // 启用调试日志以查看详细耗时
    log.DefaultLogger = log.NewLogger(log.WithLevel(log.DebugLevel))
    
    // 创建带有耗时监控的服务
    service := micro.NewService(
        micro.Name("my-service"),
        micro.WrapClient(otWrapper.NewClientWrapper()),
        micro.WrapHandler(otWrapper.NewHandlerWrapper()),
    )
}
```

#### 2. 运行和收集日志
```bash
# 运行应用并保存日志
./your-app 2>&1 | tee app.log

# 实时监控耗时
tail -f app.log | grep RPC_TIMING
```

#### 3. 分析耗时数据
```bash
# 生成详细的耗时分析报告
./tests/analyze_timing.sh app.log

# 实时分析
tail -f app.log | ./tests/analyze_timing.sh
```

### 日志格式

#### 客户端耗时
```
RPC_TIMING: service=user-service endpoint=GetUser node=127.0.0.1:8080 duration_ms=45 trace_id=abc123... span_id=def456...
```

#### 服务端耗时  
```
RPC_TIMING_SERVER: service=user-service endpoint=GetUser duration_ms=12 trace_id=abc123... span_id=def456...
```

#### 详细耗时（调试级别）
```
RPC_TIMING_DETAIL: connection_ms=5 service=user-service endpoint=GetUser address=127.0.0.1:8080
RPC_TIMING_DETAIL: send_ms=2 recv_ms=8 service=user-service endpoint=GetUser
```

### 分析功能

`analyze_timing.sh` 脚本提供：

- **统计摘要**: 平均、最小、最大、百分位数（50th, 90th, 95th, 99th）
- **按服务分析**: 每个服务和端点的性能分析
- **慢请求检测**: 识别超过阈值的请求
- **详细耗时分析**: 连接和网络传输耗时分解
- **实时监控**: 支持实时日志分析

### 运行 RPC 耗时测试

```bash
# 运行耗时功能测试
go test ./tests -v -run TestRPCTiming

# 运行性能基准测试（测量 wrapper 开销）
go test ./tests -bench=BenchmarkTimingWrapper
```

### 配置示例

参考 `timing_config.yaml` 了解不同环境的配置：

- **开发环境**: 详细调试日志 + stdout 导出
- **生产环境**: 均衡日志 + Jaeger/Tempo 导出  
- **测试环境**: 详细分析 + 低阈值
- **性能环境**: 最小开销 + 采样

### 性能影响

耗时监控的性能开销很小：
- 每个 RPC 调用约 0.1-0.5ms 的日志操作开销
- 内存开销可忽略
- 生产环境可通过 OpenTelemetry 采样降低开销

### 集成监控系统

耗时日志可以集成到各种监控系统：
- **Prometheus**: 解析日志暴露指标
- **Grafana**: 数据可视化和告警
- **ELK Stack**: 使用 Logstash 解析结构化日志
- **Jaeger/Tempo**: 导出 OpenTelemetry 追踪数据

这个测试套件确保了 go-micro 框架在版本升级过程中的稳定性和兼容性，同时提供了完整的 RPC 性能监控解决方案，为生产环境的安全升级和性能优化提供了充分的验证和工具支持。