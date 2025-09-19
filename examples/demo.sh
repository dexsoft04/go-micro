#!/bin/bash

echo "=== Go-Micro 轻量级 Trace 性能监控演示 ==="
echo

# 清理之前的数据
rm -f /tmp/demo-traces.jsonl

# 演示不同的 exporter 模式
echo "1. 演示 stdout 模式（实时输出）："
echo "export MICRO_TRACING_REPORTER_ADDRESS=stdout"
echo

echo "2. 演示文件模式（简化 JSON 格式）："
echo "export MICRO_TRACING_REPORTER_ADDRESS=file:///tmp/demo-traces.jsonl"
echo "export MICRO_TRACING_FILE_FORMAT=simple"
echo

echo "3. 演示 Jaeger 模式："
echo "export MICRO_TRACING_REPORTER_ADDRESS=http://jaeger:14268/api/traces"
echo

echo "4. 创建示例 trace 数据..."

# 创建一些示例 trace 数据
cat > /tmp/demo-traces.jsonl << 'EOF'
{"timestamp":"2024-01-20T10:00:01Z","service":"user-service","operation":"GetUser","duration_ms":23,"trace_id":"abc123","span_id":"def456","status":"ok","attributes":{"http.method":"GET","user.id":"123"}}
{"timestamp":"2024-01-20T10:00:02Z","service":"user-service","operation":"GetUser","duration_ms":45,"trace_id":"abc124","span_id":"def457","status":"ok","attributes":{"http.method":"GET","user.id":"456"}}
{"timestamp":"2024-01-20T10:00:03Z","service":"order-service","operation":"CreateOrder","duration_ms":156,"trace_id":"abc125","span_id":"def458","status":"ok","attributes":{"http.method":"POST","order.amount":99.99}}
{"timestamp":"2024-01-20T10:00:04Z","service":"order-service","operation":"CreateOrder","duration_ms":890,"trace_id":"abc126","span_id":"def459","status":"error","attributes":{"http.method":"POST","order.amount":199.99},"error":"database timeout"}
{"timestamp":"2024-01-20T10:00:05Z","service":"payment-service","operation":"ProcessPayment","duration_ms":120,"trace_id":"abc127","span_id":"def460","status":"ok","attributes":{"http.method":"POST","payment.amount":99.99}}
{"timestamp":"2024-01-20T10:00:06Z","service":"user-service","operation":"GetUser","duration_ms":78,"trace_id":"abc128","span_id":"def461","status":"ok","attributes":{"http.method":"GET","user.id":"789"}}
{"timestamp":"2024-01-20T10:00:07Z","service":"order-service","operation":"CreateOrder","duration_ms":234,"trace_id":"abc129","span_id":"def462","status":"ok","attributes":{"http.method":"POST","order.amount":49.99}}
EOF

echo "5. 使用 trace-analyzer 分析性能数据："
echo

echo "基础分析（表格格式）："
echo "go run cmd/trace-analyzer/main.go -input /tmp/demo-traces.jsonl"
echo

if command -v go &> /dev/null; then
    go run cmd/trace-analyzer/main.go -input /tmp/demo-traces.jsonl
    echo
fi

echo "过滤慢请求（>100ms）："
echo "go run cmd/trace-analyzer/main.go -input /tmp/demo-traces.jsonl -min-duration 100"
echo

if command -v go &> /dev/null; then
    go run cmd/trace-analyzer/main.go -input /tmp/demo-traces.jsonl -min-duration 100
    echo
fi

echo "只显示错误："
echo "go run cmd/trace-analyzer/main.go -input /tmp/demo-traces.jsonl -errors"
echo

if command -v go &> /dev/null; then
    go run cmd/trace-analyzer/main.go -input /tmp/demo-traces.jsonl -errors
    echo
fi

echo "生成 HTML 报告："
echo "go run cmd/trace-analyzer/main.go -input /tmp/demo-traces.jsonl -format html -output /tmp/report.html"
echo

if command -v go &> /dev/null; then
    go run cmd/trace-analyzer/main.go -input /tmp/demo-traces.jsonl -format html -output /tmp/report.html
    echo "HTML 报告已生成: /tmp/report.html"
    echo
fi

echo "6. 环境变量配置示例："
echo
echo "# 开发环境"
echo "export MICRO_TRACING_REPORTER_ADDRESS=file:///tmp/dev-traces.jsonl"
echo "export MICRO_TRACING_FILE_FORMAT=simple"
echo
echo "# 测试环境"
echo "export MICRO_TRACING_REPORTER_ADDRESS=stdout"
echo
echo "# 生产环境"
echo "export MICRO_TRACING_REPORTER_ADDRESS=http://jaeger.prod:14268/api/traces"
echo
echo "# 禁用追踪"
echo "export MICRO_TRACING_REPORTER_ADDRESS=\"\""
echo

echo "7. 实时分析命令："
echo
echo "# 监控实时性能"
echo "tail -f /var/log/traces.jsonl | go run cmd/trace-analyzer/main.go -stream"
echo
echo "# 定期性能报告"
echo "watch -n 30 \"tail -100 /var/log/traces.jsonl | go run cmd/trace-analyzer/main.go -stream -top 5\""
echo

echo "演示完成！查看文档了解更多："
echo "docs/trace-performance-monitoring.md"