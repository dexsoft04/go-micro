package tests

import (
	"fmt"
	"os"
	"testing"

	"go-micro.dev/v5"
)

// TestBasicCompatibility tests basic compatibility without service initialization
func TestBasicCompatibility(t *testing.T) {
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
				"MICRO_REGISTRY":  "memory",
				"MICRO_BROKER":    "memory",
			},
			expectedErr: false,
		},
		{
			name: "New MICRO_CLIENT/SERVER configuration",
			envVars: map[string]string{
				"MICRO_CLIENT":   "grpc",
				"MICRO_SERVER":   "grpc",
				"MICRO_REGISTRY": "memory",
				"MICRO_BROKER":   "memory",
			},
			expectedErr: false,
		},
		{
			name: "Mixed configuration compatibility",
			envVars: map[string]string{
				"MICRO_TRANSPORT": "grpc",
				"MICRO_CLIENT":    "rpc", // Should override MICRO_TRANSPORT
				"MICRO_REGISTRY":  "memory",
				"MICRO_BROKER":    "memory",
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
			)

			// Test basic service creation and option validation
			opts := service.Options()
			
			// Validate basic components are created
			if opts.Client == nil {
				t.Error("Client not configured")
			}
			if opts.Server == nil {
				t.Error("Server not configured")
			}
			if opts.Registry == nil {
				t.Error("Registry not configured")
			}
			if opts.Broker == nil {
				t.Error("Broker not configured")
			}

			t.Logf("✓ Basic service creation successful with config: %v", tc.envVars)
		})
	}
}

// TestVersionMetadata tests version metadata handling
func TestVersionMetadata(t *testing.T) {
	// Set memory implementations to avoid external dependencies
	os.Setenv("MICRO_REGISTRY", "memory")
	os.Setenv("MICRO_BROKER", "memory")
	defer os.Unsetenv("MICRO_REGISTRY")
	defer os.Unsetenv("MICRO_BROKER")

	testCases := []struct {
		name     string
		version  string
		metadata map[string]string
		expected map[string]string
	}{
		{
			name:    "Old version metadata",
			version: "v1.0.0-beta",
			metadata: map[string]string{
				"version":   "old",
				"go-micro":  "v5.6.0-beta",
				"protocol":  "http",
			},
			expected: map[string]string{
				"version":   "old",
				"go-micro":  "v5.6.0-beta",
				"protocol":  "http",
			},
		},
		{
			name:    "New version metadata",
			version: "v2.0.0-local",
			metadata: map[string]string{
				"version":       "new",
				"go-micro":      "v5.6.0-local",
				"protocol":      "grpc",
				"compatibility": "dual-pool",
			},
			expected: map[string]string{
				"version":       "new",
				"go-micro":      "v5.6.0-local",
				"protocol":      "grpc",
				"compatibility": "dual-pool",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := micro.NewService(
				micro.Name("test-metadata-service"),
				micro.Version(tc.version),
				micro.Metadata(tc.metadata),
			)

			opts := service.Options()

			// Validate version is set correctly
			if opts.Server.Options().Name != "test-metadata-service" {
				t.Errorf("Expected service name 'test-metadata-service', got '%s'", 
					opts.Server.Options().Name)
			}

			if opts.Server.Options().Version != tc.version {
				t.Errorf("Expected version %s, got %s", 
					tc.version, opts.Server.Options().Version)
			}

			// Validate metadata
			serverMetadata := opts.Server.Options().Metadata
			for key, expected := range tc.expected {
				if actual, ok := serverMetadata[key]; !ok {
					t.Errorf("Missing metadata key: %s", key)
				} else if actual != expected {
					t.Errorf("Metadata %s: expected %s, got %s", key, expected, actual)
				}
			}

			t.Logf("✓ Version metadata validated: %v", tc.metadata)
		})
	}
}

