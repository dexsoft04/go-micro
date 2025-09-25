package fly

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

var (
	// initOnce ensures auto initialization only happens once
	initOnce sync.Once
)

// IsFlyEnvironment detects if running in Fly.io environment
func IsFlyEnvironment() bool {
	return os.Getenv("FLY_MACHINE_ID") != ""
}

// GetAdvertiseAddress gets the service registration address
func GetAdvertiseAddress(port string) string {
	// Priority: explicit environment variable configuration
	if addr := os.Getenv("MICRO_SERVER_ADVERTISE"); addr != "" {
		return addr
	}

	// Fly.io environment: use IPv6 address
	if IsFlyEnvironment() {
		if privateIP := os.Getenv("FLY_PRIVATE_IP"); privateIP != "" {
			// IPv6 addresses need to be wrapped in brackets
			if port == "" {
				port = "8080" // default port
			}
			return fmt.Sprintf("[%s]:%s", privateIP, port)
		}
	}

	return ""
}

// GetNodeID gets unique node identifier
func GetNodeID(serviceName, defaultID string) string {
	return fmt.Sprintf("%s-%s", serviceName, defaultID)
}

// GetNodeMetadata gets node metadata with Fly.io specific information
func GetNodeMetadata(existing map[string]string) map[string]string {
	if !IsFlyEnvironment() {
		return existing
	}

	// Add Fly.io specific metadata
	metadata := make(map[string]string)
	for k, v := range existing {
		metadata[k] = v
	}

	// Add Fly.io environment information
	metadata["platform"] = "fly.io"
	if machineID := os.Getenv("FLY_MACHINE_ID"); machineID != "" {
		metadata["machine_id"] = machineID
	}
	if region := os.Getenv("FLY_REGION"); region != "" {
		metadata["region"] = region
	}
	if appName := os.Getenv("FLY_APP_NAME"); appName != "" {
		metadata["app_name"] = appName
	}
	if privateIP := os.Getenv("FLY_PRIVATE_IP"); privateIP != "" {
		metadata["private_ip"] = privateIP
	}

	// Add DNS address as backup
	if machineID := os.Getenv("FLY_MACHINE_ID"); machineID != "" {
		if appName := os.Getenv("FLY_APP_NAME"); appName != "" {
			metadata["dns_address"] = fmt.Sprintf("%s.vm.%s.internal", machineID, appName)
		}
	}

	return metadata
}

// FormatIPv6Address ensures IPv6 address is properly formatted with brackets
func FormatIPv6Address(addr, port string) string {
	if addr == "" {
		return ""
	}

	// Check if it's an IPv6 address (contains colons)
	if strings.Contains(addr, ":") && !strings.HasPrefix(addr, "[") {
		if port != "" {
			return fmt.Sprintf("[%s]:%s", addr, port)
		}
		return fmt.Sprintf("[%s]", addr)
	}

	// IPv4 or already formatted
	if port != "" {
		_, existingPort, err := net.SplitHostPort(addr)
		if err != nil || existingPort == "" {
			return fmt.Sprintf("%s:%s", strings.Trim(addr, "[]"), port)
		}
	}

	return addr
}

// ExtractPortFromAddress extracts port from address string
func ExtractPortFromAddress(addr string) (host, port string, err error) {
	if addr == "" {
		return "", "", fmt.Errorf("empty address")
	}

	return net.SplitHostPort(addr)
}

// GetFlyEnvironmentInfo returns a map of all Fly.io environment variables
func GetFlyEnvironmentInfo() map[string]string {
	info := make(map[string]string)

	flyVars := []string{
		"FLY_MACHINE_ID",
		"FLY_PRIVATE_IP",
		"FLY_PUBLIC_IP",
		"FLY_REGION",
		"FLY_APP_NAME",
		"FLY_ALLOC_ID",
		"FLY_MACHINE_VERSION",
		"FLY_PROCESS_GROUP",
	}

	for _, varName := range flyVars {
		if value := os.Getenv(varName); value != "" {
			info[varName] = value
		}
	}

	return info
}

// AutoConfigureEnvironment automatically configures environment variables for Fly.io
// This function is called automatically when the service starts
func AutoConfigureEnvironment() {
	initOnce.Do(func() {
		if !IsFlyEnvironment() {
			return
		}

		// Determine the service port from existing configuration or use default
		port := "8080"
		if existingAddr := os.Getenv("MICRO_SERVER_ADDRESS"); existingAddr != "" {
			if _, p, err := net.SplitHostPort(existingAddr); err == nil && p != "" {
				port = p
			}
		}

		// Auto-configure server advertise address if not already set
		if os.Getenv("MICRO_SERVER_ADVERTISE") == "" {
			if privateIP := os.Getenv("FLY_PRIVATE_IP"); privateIP != "" {
				advertiseAddr := fmt.Sprintf("[%s]:%s", privateIP, port)
				os.Setenv("MICRO_SERVER_ADVERTISE", advertiseAddr)
			}
		}

		// Auto-configure server bind address if not already set
		if os.Getenv("MICRO_SERVER_ADDRESS") == "" {
			// Bind to all IPv6 addresses to accept connections
			os.Setenv("MICRO_SERVER_ADDRESS", fmt.Sprintf("[::]:%s", port))
		}

		// Set other useful environment variables for debugging/monitoring
		if machineID := os.Getenv("FLY_MACHINE_ID"); machineID != "" {
			if appName := os.Getenv("FLY_APP_NAME"); appName != "" {
				dnsAddr := fmt.Sprintf("%s.vm.%s.internal:%s", machineID, appName, port)
				os.Setenv("MICRO_DNS_ADDRESS", dnsAddr)
			}
		}
	})
}

// init automatically configures the Fly.io environment when the package is imported
func init() {
	AutoConfigureEnvironment()
}
