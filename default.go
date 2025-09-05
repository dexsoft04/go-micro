package micro

import (
	"github.com/philchia/agollo/v4"
	"github.com/zigo2048/mcbeam-common-lib/common/config"
	"github.com/zigo2048/mcbeam-common-lib/plugins/config/apollo/v3"
	_ "go-micro.dev/v5/broker/nats"
	_ "go-micro.dev/v5/registry/etcd"
	_ "go-micro.dev/v5/transport/grpc"
	"os"
	"path/filepath"
	"sync"
)

var (
	configOnce sync.Once
	configInitialized bool
)

// initDefaultConfig initializes Apollo configuration lazily
// Similar to NATS broker pattern - only initialize when actually needed
func initDefaultConfig() {
	configOnce.Do(func() {
		// Check if Apollo configuration is available and needed
		if shouldInitializeApollo() {
			config.DefaultConfig = apollo.NewConfig(apollo.WithConfig(&agollo.Conf{
				AppID:          os.Getenv("MICRO_NAMESPACE"),
				Cluster:        "default",
				NameSpaceNames: []string{getNamespace()},
				MetaAddr:       os.Getenv("MICRO_CONFIG_ADDRESS"),
				CacheDir:       filepath.Join(os.TempDir(), "apollo"),
			}))
			configInitialized = true
		}
		// If Apollo is not available or not needed, leave DefaultConfig as nil
		// Services can check if config.DefaultConfig is nil and handle accordingly
	})
}

// shouldInitializeApollo checks if Apollo configuration should be initialized
func shouldInitializeApollo() bool {
	// Don't initialize Apollo in test environment
	if os.Getenv("GO_ENV") == "test" || os.Getenv("MICRO_CONFIG") == "memory" {
		return false
	}
	
	// Don't initialize if required Apollo environment variables are missing
	if os.Getenv("MICRO_CONFIG_ADDRESS") == "" {
		return false
	}
	
	// Don't initialize if explicitly disabled
	if os.Getenv("MICRO_CONFIG_DISABLED") == "true" {
		return false
	}
	
	return true
}

// getNamespace returns the namespace name for Apollo configuration
func getNamespace() string {
	if serverName := os.Getenv("MICRO_SERVER_NAME"); serverName != "" {
		return serverName + ".yaml"
	}
	// Fallback to application.yaml if MICRO_SERVER_NAME is not set
	return "application.yaml"
}

// EnsureConfigInitialized ensures configuration is initialized when needed
// This should be called by services that need configuration
func EnsureConfigInitialized() {
	initDefaultConfig()
}

// IsConfigInitialized returns whether Apollo configuration has been initialized
func IsConfigInitialized() bool {
	return configInitialized
}