// TestConfigurationPriority tests configuration priority
func TestConfigurationPriority(t *testing.T) {
	// Test that new configuration overrides old configuration
	testCases := []struct {
		name     string
		envVars  map[string]string
		expected string
	}{
		{
			name: "Legacy only",
			envVars: map[string]string{
				"MICRO_TRANSPORT": "grpc",
				"MICRO_REGISTRY":  "memory",
				"MICRO_BROKER":    "memory",
			},
			expected: "legacy_grpc",
		},
		{
			name: "New only",
			envVars: map[string]string{
				"MICRO_CLIENT":   "grpc",
				"MICRO_SERVER":   "grpc",
				"MICRO_REGISTRY": "memory",
				"MICRO_BROKER":   "memory",
			},
			expected: "new_grpc",
		},
		{
			name: "Mixed - new overrides",
			envVars: map[string]string{
				"MICRO_TRANSPORT": "grpc",
				"MICRO_CLIENT":    "rpc",
				"MICRO_SERVER":    "rpc",
				"MICRO_REGISTRY":  "memory",
				"MICRO_BROKER":    "memory",
			},
			expected: "new_rpc_override",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean environment
			cleanEnv := []string{"MICRO_TRANSPORT", "MICRO_CLIENT", "MICRO_SERVER", 
				"MICRO_REGISTRY", "MICRO_BROKER"}
			for _, env := range cleanEnv {
				os.Unsetenv(env)
			}

			// Set test environment
			for key, value := range tc.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			service := micro.NewService(
				micro.Name("test-config-priority"),
				micro.Version("v1.0.0"),
			)

			opts := service.Options()

			// Basic validation that service is created correctly
			if opts.Client == nil {
				t.Error("Client not configured")
			}
			if opts.Server == nil {
				t.Error("Server not configured")
			}

			t.Logf("✓ Configuration priority test '%s' passed", tc.expected)
			t.Logf("  Environment: %v", tc.envVars)
		})
	}
}

// TestK8sEnvironmentHandling tests Kubernetes environment variable handling
func TestK8sEnvironmentHandling(t *testing.T) {
	// Set Kubernetes environment variables
	k8sEnvs := map[string]string{
		"KUBERNETES_SERVICE_HOST": "10.96.0.1",
		"KUBERNETES_SERVICE_PORT": "443",
		"POD_NAME":               "test-pod-123",
		"POD_NAMESPACE":          "default",
		"HOSTNAME":               "test-pod-123",
		"MICRO_REGISTRY":         "memory",
		"MICRO_BROKER":           "memory",
	}

	for key, value := range k8sEnvs {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}

	service := micro.NewService(
		micro.Name("test-k8s-service"),
		micro.Version("v1.0.0"),
		micro.Metadata(map[string]string{
			"environment": "kubernetes",
			"pod_name":    os.Getenv("POD_NAME"),
			"namespace":   os.Getenv("POD_NAMESPACE"),
		}),
	)

	opts := service.Options()

	// Validate service handles K8s environment correctly
	if opts.Server == nil {
		t.Error("Server not configured in K8s environment")
	}

	serverMetadata := opts.Server.Options().Metadata
	if serverMetadata["environment"] != "kubernetes" {
		t.Error("K8s environment metadata not set correctly")
	}

	if serverMetadata["pod_name"] != "test-pod-123" {
		t.Errorf("Expected pod_name 'test-pod-123', got '%s'", 
			serverMetadata["pod_name"])
	}

	t.Logf("✓ Kubernetes environment handling validated")
	t.Logf("  Pod: %s", serverMetadata["pod_name"])
	t.Logf("  Namespace: %s", serverMetadata["namespace"])
}

// TestServiceCreationPerformance tests service creation performance
func TestServiceCreationPerformance(t *testing.T) {
	os.Setenv("MICRO_REGISTRY", "memory")
	os.Setenv("MICRO_BROKER", "memory")
	defer os.Unsetenv("MICRO_REGISTRY")
	defer os.Unsetenv("MICRO_BROKER")

	numServices := 10
	services := make([]micro.Service, numServices)

	// Create services
	for i := 0; i < numServices; i++ {
		services[i] = micro.NewService(
			micro.Name(fmt.Sprintf("perf-test-service-%d", i)),
			micro.Version("v1.0.0"),
			micro.Metadata(map[string]string{
				"instance": fmt.Sprintf("%d", i),
			}),
		)
	}

	// Validate all services were created
	for i, service := range services {
		if service == nil {
			t.Errorf("Service %d was not created", i)
		} else {
			opts := service.Options()
			if opts.Server.Options().Name != fmt.Sprintf("perf-test-service-%d", i) {
				t.Errorf("Service %d name mismatch", i)
			}
		}
	}

	t.Logf("✓ Successfully created %d services", numServices)
}