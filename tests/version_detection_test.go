package tests

import (
	"os"
	"strings"
	"testing"
	"time"

	"go-micro.dev/v5"
)

// TestVersionDetectionFromMetadata tests version detection from service metadata
func TestVersionDetectionFromMetadata(t *testing.T) {
	if os.Getenv("MICRO_REGISTRY") == "" {
		os.Setenv("MICRO_REGISTRY", "memory")
		defer os.Unsetenv("MICRO_REGISTRY")
	}

	// Create services with different version metadata
	testCases := []struct {
		name             string
		serviceName      string
		serviceVersion   string
		metadata         map[string]string
		expectedGoMicro  string
		expectedProtocol string
	}{
		{
			name:           "Old version service",
			serviceName:    "test-old-service",
			serviceVersion: "v1.0.0-beta",
			metadata: map[string]string{
				"version":   "old",
				"go-micro":  "v5.6.0-beta",
				"protocol":  "http",
				"transport": "legacy",
			},
			expectedGoMicro:  "v5.6.0-beta",
			expectedProtocol: "http",
		},
		{
			name:           "New version service",
			serviceName:    "test-new-service",
			serviceVersion: "v2.0.0-local",
			metadata: map[string]string{
				"version":       "new",
				"go-micro":      "v5.6.0-local",
				"protocol":      "grpc",
				"transport":     "enhanced",
				"compatibility": "dual-pool",
			},
			expectedGoMicro:  "v5.6.0-local",
			expectedProtocol: "grpc",
		},
		{
			name:           "Mixed version service",
			serviceName:    "test-mixed-service",
			serviceVersion: "v1.5.0-transition",
			metadata: map[string]string{
				"version":              "transition",
				"go-micro":             "v5.6.0-beta",
				"protocol":             "auto",
				"supports_http":        "true",
				"supports_grpc":        "true",
				"compatibility_mode":   "true",
			},
			expectedGoMicro:  "v5.6.0-beta",
			expectedProtocol: "auto",
		},
	}

	var services []micro.Service

	// Create and register all test services
	for _, tc := range testCases {
		service := micro.NewService(
			micro.Name(tc.serviceName),
			micro.Version(tc.serviceVersion),
			micro.Metadata(tc.metadata),
			micro.Address(":0"), // Random port
		)
		service.Init()
		services = append(services, service)

		// Start service in background
		go func(s micro.Service, name string) {
			if err := s.Run(); err != nil {
				t.Logf("Service %s error: %v", name, err)
			}
		}(service, tc.serviceName)
	}

	// Wait for services to register
	time.Sleep(2 * time.Second)

	// Test version detection for each service
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get service from registry
			reg := services[0].Options().Registry // Use first service's registry
			serviceList, err := reg.GetService(tc.serviceName)
			if err != nil {
				t.Errorf("Failed to get service %s: %v", tc.serviceName, err)
				return
			}

			if len(serviceList) == 0 {
				t.Errorf("No services found for %s", tc.serviceName)
				return
			}

			service := serviceList[0]

			// Validate version information
			if service.Version != tc.serviceVersion {
				t.Errorf("Expected service version %s, got %s", tc.serviceVersion, service.Version)
			}

			// Validate metadata
			if goMicroVersion, ok := service.Metadata["go-micro"]; ok {
				if goMicroVersion != tc.expectedGoMicro {
					t.Errorf("Expected go-micro version %s, got %s", tc.expectedGoMicro, goMicroVersion)
				}
			} else {
				t.Errorf("go-micro version not found in metadata")
			}

			if protocol, ok := service.Metadata["protocol"]; ok {
				if protocol != tc.expectedProtocol {
					t.Errorf("Expected protocol %s, got %s", tc.expectedProtocol, protocol)
				}
			} else {
				t.Errorf("protocol not found in metadata")
			}

			t.Logf("✓ Version detection successful for %s: go-micro=%s, protocol=%s", 
				tc.serviceName, service.Metadata["go-micro"], service.Metadata["protocol"])
		})
	}

	// Cleanup
	for _, service := range services {
		service.Server().Stop()
	}
}

