package tests

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"go-micro.dev/v5"
	"go-micro.dev/v5/broker"
	natsBroker "go-micro.dev/v5/broker/nats"
)

// TestEnvironment holds test environment configuration
type TestEnvironment struct {
	NATSUrl     string
	ETCDUrl     string
	K8sContext  string
	Namespace   string
	Registry    string
	Broker      string
}

// SetupTestEnvironment sets up the test environment with appropriate defaults
func SetupTestEnvironment() *TestEnvironment {
	env := &TestEnvironment{
		NATSUrl:    getEnvWithDefault("NATS_URL", "nats://localhost:4222"),
		ETCDUrl:    getEnvWithDefault("ETCD_ENDPOINTS", "http://localhost:2379"),
		K8sContext: getEnvWithDefault("K8S_CONTEXT", ""),
		Namespace:  getEnvWithDefault("K8S_NAMESPACE", "default"),
		Registry:   getEnvWithDefault("MICRO_REGISTRY", "memory"),
		Broker:     getEnvWithDefault("MICRO_BROKER", "memory"),
	}

	// Override with memory implementations for unit tests if external services not available
	if !isServiceAvailable(env.NATSUrl) {
		env.Broker = "memory"
		os.Setenv("MICRO_BROKER", "memory")
	}

	if !isServiceAvailable(env.ETCDUrl) {
		env.Registry = "memory"
		os.Setenv("MICRO_REGISTRY", "memory")
	}

	return env
}

// getEnvWithDefault gets environment variable with default fallback
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// isServiceAvailable checks if a service is available at the given URL
func isServiceAvailable(serviceUrl string) bool {
	// Simple connectivity check - in real implementation this would be more sophisticated
	if serviceUrl == "" {
		return false
	}
	
	// For now, return false to use memory implementations in tests
	return false
}

// MockServiceVersion represents a mock service with specific version characteristics
type MockServiceVersion struct {
	Name        string
	Version     string
	GoMicroVer  string
	Protocol    string
	Metadata    map[string]string
	Service     micro.Service
}

// CreateMockService creates a mock service with specified version characteristics
func CreateMockService(name, version, goMicroVer, protocol string, additionalMetadata map[string]string) *MockServiceVersion {
	metadata := map[string]string{
		"go-micro": goMicroVer,
		"protocol": protocol,
		"version":  version,
	}
	
	// Add additional metadata
	for k, v := range additionalMetadata {
		metadata[k] = v
	}

	service := micro.NewService(
		micro.Name(name),
		micro.Version(version),
		micro.Metadata(metadata),
		micro.Address(":0"), // Random port
	)

	return &MockServiceVersion{
		Name:       name,
		Version:    version,
		GoMicroVer: goMicroVer,
		Protocol:   protocol,
		Metadata:   metadata,
		Service:    service,
	}
}

// Start starts the mock service
func (m *MockServiceVersion) Start() error {
	m.Service.Init()
	go func() {
		m.Service.Run()
	}()
	return nil
}

// Stop stops the mock service
func (m *MockServiceVersion) Stop() error {
	return m.Service.Server().Stop()
}

// WaitForServiceRegistration waits for a service to be registered and discoverable
func WaitForServiceRegistration(service micro.Service, serviceName string, timeout time.Duration) error {
	reg := service.Options().Registry
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		services, err := reg.GetService(serviceName)
		if err == nil && len(services) > 0 {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	
	return fmt.Errorf("service %s not registered within timeout", serviceName)
}

// MessageCollector helps collect messages in tests
type MessageCollector struct {
	Messages chan *broker.Message
	Count    int
}

// NewMessageCollector creates a new message collector
func NewMessageCollector(bufferSize int) *MessageCollector {
	return &MessageCollector{
		Messages: make(chan *broker.Message, bufferSize),
		Count:    0,
	}
}

// Handler returns a broker event handler that collects messages
func (mc *MessageCollector) Handler() broker.Handler {
	return func(p broker.Event) error {
		mc.Count++
		mc.Messages <- p.Message()
		return nil
	}
}

// WaitForMessages waits for a specific number of messages
func (mc *MessageCollector) WaitForMessages(expected int, timeout time.Duration) ([]*broker.Message, error) {
	var messages []*broker.Message
	deadline := time.Now().Add(timeout)
	
	for len(messages) < expected && time.Now().Before(deadline) {
		select {
		case msg := <-mc.Messages:
			messages = append(messages, msg)
		case <-time.After(100 * time.Millisecond):
			// Continue waiting
		}
	}
	
	if len(messages) < expected {
		return messages, fmt.Errorf("expected %d messages, got %d", expected, len(messages))
	}
	
	return messages, nil
}

// TestMessage represents a structured test message
type TestMessage struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Version     string                 `json:"version"`
	Timestamp   int64                  `json:"timestamp"`
	Source      string                 `json:"source"`
	Target      string                 `json:"target"`
	Data        map[string]interface{} `json:"data"`
	Headers     map[string]string      `json:"headers"`
}

