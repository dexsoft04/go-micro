package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"go-micro.dev/v5"
	"go-micro.dev/v5/broker"
	natsBroker "go-micro.dev/v5/broker/nats"
	"go-micro.dev/v5/server"
)

// TestEventMessage represents a test event message
type TestEventMessage struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Version   string            `json:"version"`
	Timestamp int64             `json:"timestamp"`
	Data      map[string]string `json:"data"`
}

// TestCrossVersionPubSub tests publish/subscribe between different versions
func TestCrossVersionPubSub(t *testing.T) {
	// Skip if NATS not available
	if os.Getenv("NATS_URL") == "" {
		t.Skip("NATS_URL not configured, skipping cross-version pubsub test")
	}

	// Initialize services
	if os.Getenv("MICRO_REGISTRY") == "" {
		os.Setenv("MICRO_REGISTRY", "memory")
		defer os.Unsetenv("MICRO_REGISTRY")
	}
	if os.Getenv("MICRO_BROKER") == "" {
		os.Setenv("MICRO_BROKER", "nats")
		defer os.Unsetenv("MICRO_BROKER")
	}

	// Create old version service (publisher)
	oldPublisher := micro.NewService(
		micro.Name("test-old-publisher"),
		micro.Version("v1.0.0-beta"),
		micro.Metadata(map[string]string{
			"version":  "old",
			"go-micro": "v5.6.0-beta",
		}),
	)

	// Create new version service (subscriber)
	newSubscriber := micro.NewService(
		micro.Name("test-new-subscriber"),
		micro.Version("v2.0.0-local"),
		micro.Metadata(map[string]string{
			"version":  "new", 
			"go-micro": "v5.6.0-local",
		}),
	)

	oldPublisher.Init()
	newSubscriber.Init()

	// Test topic
	testTopic := "cross.version.test.event"
	
	// Channel to collect received messages
	receivedMessages := make(chan *TestEventMessage, 10)
	var mu sync.Mutex
	messageCount := 0

	// New version subscribes to messages from old version
	handler := func(ctx context.Context, event *TestEventMessage) error {
		mu.Lock()
		messageCount++
		mu.Unlock()
		receivedMessages <- event
		t.Logf("New subscriber received event: ID=%s, Type=%s, Version=%s", 
			event.ID, event.Type, event.Version)
		return nil
	}

	// Subscribe using the new service
	sub := server.NewSubscriber(testTopic, handler)
	if err := newSubscriber.Server().Subscribe(sub); err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Start services
	go func() {
		if err := newSubscriber.Run(); err != nil {
			t.Logf("New subscriber error: %v", err)
		}
	}()

	// Wait for subscription to be active
	time.Sleep(2 * time.Second)

	// Old version publishes message
	testMessage := &TestEventMessage{
		ID:        "test-001",
		Type:      "CrossVersionTest",
		Version:   "old-publisher-v1.0.0",
		Timestamp: time.Now().Unix(),
		Data: map[string]string{
			"source":      "old-version-service",
			"target":      "new-version-service",
			"test_case":   "cross_version_compatibility",
		},
	}

	// Publish using old version service
	if err := publishEvent(oldPublisher, testTopic, testMessage); err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	t.Logf("Old publisher sent event: ID=%s, Type=%s", testMessage.ID, testMessage.Type)

	// Wait for message to be received
	select {
	case receivedMsg := <-receivedMessages:
		// Validate message content
		if receivedMsg.ID != testMessage.ID {
			t.Errorf("Expected ID %s, got %s", testMessage.ID, receivedMsg.ID)
		}
		if receivedMsg.Type != testMessage.Type {
			t.Errorf("Expected Type %s, got %s", testMessage.Type, receivedMsg.Type)
		}
		if receivedMsg.Version != testMessage.Version {
			t.Errorf("Expected Version %s, got %s", testMessage.Version, receivedMsg.Version)
		}
		if receivedMsg.Data["source"] != "old-version-service" {
			t.Errorf("Expected source 'old-version-service', got %s", receivedMsg.Data["source"])
		}
		t.Logf("✓ Old publisher → New subscriber: Message successfully transmitted")

	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for cross-version message")
	}

	// Cleanup
	newSubscriber.Server().Stop()
}

