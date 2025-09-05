// Package tls provides TLS configuration utilities for go-micro components
package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"strings"
)

// Config represents TLS configuration for a component
type Config struct {
	// Enable TLS
	Enabled bool
	// CA certificate file path or content
	CA string
	// Certificate file path or content
	Cert string
	// Private key file path or content
	Key string
	// Skip certificate verification (for testing)
	InsecureSkipVerify bool
	// Server name for certificate verification
	ServerName string
}

// Options for TLS configuration
type Option func(*Config)

// WithCA sets the CA certificate
func WithCA(ca string) Option {
	return func(c *Config) {
		c.CA = ca
	}
}

// WithCert sets the client certificate
func WithCert(cert string) Option {
	return func(c *Config) {
		c.Cert = cert
	}
}

// WithKey sets the private key
func WithKey(key string) Option {
	return func(c *Config) {
		c.Key = key
	}
}

// WithInsecureSkipVerify skips certificate verification
func WithInsecureSkipVerify(skip bool) Option {
	return func(c *Config) {
		c.InsecureSkipVerify = skip
	}
}

// WithServerName sets the server name for certificate verification
func WithServerName(name string) Option {
	return func(c *Config) {
		c.ServerName = name
	}
}

// NewConfig creates a new TLS configuration
func NewConfig(opts ...Option) *Config {
	config := &Config{}
	for _, opt := range opts {
		opt(config)
	}
	return config
}

// BuildTLSConfig builds a crypto/tls.Config from the TLS configuration
func (c *Config) BuildTLSConfig() (*tls.Config, error) {
	if !c.Enabled && c.CA == "" && c.Cert == "" && c.Key == "" {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.InsecureSkipVerify,
		ServerName:         c.ServerName,
	}

	// Load CA certificate if provided
	if c.CA != "" {
		caCert, err := loadCertificate(c.CA)
		if err != nil {
			return nil, fmt.Errorf("failed to load CA certificate: %v", err)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}

	// Load client certificate and key if provided
	if c.Cert != "" && c.Key != "" {
		cert, err := loadCertificate(c.Cert)
		if err != nil {
			return nil, fmt.Errorf("failed to load certificate: %v", err)
		}

		key, err := loadCertificate(c.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to load private key: %v", err)
		}

		clientCert, err := tls.X509KeyPair(cert, key)
		if err != nil {
			return nil, fmt.Errorf("failed to create X509 key pair: %v", err)
		}

		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}

	return tlsConfig, nil
}

// loadCertificate loads a certificate from file path or direct content
func loadCertificate(certData string) ([]byte, error) {
	// Check if it's a file path or direct content
	if strings.Contains(certData, "-----BEGIN") {
		// Direct certificate content
		return []byte(certData), nil
	}

	// Assume it's a file path
	data, err := ioutil.ReadFile(certData)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file %s: %v", certData, err)
	}

	return data, nil
}

// ParseCommaSeparated parses comma-separated certificate paths/content
func ParseCommaSeparated(input string) []string {
	if input == "" {
		return nil
	}

	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// FromEnvironment creates TLS config from environment variables
func FromEnvironment(caEnv, certEnv, keyEnv string) *Config {
	config := &Config{}
	
	if ca := strings.TrimSpace(caEnv); ca != "" {
		config.CA = ca
		config.Enabled = true
	}
	
	if cert := strings.TrimSpace(certEnv); cert != "" {
		config.Cert = cert
		config.Enabled = true
	}
	
	if key := strings.TrimSpace(keyEnv); key != "" {
		config.Key = key
		config.Enabled = true
	}
	
	return config
}