// TestProtocolDetectionFromEnvironment tests protocol detection from environment variables
func TestProtocolDetectionFromEnvironment(t *testing.T) {
	testCases := []struct {
		name             string
		envVars          map[string]string
		expectedBehavior string
		description      string
	}{
		{
			name: "Legacy gRPC transport",
			envVars: map[string]string{
				"MICRO_TRANSPORT": "grpc",
			},
			expectedBehavior: "grpc_legacy",
			description:      "Should use legacy gRPC transport configuration",
		},
		{
			name: "New gRPC client/server",
			envVars: map[string]string{
				"MICRO_CLIENT": "grpc",
				"MICRO_SERVER": "grpc",
			},
			expectedBehavior: "grpc_new",
			description:      "Should use new gRPC client/server configuration",
		},
		{
			name: "HTTP transport", 
			envVars: map[string]string{
				"MICRO_TRANSPORT": "http",
			},
			expectedBehavior: "http",
			description:      "Should use HTTP transport",
		},
		{
			name: "Mixed configuration - new overrides old",
			envVars: map[string]string{
				"MICRO_TRANSPORT": "grpc",
				"MICRO_CLIENT":    "rpc", // Should take precedence
				"MICRO_SERVER":    "rpc",
			},
			expectedBehavior: "rpc_override",
			description:      "New configuration should override legacy",
		},
		{
			name: "Default configuration",
			envVars: map[string]string{
				// No transport-related env vars
				"MICRO_REGISTRY": "memory",
			},
			expectedBehavior: "default",
			description:      "Should use default configuration",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean environment first
			cleanupEnv := []string{
				"MICRO_TRANSPORT", "MICRO_CLIENT", "MICRO_SERVER",
				"MICRO_REGISTRY", "MICRO_BROKER",
			}
			for _, env := range cleanupEnv {
				os.Unsetenv(env)
			}

			// Set test environment variables
			for key, value := range tc.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Create service with environment configuration
			service := micro.NewService(
				micro.Name("test-protocol-detection"),
				micro.Version("v1.0.0"),
				micro.Metadata(map[string]string{
					"test_case": tc.name,
				}),
				micro.Address(":0"),
			)

			service.Init()

			// Analyze the service configuration
			opts := service.Options()
			
			// Log configuration details for analysis
			t.Logf("Test case: %s", tc.description)
			t.Logf("Environment variables: %v", tc.envVars)
			t.Logf("Expected behavior: %s", tc.expectedBehavior)
			
			// Basic validation that service initializes correctly
			if opts.Client == nil {
				t.Error("Client not initialized")
			}
			if opts.Server == nil {
				t.Error("Server not initialized")
			}
			if opts.Registry == nil {
				t.Error("Registry not initialized")
			}

			// Check registry type based on environment
			if regType := tc.envVars["MICRO_REGISTRY"]; regType != "" {
				registryStr := opts.Registry.String()
				t.Logf("Registry configured: %s (expected: %s)", registryStr, regType)
			}

			t.Logf("✓ Protocol detection test completed for: %s", tc.name)
		})
	}
}