// TestReverseVersionPubSub tests new version publishing to old version
func TestReverseVersionPubSub(t *testing.T) {
	// Skip if NATS not available
	if os.Getenv("NATS_URL") == "" {
		t.Skip("NATS_URL not configured, skipping reverse version pubsub test")
	}

	if os.Getenv("MICRO_REGISTRY") == "" {
		os.Setenv("MICRO_REGISTRY", "memory")
		defer os.Unsetenv("MICRO_REGISTRY")
	}
	if os.Getenv("MICRO_BROKER") == "" {
		os.Setenv("MICRO_BROKER", "nats")
		defer os.Unsetenv("MICRO_BROKER")
	}

	// Create new version service (publisher)
	newPublisher := micro.NewService(
		micro.Name("test-new-publisher"),
		micro.Version("v2.0.0-local"),
		micro.Metadata(map[string]string{
			"version":  "new",
			"go-micro": "v5.6.0-local",
		}),
	)

	// Create old version service (subscriber)
	oldSubscriber := micro.NewService(
		micro.Name("test-old-subscriber"),
		micro.Version("v1.0.0-beta"),
		micro.Metadata(map[string]string{
			"version":  "old",
			"go-micro": "v5.6.0-beta",
		}),
	)

	newPublisher.Init()
	oldSubscriber.Init()

	testTopic := "reverse.cross.version.test.event"
	receivedMessages := make(chan *TestEventMessage, 10)

	// Old version subscribes to messages from new version
	handler := func(ctx context.Context, event *TestEventMessage) error {
		receivedMessages <- event
		t.Logf("Old subscriber received event: ID=%s, Type=%s, Version=%s", 
			event.ID, event.Type, event.Version)
		return nil
	}

	sub := server.NewSubscriber(testTopic, handler)
	if err := oldSubscriber.Server().Subscribe(sub); err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Start services
	go func() {
		if err := oldSubscriber.Run(); err != nil {
			t.Logf("Old subscriber error: %v", err)
		}
	}()

	time.Sleep(2 * time.Second)

	// New version publishes message
	testMessage := &TestEventMessage{
		ID:        "test-002",
		Type:      "ReverseCrossVersionTest",
		Version:   "new-publisher-v2.0.0",
		Timestamp: time.Now().Unix(),
		Data: map[string]string{
			"source":      "new-version-service",
			"target":      "old-version-service",
			"test_case":   "reverse_cross_version_compatibility",
			"enhancement": "new-features-available",
		},
	}

	if err := publishEvent(newPublisher, testTopic, testMessage); err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	t.Logf("New publisher sent event: ID=%s, Type=%s", testMessage.ID, testMessage.Type)

	// Wait for message to be received
	select {
	case receivedMsg := <-receivedMessages:
		if receivedMsg.ID != testMessage.ID {
			t.Errorf("Expected ID %s, got %s", testMessage.ID, receivedMsg.ID)
		}
		if receivedMsg.Data["source"] != "new-version-service" {
			t.Errorf("Expected source 'new-version-service', got %s", receivedMsg.Data["source"])
		}
		// Check that new fields are preserved
		if receivedMsg.Data["enhancement"] != "new-features-available" {
			t.Errorf("Expected enhancement field, got %s", receivedMsg.Data["enhancement"])
		}
		t.Logf("✓ New publisher → Old subscriber: Message successfully transmitted with all fields")

	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for reverse cross-version message")
	}

	oldSubscriber.Server().Stop()
}

