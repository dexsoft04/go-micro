package tls

import (
	"testing"
)

func TestNewConfig(t *testing.T) {
	config := NewConfig(
		WithCA("test-ca.pem"),
		WithCert("test-cert.pem"),
		WithKey("test-key.pem"),
		WithInsecureSkipVerify(true),
		WithServerName("test.example.com"),
	)

	if config.CA != "test-ca.pem" {
		t.Errorf("Expected CA to be 'test-ca.pem', got '%s'", config.CA)
	}

	if config.Cert != "test-cert.pem" {
		t.Errorf("Expected Cert to be 'test-cert.pem', got '%s'", config.Cert)
	}

	if config.Key != "test-key.pem" {
		t.Errorf("Expected Key to be 'test-key.pem', got '%s'", config.Key)
	}

	if !config.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be true")
	}

	if config.ServerName != "test.example.com" {
		t.Errorf("Expected ServerName to be 'test.example.com', got '%s'", config.ServerName)
	}
}

func TestBuildTLSConfig_Empty(t *testing.T) {
	config := &Config{}
	tlsConfig, err := config.BuildTLSConfig()
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if tlsConfig != nil {
		t.Error("Expected nil TLS config for empty configuration")
	}
}

func TestBuildTLSConfig_InsecureSkipVerify(t *testing.T) {
	config := &Config{
		Enabled:            true,
		InsecureSkipVerify: true,
		ServerName:         "test.example.com",
	}
	
	tlsConfig, err := config.BuildTLSConfig()
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if tlsConfig == nil {
		t.Fatal("Expected TLS config, got nil")
	}
	
	if !tlsConfig.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be true")
	}
	
	if tlsConfig.ServerName != "test.example.com" {
		t.Errorf("Expected ServerName to be 'test.example.com', got '%s'", tlsConfig.ServerName)
	}
}

func TestParseCommaSeparated(t *testing.T) {
	testCases := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"single", []string{"single"}},
		{"first,second", []string{"first", "second"}},
		{" first , second , third ", []string{"first", "second", "third"}},
		{"first,,third", []string{"first", "third"}},
	}

	for _, tc := range testCases {
		result := ParseCommaSeparated(tc.input)
		
		if len(result) != len(tc.expected) {
			t.Errorf("For input '%s', expected %d items, got %d", tc.input, len(tc.expected), len(result))
			continue
		}
		
		for i, expected := range tc.expected {
			if result[i] != expected {
				t.Errorf("For input '%s', expected item %d to be '%s', got '%s'", tc.input, i, expected, result[i])
			}
		}
	}
}

func TestFromEnvironment(t *testing.T) {
	config := FromEnvironment("test-ca", "test-cert", "test-key")
	
	if !config.Enabled {
		t.Error("Expected Enabled to be true")
	}
	
	if config.CA != "test-ca" {
		t.Errorf("Expected CA to be 'test-ca', got '%s'", config.CA)
	}
	
	if config.Cert != "test-cert" {
		t.Errorf("Expected Cert to be 'test-cert', got '%s'", config.Cert)
	}
	
	if config.Key != "test-key" {
		t.Errorf("Expected Key to be 'test-key', got '%s'", config.Key)
	}
}

func TestFromEnvironment_Empty(t *testing.T) {
	config := FromEnvironment("", "", "")
	
	if config.Enabled {
		t.Error("Expected Enabled to be false for empty environment")
	}
	
	if config.CA != "" {
		t.Errorf("Expected CA to be empty, got '%s'", config.CA)
	}
}

func TestLoadCertificate_DirectContent(t *testing.T) {
	certContent := "-----BEGIN CERTIFICATE-----\ntest content\n-----END CERTIFICATE-----"
	
	data, err := loadCertificate(certContent)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if string(data) != certContent {
		t.Errorf("Expected data to match input, got '%s'", string(data))
	}
}

func TestLoadCertificate_InvalidFile(t *testing.T) {
	_, err := loadCertificate("nonexistent-file.pem")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}