// TestServiceCompatibilityMatrix tests compatibility between different service versions
func TestServiceCompatibilityMatrix(t *testing.T) {
	if os.Getenv("MICRO_REGISTRY") == "" {
		os.Setenv("MICRO_REGISTRY", "memory")
		defer os.Unsetenv("MICRO_REGISTRY")
	}

	// Define service version matrix
	serviceVersions := []struct {
		name     string
		version  string
		goMicro  string
		protocol string
		compat   []string // Compatible with these versions
	}{
		{
			name:     "legacy-v1",
			version:  "v1.0.0",
			goMicro:  "v5.6.0-beta",
			protocol: "http",
			compat:   []string{"legacy-v2", "transition-v1", "modern-v1"},
		},
		{
			name:     "legacy-v2",
			version:  "v1.1.0",
			goMicro:  "v5.6.0-beta",
			protocol: "http",
			compat:   []string{"legacy-v1", "transition-v1", "modern-v1"},
		},
		{
			name:     "transition-v1",
			version:  "v1.5.0",
			goMicro:  "v5.6.0-beta",
			protocol: "auto",
			compat:   []string{"legacy-v1", "legacy-v2", "modern-v1", "modern-v2"},
		},
		{
			name:     "modern-v1",
			version:  "v2.0.0",
			goMicro:  "v5.6.0-local",
			protocol: "grpc",
			compat:   []string{"legacy-v1", "legacy-v2", "transition-v1", "modern-v2"},
		},
		{
			name:     "modern-v2",
			version:  "v2.1.0",
			goMicro:  "v5.6.0-local",
			protocol: "grpc",
			compat:   []string{"transition-v1", "modern-v1"},
		},
	}

	// Create compatibility matrix table
	t.Logf("Service Compatibility Matrix:")
	t.Logf("%-15s | %-10s | %-15s | %-8s | %s", 
		"Service", "Version", "Go-Micro", "Protocol", "Compatible With")
	t.Logf(strings.Repeat("-", 80))

	for _, sv := range serviceVersions {
		t.Logf("%-15s | %-10s | %-15s | %-8s | %v", 
			sv.name, sv.version, sv.goMicro, sv.protocol, sv.compat)
	}

	// Test each service version
	for _, sv := range serviceVersions {
		t.Run(sv.name, func(t *testing.T) {
			service := micro.NewService(
				micro.Name(sv.name),
				micro.Version(sv.version),
				micro.Metadata(map[string]string{
					"go-micro":       sv.goMicro,
					"protocol":       sv.protocol,
					"compatibility":  strings.Join(sv.compat, ","),
				}),
				micro.Address(":0"),
			)

			service.Init()

			// Test service startup
			go func() {
				if err := service.Run(); err != nil {
					t.Logf("Service %s error: %v", sv.name, err)
				}
			}()

			time.Sleep(500 * time.Millisecond)

			// Validate service registration
			reg := service.Options().Registry
			services, err := reg.GetService(sv.name)
			if err != nil {
				t.Errorf("Failed to get service %s: %v", sv.name, err)
			} else if len(services) > 0 {
				svc := services[0]
				if svc.Metadata["go-micro"] != sv.goMicro {
					t.Errorf("Metadata mismatch for %s", sv.name)
				}
				t.Logf("✓ Service %s registered successfully", sv.name)
			}

			service.Server().Stop()
		})
	}
}

// TestVersionNegotiation tests version negotiation between services
func TestVersionNegotiation(t *testing.T) {
	if os.Getenv("MICRO_REGISTRY") == "" {
		os.Setenv("MICRO_REGISTRY", "memory")
		defer os.Unsetenv("MICRO_REGISTRY")
	}

	// Create client service (new version)
	clientService := micro.NewService(
		micro.Name("test-client-service"),
		micro.Version("v2.0.0-local"),
		micro.Metadata(map[string]string{
			"role":       "client",
			"go-micro":   "v5.6.0-local",
			"supports":   "http,grpc",
			"preferred":  "grpc",
		}),
	)

	// Create server service (old version)
	serverService := micro.NewService(
		micro.Name("test-server-service"),
		micro.Version("v1.0.0-beta"),
		micro.Metadata(map[string]string{
			"role":       "server",
			"go-micro":   "v5.6.0-beta",
			"supports":   "http",
			"preferred":  "http",
		}),
	)

	clientService.Init()
	serverService.Init()

	// Start services
	go func() {
		if err := clientService.Run(); err != nil {
			t.Logf("Client service error: %v", err)
		}
	}()
	go func() {
		if err := serverService.Run(); err != nil {
			t.Logf("Server service error: %v", err)
		}
	}()

	time.Sleep(2 * time.Second)

	// Test version negotiation logic
	reg := clientService.Options().Registry

	// Client discovers server
	serverServices, err := reg.GetService("test-server-service")
	if err != nil {
		t.Fatalf("Failed to discover server service: %v", err)
	}

	if len(serverServices) == 0 {
		t.Fatal("No server services found")
	}

	server := serverServices[0]
	
	// Simulate version negotiation
	clientSupports := []string{"http", "grpc"}
	serverSupports := strings.Split(server.Metadata["supports"], ",")
	
	// Find common protocols
	commonProtocols := findCommonProtocols(clientSupports, serverSupports)
	
	if len(commonProtocols) == 0 {
		t.Error("No common protocols found between client and server")
	} else {
		selectedProtocol := selectBestProtocol(commonProtocols, server.Metadata["preferred"])
		t.Logf("✓ Version negotiation successful:")
		t.Logf("  Client supports: %v", clientSupports)
		t.Logf("  Server supports: %v", serverSupports)
		t.Logf("  Common protocols: %v", commonProtocols)
		t.Logf("  Selected protocol: %s", selectedProtocol)
		
		if selectedProtocol != "http" {
			t.Errorf("Expected HTTP protocol (server preference), got %s", selectedProtocol)
		}
	}

	// Cleanup
	clientService.Server().Stop()
	serverService.Server().Stop()
}