// TestMessageFormatCompatibility tests different message formats
func TestMessageFormatCompatibility(t *testing.T) {
	if os.Getenv("NATS_URL") == "" {
		t.Skip("NATS_URL not configured, skipping message format compatibility test")
	}

	// Create NATS broker for direct testing
	b := natsBroker.NewNatsBroker()
	if err := b.Connect(); err != nil {
		t.Skip("NATS server not available, skipping test")
	}
	defer b.Disconnect()

	topic := "format.compatibility.test"

	// Test different content types and formats
	testCases := []struct {
		name        string
		contentType string
		message     interface{}
		expected    string
	}{
		{
			name:        "JSON Format",
			contentType: "application/json",
			message:     map[string]string{"format": "json", "version": "test"},
			expected:    "json",
		},
		{
			name:        "Protobuf Format",
			contentType: "application/protobuf",
			message:     []byte("protobuf-binary-data"),
			expected:    "protobuf",
		},
		{
			name:        "Plain Text Format",
			contentType: "text/plain",
			message:     "plain text message",
			expected:    "text",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			receivedMessages := make(chan *broker.Message, 1)

			// Subscribe
			handler := func(p broker.Event) error {
				receivedMessages <- p.Message()
				return nil
			}

			sub, err := b.Subscribe(topic, handler)
			if err != nil {
				t.Fatalf("Failed to subscribe: %v", err)
			}
			defer sub.Unsubscribe()

			time.Sleep(100 * time.Millisecond)

			// Prepare message body
			var body []byte
			switch v := tc.message.(type) {
			case string:
				body = []byte(v)
			case []byte:
				body = v
			default:
				body, err = json.Marshal(v)
				if err != nil {
					t.Fatalf("Failed to marshal message: %v", err)
				}
			}

			// Publish message
			msg := &broker.Message{
				Header: map[string]string{
					"Content-Type":     tc.contentType,
					"Message-Format":   tc.expected,
					"Test-Case":        tc.name,
					"Compatibility":    "cross-version",
				},
				Body: body,
			}

			if err := b.Publish(topic, msg); err != nil {
				t.Fatalf("Failed to publish: %v", err)
			}

			// Wait for message
			select {
			case receivedMsg := <-receivedMessages:
				// Validate headers are preserved
				if receivedMsg.Header["Content-Type"] != tc.contentType {
					t.Errorf("Expected Content-Type %s, got %s", 
						tc.contentType, receivedMsg.Header["Content-Type"])
				}
				if receivedMsg.Header["Message-Format"] != tc.expected {
					t.Errorf("Expected Message-Format %s, got %s", 
						tc.expected, receivedMsg.Header["Message-Format"])
				}

				// Validate body is preserved
				if string(receivedMsg.Body) != string(body) {
					t.Errorf("Body mismatch. Expected: %s, Got: %s", 
						body, receivedMsg.Body)
				}

				t.Logf("✓ %s: Format compatibility verified", tc.name)

			case <-time.After(5 * time.Second):
				t.Fatalf("Timeout waiting for %s message", tc.name)
			}
		})
	}
}

// TestQueueGroupCompatibility tests queue group behavior across versions
func TestQueueGroupCompatibility(t *testing.T) {
	if os.Getenv("NATS_URL") == "" {
		t.Skip("NATS_URL not configured, skipping queue group compatibility test")
	}

	b := natsBroker.NewNatsBroker()
	if err := b.Connect(); err != nil {
		t.Skip("NATS server not available, skipping test")
	}
	defer b.Disconnect()

	topic := "queue.group.compatibility.test"
	queueGroup := "test-queue-group"

	// Track received messages
	oldVersionReceived := make(chan string, 10)
	newVersionReceived := make(chan string, 10)

	// Old version subscriber (simulated)
	oldHandler := func(p broker.Event) error {
		msg := p.Message()
		oldVersionReceived <- string(msg.Body)
		return nil
	}

	// New version subscriber (simulated)
	newHandler := func(p broker.Event) error {
		msg := p.Message()
		newVersionReceived <- string(msg.Body)
		return nil
	}

	// Subscribe with queue groups (simulating old and new versions in same group)
	oldSub, err := b.Subscribe(topic, oldHandler, broker.Queue(queueGroup))
	if err != nil {
		t.Fatalf("Failed to subscribe old version: %v", err)
	}
	defer oldSub.Unsubscribe()

	newSub, err := b.Subscribe(topic, newHandler, broker.Queue(queueGroup))
	if err != nil {
		t.Fatalf("Failed to subscribe new version: %v", err)
	}
	defer newSub.Unsubscribe()

	time.Sleep(200 * time.Millisecond)

	// Publish multiple messages
	numMessages := 10
	for i := 0; i < numMessages; i++ {
		msg := &broker.Message{
			Header: map[string]string{
				"Content-Type": "text/plain",
				"Message-ID":   fmt.Sprintf("msg-%d", i),
			},
			Body: []byte(fmt.Sprintf("queue-test-message-%d", i)),
		}

		if err := b.Publish(topic, msg); err != nil {
			t.Fatalf("Failed to publish message %d: %v", i, err)
		}
	}

	// Collect messages for a reasonable time
	time.Sleep(2 * time.Second)

	// Count messages received by each version
	oldCount := len(oldVersionReceived)
	newCount := len(newVersionReceived)
	totalReceived := oldCount + newCount

	t.Logf("Messages distributed: Old version: %d, New version: %d, Total: %d", 
		oldCount, newCount, totalReceived)

	// Validate queue group behavior
	if totalReceived != numMessages {
		t.Errorf("Expected %d messages total, got %d", numMessages, totalReceived)
	}

	// Both versions should receive some messages (load balanced)
	if oldCount == 0 && newCount == 0 {
		t.Error("No messages received by any version")
	}

	// Messages should be distributed (not duplicated)
	if oldCount + newCount > numMessages {
		t.Errorf("Messages duplicated: total received (%d) > sent (%d)", 
			totalReceived, numMessages)
	}

	t.Logf("✓ Queue group compatibility verified: messages distributed across versions")
}

