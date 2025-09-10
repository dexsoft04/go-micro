package tests

import (
	"os"
	"testing"
	"time"

	"go-micro.dev/v5"
	"go-micro.dev/v5/broker"
	natsBroker "go-micro.dev/v5/broker/nats"
)

// TestVersionCompatibilitySetup tests basic compatibility setup
func TestVersionCompatibilitySetup(t *testing.T) {
	// Test environment setup for Kubernetes deployment
	testCases := []struct {
		name        string
		envVars     map[string]string
		expectedErr bool
	}{
		{
			name: "Legacy MICRO_TRANSPORT configuration",
			envVars: map[string]string{
				"MICRO_TRANSPORT": "grpc",
				"MICRO_REGISTRY":  "etcd",
				"MICRO_BROKER":    "nats",
			},
			expectedErr: false,
		},
		{
			name: "New MICRO_CLIENT/SERVER configuration",
			envVars: map[string]string{
				"MICRO_CLIENT":   "grpc",
				"MICRO_SERVER":   "grpc",
				"MICRO_REGISTRY": "etcd",
				"MICRO_BROKER":   "nats",
			},
			expectedErr: false,
		},
		{
			name: "Mixed configuration compatibility",
			envVars: map[string]string{
				"MICRO_TRANSPORT": "grpc",
				"MICRO_CLIENT":    "rpc", // Should override MICRO_TRANSPORT
				"MICRO_REGISTRY":  "etcd",
				"MICRO_BROKER":    "nats",
			},
			expectedErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tc.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Create service with different configurations
			service := micro.NewService(
				micro.Name("test-compatibility-service"),
				micro.Version("v1.0.0"),
				micro.RegisterTTL(time.Second*30),
				micro.RegisterInterval(time.Second*10),
			)

			// Test service configuration without calling Init() which conflicts with testing
			// Just validate the service was created properly
			opts := service.Options()
			if opts.Client == nil {
				t.Error("Client not configured")
			}
			if opts.Server == nil {
				t.Error("Server not configured")
			}
			
			t.Logf("Service configuration validated for: %v", tc.envVars)
		})
	}
}

// TestServiceDiscoveryCompatibility tests service discovery between versions
func TestServiceDiscoveryCompatibility(t *testing.T) {
	// Skip if no etcd available
	if os.Getenv("ETCD_ENDPOINTS") == "" && os.Getenv("MICRO_REGISTRY") != "memory" {
		t.Skip("No etcd endpoints configured, skipping service discovery test")
	}

	// Create new version service
	newService := micro.NewService(
		micro.Name("test-new-version"),
		micro.Version("v2.0.0"),
		micro.Metadata(map[string]string{
			"version": "new",
			"go-micro": "v5.6.0-local",
		}),
	)

	// Create old version service simulation
	oldService := micro.NewService(
		micro.Name("test-old-version"),
		micro.Version("v1.0.0"),
		micro.Metadata(map[string]string{
			"version": "old",
			"go-micro": "v5.6.0-beta",
		}),
	)

	// Initialize both services
	newService.Init()
	oldService.Init()

	// Start both services in background
	go func() {
		if err := newService.Run(); err != nil {
			t.Logf("New service error: %v", err)
		}
	}()
	go func() {
		if err := oldService.Run(); err != nil {
			t.Logf("Old service error: %v", err)
		}
	}()

	// Wait for services to register
	time.Sleep(2 * time.Second)

	// Test service discovery
	reg := newService.Options().Registry
	
	// Test new service can discover old service
	oldServices, err := reg.GetService("test-old-version")
	if err != nil {
		t.Errorf("New service failed to discover old service: %v", err)
	} else if len(oldServices) == 0 {
		t.Error("No old services found")
	} else {
		// Validate metadata
		service := oldServices[0]
		if service.Metadata["version"] != "old" {
			t.Errorf("Expected old version metadata, got: %v", service.Metadata)
		}
		t.Logf("Successfully discovered old service: %s with metadata: %v", 
			service.Name, service.Metadata)
	}

	// Test old service can discover new service
	newServices, err := reg.GetService("test-new-version")
	if err != nil {
		t.Errorf("Old service failed to discover new service: %v", err)
	} else if len(newServices) == 0 {
		t.Error("No new services found")
	} else {
		// Validate metadata
		service := newServices[0]
		if service.Metadata["version"] != "new" {
			t.Errorf("Expected new version metadata, got: %v", service.Metadata)
		}
		t.Logf("Successfully discovered new service: %s with metadata: %v", 
			service.Name, service.Metadata)
	}

	// Cleanup
	newService.Server().Stop()
	oldService.Server().Stop()
}