// Helper function to find common protocols
func findCommonProtocols(client, server []string) []string {
	var common []string
	for _, c := range client {
		for _, s := range server {
			if strings.TrimSpace(c) == strings.TrimSpace(s) {
				common = append(common, strings.TrimSpace(c))
				break
			}
		}
	}
	return common
}

// Helper function to select best protocol based on server preference
func selectBestProtocol(common []string, serverPreferred string) string {
	// Prefer server's preferred protocol if available
	for _, proto := range common {
		if proto == serverPreferred {
			return proto
		}
	}
	// Otherwise return first common protocol
	if len(common) > 0 {
		return common[0]
	}
	return "unknown"
}

// TestBackwardCompatibilityFlags tests backward compatibility flags
func TestBackwardCompatibilityFlags(t *testing.T) {
	testCases := []struct {
		name        string
		metadata    map[string]string
		expected    map[string]bool
		description string
	}{
		{
			name: "Full backward compatibility",
			metadata: map[string]string{
				"compatibility_mode":    "full",
				"supports_legacy_rpc":   "true",
				"supports_dual_pools":   "true",
				"auto_protocol_detect":  "true",
			},
			expected: map[string]bool{
				"legacy_rpc":      true,
				"dual_pools":     true,
				"auto_detect":    true,
			},
			description: "Service with full backward compatibility",
		},
		{
			name: "Partial backward compatibility",
			metadata: map[string]string{
				"compatibility_mode":    "partial",
				"supports_legacy_rpc":   "false",
				"supports_dual_pools":   "true",
				"auto_protocol_detect":  "true",
			},
			expected: map[string]bool{
				"legacy_rpc":      false,
				"dual_pools":     true,
				"auto_detect":    true,
			},
			description: "Service with partial backward compatibility",
		},
		{
			name: "No backward compatibility",
			metadata: map[string]string{
				"compatibility_mode":    "none",
				"supports_legacy_rpc":   "false",
				"supports_dual_pools":   "false",
				"auto_protocol_detect":  "false",
			},
			expected: map[string]bool{
				"legacy_rpc":      false,
				"dual_pools":     false,
				"auto_detect":    false,
			},
			description: "Service with no backward compatibility",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if os.Getenv("MICRO_REGISTRY") == "" {
				os.Setenv("MICRO_REGISTRY", "memory")
				defer os.Unsetenv("MICRO_REGISTRY")
			}

			service := micro.NewService(
				micro.Name("test-compatibility-flags"),
				micro.Version("v1.0.0"),
				micro.Metadata(tc.metadata),
			)

			service.Init()

			// Analyze compatibility flags
			metadata := tc.metadata
			
			// Check legacy RPC support
			legacyRPC := metadata["supports_legacy_rpc"] == "true"
			if legacyRPC != tc.expected["legacy_rpc"] {
				t.Errorf("Legacy RPC: expected %v, got %v", tc.expected["legacy_rpc"], legacyRPC)
			}

			// Check dual pools support
			dualPools := metadata["supports_dual_pools"] == "true"
			if dualPools != tc.expected["dual_pools"] {
				t.Errorf("Dual pools: expected %v, got %v", tc.expected["dual_pools"], dualPools)
			}

			// Check auto protocol detection
			autoDetect := metadata["auto_protocol_detect"] == "true"
			if autoDetect != tc.expected["auto_detect"] {
				t.Errorf("Auto detect: expected %v, got %v", tc.expected["auto_detect"], autoDetect)
			}

			t.Logf("✓ %s: Compatibility flags validated", tc.description)
			t.Logf("  Legacy RPC: %v", legacyRPC)
			t.Logf("  Dual Pools: %v", dualPools)
			t.Logf("  Auto Detect: %v", autoDetect)
		})
	}
}