// NewTestMessage creates a new test message
func NewTestMessage(id, msgType, version, source, target string) *TestMessage {
	return &TestMessage{
		ID:        id,
		Type:      msgType,
		Version:   version,
		Timestamp: time.Now().Unix(),
		Source:    source,
		Target:    target,
		Data:      make(map[string]interface{}),
		Headers:   make(map[string]string),
	}
}

// WithData adds data to the test message
func (tm *TestMessage) WithData(key string, value interface{}) *TestMessage {
	tm.Data[key] = value
	return tm
}

// WithHeader adds a header to the test message
func (tm *TestMessage) WithHeader(key, value string) *TestMessage {
	tm.Headers[key] = value
	return tm
}

// ToBrokerMessage converts test message to broker message
func (tm *TestMessage) ToBrokerMessage() (*broker.Message, error) {
	body, err := json.Marshal(tm)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Content-Type": "application/json",
		"Message-ID":   tm.ID,
		"Message-Type": tm.Type,
		"Version":      tm.Version,
		"Source":       tm.Source,
		"Target":       tm.Target,
	}

	// Add custom headers
	for k, v := range tm.Headers {
		headers[k] = v
	}

	return &broker.Message{
		Header: headers,
		Body:   body,
	}, nil
}

// FromBrokerMessage creates test message from broker message
func FromBrokerMessage(msg *broker.Message) (*TestMessage, error) {
	var tm TestMessage
	if err := json.Unmarshal(msg.Body, &tm); err != nil {
		return nil, err
	}

	// Copy headers from message
	if tm.Headers == nil {
		tm.Headers = make(map[string]string)
	}
	for k, v := range msg.Header {
		tm.Headers[k] = v
	}

	return &tm, nil
}

// VersionCompatibilityTester helps test version compatibility
type VersionCompatibilityTester struct {
	OldService *MockServiceVersion
	NewService *MockServiceVersion
	Broker     broker.Broker
}

// NewVersionCompatibilityTester creates a new version compatibility tester
func NewVersionCompatibilityTester(oldName, newName string) *VersionCompatibilityTester {
	return &VersionCompatibilityTester{
		OldService: CreateMockService(
			oldName, "v1.0.0-beta", "v5.6.0-beta", "http",
			map[string]string{"generation": "old"},
		),
		NewService: CreateMockService(
			newName, "v2.0.0-local", "v5.6.0-local", "grpc",
			map[string]string{"generation": "new"},
		),
	}
}

// Setup initializes the compatibility tester
func (vct *VersionCompatibilityTester) Setup() error {
	// Initialize broker
	if os.Getenv("NATS_URL") != "" {
		vct.Broker = natsBroker.NewNatsBroker()
		if err := vct.Broker.Connect(); err != nil {
			// Fall back to memory broker
			vct.Broker = broker.NewMemoryBroker()
			if err := vct.Broker.Connect(); err != nil {
				return fmt.Errorf("failed to connect to broker: %v", err)
			}
		}
	} else {
		vct.Broker = broker.NewMemoryBroker()
		if err := vct.Broker.Connect(); err != nil {
			return fmt.Errorf("failed to connect to memory broker: %v", err)
		}
	}

	// Start services
	if err := vct.OldService.Start(); err != nil {
		return fmt.Errorf("failed to start old service: %v", err)
	}
	
	if err := vct.NewService.Start(); err != nil {
		return fmt.Errorf("failed to start new service: %v", err)
	}

	// Wait for services to register
	time.Sleep(1 * time.Second)

	return nil
}