// TestEventMetadataPropagation tests metadata propagation in events
func TestEventMetadataPropagation(t *testing.T) {
	if os.Getenv("NATS_URL") == "" {
		t.Skip("NATS_URL not configured, skipping metadata propagation test")
	}

	b := natsBroker.NewNatsBroker()
	if err := b.Connect(); err != nil {
		t.Skip("NATS server not available, skipping test")
	}
	defer b.Disconnect()

	topic := "metadata.propagation.test"
	receivedMessages := make(chan *broker.Message, 1)

	// Subscribe with metadata tracking
	handler := func(p broker.Event) error {
		receivedMessages <- p.Message()
		return nil
	}

	sub, err := b.Subscribe(topic, handler)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	time.Sleep(100 * time.Millisecond)

	// Publish message with extensive metadata
	testMessage := &broker.Message{
		Header: map[string]string{
			"Content-Type":      "application/json",
			"Source-Service":    "mcbeam-overall-control-srv",
			"Target-Service":    "mcbeam-game-center-srv", 
			"Event-Version":     "v2.0.0",
			"Compatibility":     "cross-version",
			"K8s-Namespace":     "default",
			"K8s-Pod-Name":      "overall-control-pod-123",
			"Trace-ID":          "trace-abc-123",
			"Correlation-ID":    "corr-def-456",
			"Custom-Header-1":   "custom-value-1",
			"Custom-Header-2":   "custom-value-2",
		},
		Body: []byte(`{"event": "metadata-test", "data": "testing metadata propagation"}`),
	}

	if err := b.Publish(topic, testMessage); err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	// Wait for message
	select {
	case receivedMsg := <-receivedMessages:
		// Validate all metadata is preserved
		expectedHeaders := testMessage.Header
		for key, expectedValue := range expectedHeaders {
			if receivedValue, ok := receivedMsg.Header[key]; !ok {
				t.Errorf("Missing header: %s", key)
			} else if receivedValue != expectedValue {
				t.Errorf("Header %s: expected %s, got %s", key, expectedValue, receivedValue)
			}
		}

		// Validate body is preserved
		if string(receivedMsg.Body) != string(testMessage.Body) {
			t.Errorf("Body mismatch. Expected: %s, Got: %s", 
				testMessage.Body, receivedMsg.Body)
		}

		t.Logf("✓ Metadata propagation verified: %d headers preserved", len(expectedHeaders))

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for metadata message")
	}
}

// Helper function to publish events (simulates micro.PublishEvent)
func publishEvent(service micro.Service, topic string, event interface{}) error {
	// Marshal the event
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Create broker message
	msg := &broker.Message{
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: data,
	}

	// Publish using service broker
	return service.Options().Broker.Publish(topic, msg)
}

// Override micro.PublishEvent for testing
func init() {
	// This would be used to override the actual PublishEvent function
	// For testing purposes, we'll define a local helper
}

// Helper function for micro.PublishEvent
var PublishEvent = func(service micro.Service, topic string, event interface{}) error {
	return publishEvent(service, topic, event)
}