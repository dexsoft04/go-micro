#!/bin/bash

echo "=== 演示三种文件格式的区别 ==="
echo

# 创建测试目录
mkdir -p /tmp/format-test

echo "1. Simple 格式（专为性能分析优化）"
echo "export MICRO_TRACING_FILE_FORMAT=simple"
echo "特点：简化字段、便于解析、文件小"
echo

cat > /tmp/format-test/simple-format.jsonl << 'EOF'
{"timestamp":"2024-01-20T10:00:01Z","service":"user-service","operation":"GetUser","duration_ms":23,"trace_id":"abc123","span_id":"def456","status":"ok","attributes":{"http.method":"GET","user.id":"123"}}
{"timestamp":"2024-01-20T10:00:02Z","service":"order-service","operation":"CreateOrder","duration_ms":156,"trace_id":"abc124","span_id":"def457","status":"error","attributes":{"http.method":"POST","order.amount":99.99},"error":"validation failed"}
EOF

echo "示例内容："
cat /tmp/format-test/simple-format.jsonl | head -1 | jq .
echo

echo "2. JSON 格式（标准 OpenTelemetry 美化格式）"
echo "export MICRO_TRACING_FILE_FORMAT=json"
echo "特点：标准字段、完整信息、可读性好"
echo

cat > /tmp/format-test/json-format.jsonl << 'EOF'
{
  "Name": "user-service.GetUser",
  "SpanContext": {
    "TraceID": "7f8a9b2c3d4e5f6789abcdef01234567",
    "SpanID": "123456789abcdef0"
  },
  "StartTime": "2024-01-20T10:00:01.000000Z",
  "EndTime": "2024-01-20T10:00:01.023000Z",
  "Status": {
    "Code": "Ok"
  },
  "Attributes": [
    {
      "Key": "http.method",
      "Value": {
        "Type": "STRING",
        "Value": "GET"
      }
    },
    {
      "Key": "user.id",
      "Value": {
        "Type": "STRING",
        "Value": "123"
      }
    }
  ]
}
EOF

echo "示例内容："
cat /tmp/format-test/json-format.jsonl | jq .
echo

echo "3. Standard 格式（标准 OpenTelemetry 紧凑格式）"
echo "export MICRO_TRACING_FILE_FORMAT=standard 或不设置"
echo "特点：完整信息、紧凑存储、标准兼容"
echo

cat > /tmp/format-test/standard-format.jsonl << 'EOF'
{"Name":"user-service.GetUser","SpanContext":{"TraceID":"7f8a9b2c3d4e5f6789abcdef01234567","SpanID":"123456789abcdef0"},"StartTime":"2024-01-20T10:00:01.000000Z","EndTime":"2024-01-20T10:00:01.023000Z","Status":{"Code":"Ok"},"Attributes":[{"Key":"http.method","Value":{"Type":"STRING","Value":"GET"}},{"Key":"user.id","Value":{"Type":"STRING","Value":"123"}}]}
EOF

echo "示例内容："
cat /tmp/format-test/standard-format.jsonl | jq .
echo

echo "4. 性能分析工具兼容性："
echo

echo "Simple 格式 - 直接支持："
if command -v go &> /dev/null; then
    go run cmd/trace-analyzer/main.go -input /tmp/format-test/simple-format.jsonl 2>/dev/null || echo "（需要服务运行生成真实数据）"
fi
echo

echo "JSON/Standard 格式 - 需要格式转换："
echo "可以使用 jq 等工具进行转换，或者使用 Simple 格式进行性能分析"
echo

echo "5. 使用建议："
echo "- 开发调试：使用 json 格式（便于阅读）"
echo "- 性能分析：使用 simple 格式（便于解析）"
echo "- 生产环境：使用 standard 格式（节省空间）或直接用 Jaeger"
echo

echo "6. 文件大小对比："
echo "Simple 格式："
wc -c /tmp/format-test/simple-format.jsonl | awk '{print $1 " bytes"}'

echo "JSON 格式："
wc -c /tmp/format-test/json-format.jsonl | awk '{print $1 " bytes"}'

echo "Standard 格式："
wc -c /tmp/format-test/standard-format.jsonl | awk '{print $1 " bytes"}'

echo
echo "清理测试文件..."
rm -rf /tmp/format-test
echo "演示完成！"