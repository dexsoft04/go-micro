package tests

import (
	"bytes"
	"context"
	"testing"
	"time"

	"go-micro.dev/v5/broker"
	natsBroker "go-micro.dev/v5/broker/nats"
	"go-micro.dev/v5/codec/proto"
)

// TestNATSMessagePassthrough 测试 NATS broker 是否正确透传消息
func TestNATSMessagePassthrough(t *testing.T) {
	// 创建 NATS broker 实例
	b := natsBroker.NewNatsBroker()

	// 连接
	if err := b.Connect(); err != nil {
		t.Skip("NATS server not available, skipping test")
	}
	defer b.Disconnect()

	topic := "test.message.passthrough"
	
	// 创建一个测试消息，包含特定的 Content-Type
	testMessage := &broker.Message{
		Header: map[string]string{
			"Content-Type": "application/protobuf",
			"Test-Header":  "test-value",
		},
		Body: []byte("test message body with protobuf content"),
	}

	// 用于接收消息的通道
	msgChan := make(chan *broker.Message, 1)

	// 订阅处理器
	handler := func(p broker.Event) error {
		msg := p.Message()
		msgChan <- msg
		return nil
	}

	// 订阅主题
	sub, err := b.Subscribe(topic, handler)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	// 等待订阅生效
	time.Sleep(100 * time.Millisecond)

	// 发布消息
	if err := b.Publish(topic, testMessage); err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	// 等待消息接收
	select {
	case receivedMsg := <-msgChan:
		// 验证 Content-Type 是否正确透传
		if receivedMsg.Header["Content-Type"] != "application/protobuf" {
			t.Errorf("Expected Content-Type 'application/protobuf', got '%s'", receivedMsg.Header["Content-Type"])
		}

		// 验证自定义 header 是否正确透传
		if receivedMsg.Header["Test-Header"] != "test-value" {
			t.Errorf("Expected Test-Header 'test-value', got '%s'", receivedMsg.Header["Test-Header"])
		}

		// 验证消息体是否正确透传
		if !bytes.Equal(receivedMsg.Body, testMessage.Body) {
			t.Errorf("Message body mismatch. Expected: %s, Got: %s", testMessage.Body, receivedMsg.Body)
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

// TestNATSMessageWithDifferentContentTypes 测试不同 Content-Type 的消息处理
func TestNATSMessageWithDifferentContentTypes(t *testing.T) {
	// 创建 NATS broker 实例
	b := natsBroker.NewNatsBroker()

	// 连接
	if err := b.Connect(); err != nil {
		t.Skip("NATS server not available, skipping test")
	}
	defer b.Disconnect()

	testCases := []struct {
		contentType string
		body        []byte
	}{
		{"application/json", []byte(`{"test": "json"}`)},
		{"application/protobuf", []byte("protobuf binary data")},
		{"application/octet-stream", []byte("raw binary data")},
	}

	for _, tc := range testCases {
		t.Run(tc.contentType, func(t *testing.T) {
			topic := "test.content.type." + tc.contentType

			// 创建测试消息
			testMessage := &broker.Message{
				Header: map[string]string{
					"Content-Type": tc.contentType,
				},
				Body: tc.body,
			}

			// 用于接收消息的通道
			msgChan := make(chan *broker.Message, 1)

			// 订阅处理器
			handler := func(p broker.Event) error {
				msg := p.Message()
				msgChan <- msg
				return nil
			}

			// 订阅主题
			sub, err := b.Subscribe(topic, handler)
			if err != nil {
				t.Fatalf("Failed to subscribe: %v", err)
			}
			defer sub.Unsubscribe()

			// 等待订阅生效
			time.Sleep(100 * time.Millisecond)

			// 发布消息
			if err := b.Publish(topic, testMessage); err != nil {
				t.Fatalf("Failed to publish: %v", err)
			}

			// 等待消息接收
			select {
			case receivedMsg := <-msgChan:
				// 验证 Content-Type 是否正确透传
				if receivedMsg.Header["Content-Type"] != tc.contentType {
					t.Errorf("Expected Content-Type '%s', got '%s'", tc.contentType, receivedMsg.Header["Content-Type"])
				}

				// 验证消息体是否正确透传
				if !bytes.Equal(receivedMsg.Body, tc.body) {
					t.Errorf("Message body mismatch. Expected: %s, Got: %s", tc.body, receivedMsg.Body)
				}

			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for message")
			}
		})
	}
}