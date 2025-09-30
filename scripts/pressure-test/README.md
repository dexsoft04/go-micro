# Go-Micro 压力测试工具集

轻量级分布式链路性能分析解决方案，无需部署 Jaeger 等重型基础设施。

## 快速开始

### 1. 一键环境初始化
```bash
./setup.sh
```

### 2. 启动服务并配置追踪
```bash
# 加载测试环境变量
source /tmp/pressure-test-traces/load-env.sh user-service

# 启动服务
go run ./your-service/main.go
```

### 3. 执行压力测试
```bash
# 中等强度测试
./run-test.sh medium http://localhost:8080

# 自定义参数测试
./run-test.sh heavy http://localhost:8080 120s 16 1000
```

### 4. 实时监控
```bash
# 终端监控
./monitor.sh

# Web 大屏监控
open dashboard.html
```

### 5. 分析报告
```bash
# 生成分析报告
./analyze.sh

# 使用专用分析工具
go run ../../cmd/trace-analyzer/main.go -input /tmp/pressure-test-traces/service.jsonl -format pressure
```

## 工具文件说明

| 文件 | 功能 | 用途 |
|------|------|------|
| `setup.sh` | 环境初始化 | 创建目录、生成配置、检查依赖 |
| `run-test.sh` | 压力测试执行 | 支持 wrk/ab/自定义客户端 |
| `monitor.sh` | 实时监控 | 终端界面实时性能监控 |
| `analyze.sh` | 结果分析 | 生成详细的性能分析报告 |
| `dashboard.html` | 监控大屏 | Web 界面可视化监控 |
| `configs/` | 配置文件 | 测试场景、阈值等配置 |

## 预设测试场景

| 场景 | 持续时间 | 线程数 | 连接数 | 适用场景 |
|------|----------|--------|--------|----------|
| `light` | 30s | 4 | 50 | 轻量级测试 |
| `medium` | 60s | 8 | 200 | 日常性能验证 |
| `heavy` | 120s | 12 | 500 | 压力测试 |
| `stress` | 300s | 16 | 1000 | 极限压力测试 |

## 输出目录结构

```
/tmp/pressure-test-traces/
├── service-a-instance-1.jsonl    # 服务A实例1的追踪数据
├── service-b-instance-1.jsonl    # 服务B实例1的追踪数据
└── ...

/tmp/pressure-test-logs/
├── test-20240115-143025/          # 测试会话目录
│   ├── wrk-output.log            # 负载测试结果
│   ├── trace-analysis.txt        # 追踪分析结果
│   ├── trace-report.html         # HTML 分析报告
│   ├── monitor.log               # 监控日志
│   └── test-report.md            # 测试总结报告
└── ...
```

## 快速问题排查

### 追踪数据为空
```bash
# 检查环境变量
env | grep MICRO_

# 重新加载环境
source /tmp/pressure-test-traces/load-env.sh your-service
```

### 监控显示 "No data"
```bash
# 检查追踪文件
ls -la /tmp/pressure-test-traces/*.jsonl

# 生成测试数据
/tmp/pressure-test-traces/generate-test-data.sh
```

### 分析工具编译失败
```bash
# 更新依赖
cd ../..
go mod tidy
go build -o bin/trace-analyzer ./cmd/trace-analyzer
```

## 更多信息

详细文档请查看：[Go-Micro 压力测试指南](../../docs/pressure-testing-guide.md)