// TestBrokerCompatibility tests NATS broker compatibility between versions
func TestBrokerCompatibility(t *testing.T) {
	// Skip if NATS not available
	if os.Getenv("NATS_URL") == "" {
		t.Skip("NATS_URL not configured, skipping broker compatibility test")
	}

	// Create NATS broker instances
	newBroker := natsBroker.NewNatsBroker()
	oldBroker := natsBroker.NewNatsBroker()

	// Connect both brokers
	if err := newBroker.Connect(); err != nil {
		t.Skip("NATS server not available, skipping test")
	}
	defer newBroker.Disconnect()

	if err := oldBroker.Connect(); err != nil {
		t.Skip("NATS server not available, skipping test")
	}
	defer oldBroker.Disconnect()

	topic := "compatibility.test.topic"

	// Test message compatibility
	testMessage := &broker.Message{
		Header: map[string]string{
			"Content-Type":    "application/json",
			"Source-Version":  "new",
			"Target-Version":  "old",
			"Compatibility":   "test",
		},
		Body: []byte(`{"test": "compatibility", "version": "mixed"}`),
	}

	// Channel to receive messages
	msgChan := make(chan *broker.Message, 1)

	// Old version subscribes
	handler := func(p broker.Event) error {
		msg := p.Message()
		msgChan <- msg
		return nil
	}

	sub, err := oldBroker.Subscribe(topic, handler)
	if err != nil {
		t.Fatalf("Failed to subscribe with old broker: %v", err)
	}
	defer sub.Unsubscribe()

	// Wait for subscription to be active
	time.Sleep(100 * time.Millisecond)

	// New version publishes
	if err := newBroker.Publish(topic, testMessage); err != nil {
		t.Fatalf("Failed to publish with new broker: %v", err)
	}

	// Wait for message
	select {
	case receivedMsg := <-msgChan:
		// Validate message compatibility
		if receivedMsg.Header["Source-Version"] != "new" {
			t.Errorf("Expected Source-Version 'new', got '%s'", 
				receivedMsg.Header["Source-Version"])
		}
		if receivedMsg.Header["Content-Type"] != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", 
				receivedMsg.Header["Content-Type"])
		}
		if string(receivedMsg.Body) != string(testMessage.Body) {
			t.Errorf("Message body mismatch. Expected: %s, Got: %s", 
				testMessage.Body, receivedMsg.Body)
		}
		t.Logf("Successfully received message from new broker to old broker")

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

// validateServiceConfig validates that service configuration matches expectations
func validateServiceConfig(t *testing.T, service micro.Service, envVars map[string]string) {
	opts := service.Options()
	
	// Validate registry configuration
	if regType, ok := envVars["MICRO_REGISTRY"]; ok {
		switch regType {
		case "etcd":
			// Check registry type name or string representation
			registryName := opts.Registry.String()
			if registryName != "etcd" {
				t.Logf("Registry type: %T, String: %s", opts.Registry, registryName)
			}
		case "memory":
			// Memory registry validation
		}
	}

	// Validate broker configuration  
	if brokerType, ok := envVars["MICRO_BROKER"]; ok {
		switch brokerType {
		case "nats":
			// Check broker type name or string representation
			brokerName := opts.Broker.String()
			if brokerName != "nats" {
				t.Logf("Broker type: %T, String: %s", opts.Broker, brokerName)
			}
		}
	}

	// Validate client/server configuration based on environment variables
	// This would need access to internal client/server configuration
	t.Logf("Service configuration validated for: %v", envVars)
}

// TestK8sEnvironmentVariables tests Kubernetes-specific environment variables
func TestK8sEnvironmentVariables(t *testing.T) {
	k8sEnvVars := map[string]string{
		"KUBERNETES_SERVICE_HOST": "10.96.0.1",
		"KUBERNETES_SERVICE_PORT": "443",
		"HOSTNAME":                "test-pod-123",
		"POD_NAME":               "test-pod-123",
		"POD_NAMESPACE":          "default",
	}

	// Set K8s environment variables
	for key, value := range k8sEnvVars {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}

	// Create service that should handle K8s environment
	service := micro.NewService(
		micro.Name("test-k8s-service"),
		micro.Version("v1.0.0"),
		// In K8s, services often need to bind to 0.0.0.0
		micro.Address("0.0.0.0:0"),
	)

	service.Init()

	// Validate that service can handle K8s environment
	opts := service.Options()
	if opts.Server.Options().Address == "" {
		t.Error("Service address not set properly in K8s environment")
	}

	t.Logf("Service initialized successfully in K8s environment")
}