// Cleanup cleans up the compatibility tester
func (vct *VersionCompatibilityTester) Cleanup() error {
	var errors []error
	
	if vct.OldService != nil {
		if err := vct.OldService.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop old service: %v", err))
		}
	}
	
	if vct.NewService != nil {
		if err := vct.NewService.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop new service: %v", err))
		}
	}
	
	if vct.Broker != nil {
		if err := vct.Broker.Disconnect(); err != nil {
			errors = append(errors, fmt.Errorf("failed to disconnect broker: %v", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}
	
	return nil
}

// TestCrossVersionMessage tests message passing between versions
func (vct *VersionCompatibilityTester) TestCrossVersionMessage(topic string, message *TestMessage) error {
	collector := NewMessageCollector(1)
	
	// New service subscribes
	sub, err := vct.Broker.Subscribe(topic, collector.Handler())
	if err != nil {
		return fmt.Errorf("failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()
	
	time.Sleep(100 * time.Millisecond) // Wait for subscription
	
	// Old service publishes
	brokerMsg, err := message.ToBrokerMessage()
	if err != nil {
		return fmt.Errorf("failed to convert message: %v", err)
	}
	
	if err := vct.Broker.Publish(topic, brokerMsg); err != nil {
		return fmt.Errorf("failed to publish: %v", err)
	}
	
	// Wait for message
	messages, err := collector.WaitForMessages(1, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to receive message: %v", err)
	}
	
	receivedMsg, err := FromBrokerMessage(messages[0])
	if err != nil {
		return fmt.Errorf("failed to parse received message: %v", err)
	}
	
	// Validate message
	if receivedMsg.ID != message.ID {
		return fmt.Errorf("message ID mismatch: expected %s, got %s", message.ID, receivedMsg.ID)
	}
	
	return nil
}

// K8sTestHelper helps with Kubernetes-specific testing
type K8sTestHelper struct {
	Namespace string
	PodName   string
	Hostname  string
}

// NewK8sTestHelper creates a new Kubernetes test helper
func NewK8sTestHelper() *K8sTestHelper {
	return &K8sTestHelper{
		Namespace: getEnvWithDefault("POD_NAMESPACE", "default"),
		PodName:   getEnvWithDefault("POD_NAME", "test-pod-"+strconv.FormatInt(time.Now().Unix(), 10)),
		Hostname:  getEnvWithDefault("HOSTNAME", "localhost"),
	}
}

// SetupK8sEnvironment sets up Kubernetes environment variables
func (k8s *K8sTestHelper) SetupK8sEnvironment() {
	os.Setenv("KUBERNETES_SERVICE_HOST", "10.96.0.1")
	os.Setenv("KUBERNETES_SERVICE_PORT", "443")
	os.Setenv("POD_NAMESPACE", k8s.Namespace)
	os.Setenv("POD_NAME", k8s.PodName)
	os.Setenv("HOSTNAME", k8s.Hostname)
}

// CleanupK8sEnvironment cleans up Kubernetes environment variables
func (k8s *K8sTestHelper) CleanupK8sEnvironment() {
	envVars := []string{
		"KUBERNETES_SERVICE_HOST",
		"KUBERNETES_SERVICE_PORT",
		"POD_NAMESPACE",
		"POD_NAME",
		"HOSTNAME",
	}
	
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}

// GetFreePort finds a free port for testing
func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}

// CreateServiceWithPort creates a service bound to a specific port
func CreateServiceWithPort(name, version string, port int, metadata map[string]string) micro.Service {
	address := fmt.Sprintf(":%d", port)
	
	service := micro.NewService(
		micro.Name(name),
		micro.Version(version),
		micro.Address(address),
		micro.Metadata(metadata),
	)
	
	return service
}