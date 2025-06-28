package config

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config.Rate != 10 {
		t.Errorf("Expected Rate to be 10, got %d", config.Rate)
	}
	if config.Burst != 20 {
		t.Errorf("Expected Burst to be 20, got %d", config.Burst)
	}
	if config.Window != time.Second {
		t.Errorf("Expected Window to be 1s, got %v", config.Window)
	}
	if !config.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if config.ErrorMessage != "Too Many Requests" {
		t.Errorf("Expected default error message, got %q", config.ErrorMessage)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				Rate:  10,
				Burst: 20,
			},
			wantErr: false,
		},
		{
			name: "negative rate",
			config: &Config{
				Rate:  -1,
				Burst: 10,
			},
			wantErr: true,
			errMsg:  "rate must be positive",
		},
		{
			name: "zero rate",
			config: &Config{
				Rate:  0,
				Burst: 10,
			},
			wantErr: true,
			errMsg:  "rate must be positive",
		},
		{
			name: "negative burst",
			config: &Config{
				Rate:  10,
				Burst: -1,
			},
			wantErr: true,
			errMsg:  "burst must be positive",
		},
		{
			name: "burst less than rate",
			config: &Config{
				Rate:  20,
				Burst: 10,
			},
			wantErr: true,
			errMsg:  "burst must be greater than or equal to rate",
		},
		{
			name: "negative window",
			config: &Config{
				Rate:   10,
				Burst:  20,
				Window: -time.Second,
			},
			wantErr: true,
			errMsg:  "window must be non-negative",
		},
		{
			name: "burst equals rate",
			config: &Config{
				Rate:  10,
				Burst: 10,
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestLoadFromReader(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
		check   func(t *testing.T, c *Config)
	}{
		{
			name: "valid json",
			json: `{
				"rate": 50,
				"burst": 100,
				"window": 2000000000,
				"name": "test-limiter",
				"enabled": true,
				"error_message": "Custom error"
			}`,
			wantErr: false,
			check: func(t *testing.T, c *Config) {
				if c.Rate != 50 {
					t.Errorf("Expected Rate 50, got %d", c.Rate)
				}
				if c.Burst != 100 {
					t.Errorf("Expected Burst 100, got %d", c.Burst)
				}
				if c.Window != 2*time.Second {
					t.Errorf("Expected Window 2s, got %v", c.Window)
				}
				if c.Name != "test-limiter" {
					t.Errorf("Expected Name 'test-limiter', got %q", c.Name)
				}
				if c.ErrorMessage != "Custom error" {
					t.Errorf("Expected custom error message, got %q", c.ErrorMessage)
				}
			},
		},
		{
			name:    "invalid json",
			json:    `{"rate": "not a number"}`,
			wantErr: true,
		},
		{
			name:    "invalid config",
			json:    `{"rate": -1, "burst": 10}`,
			wantErr: true,
		},
		{
			name: "partial config uses defaults",
			json: `{"rate": 5}`,
			wantErr: false,
			check: func(t *testing.T, c *Config) {
				if c.Rate != 5 {
					t.Errorf("Expected Rate 5, got %d", c.Rate)
				}
				if c.Burst != 20 {
					t.Errorf("Expected default Burst 20, got %d", c.Burst)
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.json)
			config, err := LoadFromReader(reader)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromReader() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if !tt.wantErr && tt.check != nil {
				tt.check(t, config)
			}
		})
	}
}

func TestSaveToWriter(t *testing.T) {
	config := &Config{
		Rate:         30,
		Burst:        60,
		Window:       5 * time.Second,
		Name:         "test",
		Enabled:      true,
		ErrorMessage: "Rate limited",
		ExcludedPaths: []string{"/health", "/metrics"},
		CustomHeaders: map[string]string{
			"X-RateLimit-Limit": "60",
			"Retry-After":       "5",
		},
	}
	
	var buf bytes.Buffer
	err := config.SaveToWriter(&buf)
	if err != nil {
		t.Fatalf("SaveToWriter() error = %v", err)
	}
	
	// Load it back and verify
	loaded, err := LoadFromReader(&buf)
	if err != nil {
		t.Fatalf("LoadFromReader() error = %v", err)
	}
	
	if loaded.Rate != config.Rate {
		t.Errorf("Rate mismatch: got %d, want %d", loaded.Rate, config.Rate)
	}
	if loaded.Burst != config.Burst {
		t.Errorf("Burst mismatch: got %d, want %d", loaded.Burst, config.Burst)
	}
	if loaded.Name != config.Name {
		t.Errorf("Name mismatch: got %q, want %q", loaded.Name, config.Name)
	}
	if !reflect.DeepEqual(loaded.ExcludedPaths, config.ExcludedPaths) {
		t.Errorf("ExcludedPaths mismatch: got %v, want %v", loaded.ExcludedPaths, config.ExcludedPaths)
	}
	if !reflect.DeepEqual(loaded.CustomHeaders, config.CustomHeaders) {
		t.Errorf("CustomHeaders mismatch: got %v, want %v", loaded.CustomHeaders, config.CustomHeaders)
	}
}

func TestConfigClone(t *testing.T) {
	original := &Config{
		Rate:          10,
		Burst:         20,
		Name:          "original",
		ExcludedPaths: []string{"/path1", "/path2"},
		ExcludedIPs:   []string{"192.168.1.1", "192.168.1.2"},
		CustomHeaders: map[string]string{
			"Header1": "Value1",
			"Header2": "Value2",
		},
	}
	
	clone := original.Clone()
	
	// Verify values are copied
	if clone.Rate != original.Rate {
		t.Errorf("Rate not cloned correctly")
	}
	if clone.Name != original.Name {
		t.Errorf("Name not cloned correctly")
	}
	
	// Verify slices are deep copied
	clone.ExcludedPaths[0] = "/modified"
	if original.ExcludedPaths[0] == "/modified" {
		t.Error("ExcludedPaths not deep copied")
	}
	
	clone.ExcludedIPs[0] = "10.0.0.1"
	if original.ExcludedIPs[0] == "10.0.0.1" {
		t.Error("ExcludedIPs not deep copied")
	}
	
	// Verify map is deep copied
	clone.CustomHeaders["Header1"] = "ModifiedValue"
	if original.CustomHeaders["Header1"] == "ModifiedValue" {
		t.Error("CustomHeaders not deep copied")
	}
}

func TestConfigSet(t *testing.T) {
	cs := NewConfigSet()
	
	// Test adding configs
	config1 := &Config{Rate: 10, Burst: 20}
	config2 := &Config{Rate: 5, Burst: 10}
	
	err := cs.Add("api", config1)
	if err != nil {
		t.Errorf("Add() error = %v", err)
	}
	
	err = cs.Add("admin", config2)
	if err != nil {
		t.Errorf("Add() error = %v", err)
	}
	
	// Test getting configs
	got, exists := cs.Get("api")
	if !exists {
		t.Error("Expected 'api' config to exist")
	}
	if got.Rate != 10 {
		t.Errorf("Expected Rate 10, got %d", got.Rate)
	}
	
	// Test non-existent config
	_, exists = cs.Get("nonexistent")
	if exists {
		t.Error("Expected 'nonexistent' config to not exist")
	}
	
	// Test names
	names := cs.Names()
	if len(names) != 2 {
		t.Errorf("Expected 2 names, got %d", len(names))
	}
	
	// Test remove
	cs.Remove("api")
	_, exists = cs.Get("api")
	if exists {
		t.Error("Expected 'api' config to be removed")
	}
	
	// Test adding invalid config
	err = cs.Add("", config1)
	if err == nil {
		t.Error("Expected error for empty name")
	}
	
	err = cs.Add("invalid", &Config{Rate: -1, Burst: 10})
	if err == nil {
		t.Error("Expected error for invalid config")
	}
}

func TestConfigSetFileOperations(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "configset.json")
	
	// Create and save a config set
	cs := NewConfigSet()
	cs.Add("default", &Config{Rate: 10, Burst: 20})
	cs.Add("premium", &Config{Rate: 100, Burst: 200})
	
	err := cs.SaveToFile(filename)
	if err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}
	
	// Load it back
	cs2 := NewConfigSet()
	err = cs2.LoadFromFile(filename)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	
	// Verify contents
	config, exists := cs2.Get("default")
	if !exists {
		t.Error("Expected 'default' config to exist")
	}
	if config.Rate != 10 {
		t.Errorf("Expected Rate 10, got %d", config.Rate)
	}
	
	config, exists = cs2.Get("premium")
	if !exists {
		t.Error("Expected 'premium' config to exist")
	}
	if config.Rate != 100 {
		t.Errorf("Expected Rate 100, got %d", config.Rate)
	}
}

func TestBuilder(t *testing.T) {
	config, err := NewBuilder().
		WithRate(50).
		WithBurst(100).
		WithWindow(5 * time.Second).
		WithName("test-builder").
		WithEnabled(true).
		WithPerKeyLimits(true).
		WithErrorMessage("Custom rate limit error").
		WithExcludedPaths("/health", "/status").
		WithExcludedIPs("127.0.0.1", "::1").
		WithCustomHeaders(map[string]string{
			"X-Custom": "value",
		}).
		Build()
	
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	
	if config.Rate != 50 {
		t.Errorf("Expected Rate 50, got %d", config.Rate)
	}
	if config.Burst != 100 {
		t.Errorf("Expected Burst 100, got %d", config.Burst)
	}
	if config.Window != 5*time.Second {
		t.Errorf("Expected Window 5s, got %v", config.Window)
	}
	if config.Name != "test-builder" {
		t.Errorf("Expected Name 'test-builder', got %q", config.Name)
	}
	if !config.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if !config.PerKeyLimits {
		t.Error("Expected PerKeyLimits to be true")
	}
	if config.ErrorMessage != "Custom rate limit error" {
		t.Errorf("Expected custom error message, got %q", config.ErrorMessage)
	}
	if len(config.ExcludedPaths) != 2 {
		t.Errorf("Expected 2 excluded paths, got %d", len(config.ExcludedPaths))
	}
	if len(config.ExcludedIPs) != 2 {
		t.Errorf("Expected 2 excluded IPs, got %d", len(config.ExcludedIPs))
	}
	if config.CustomHeaders["X-Custom"] != "value" {
		t.Error("Expected custom header not found")
	}
}

func TestBuilderInvalidConfig(t *testing.T) {
	_, err := NewBuilder().
		WithRate(-1).
		Build()
	
	if err == nil {
		t.Error("Expected error for invalid rate")
	}
}

func TestLoadFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "config.json")
	
	// Create a test config file
	testConfig := &Config{
		Rate:  25,
		Burst: 50,
		Name:  "file-test",
	}
	
	err := testConfig.SaveToFile(filename)
	if err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}
	
	// Load it back
	loaded, err := LoadFromFile(filename)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	
	if loaded.Rate != 25 {
		t.Errorf("Expected Rate 25, got %d", loaded.Rate)
	}
	if loaded.Name != "file-test" {
		t.Errorf("Expected Name 'file-test', got %q", loaded.Name)
	}
	
	// Test non-existent file
	_, err = LoadFromFile("/non/existent/file.json")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestConfigSetAddNil(t *testing.T) {
	cs := NewConfigSet()
	err := cs.Add("nil-config", nil)
	if err == nil {
		t.Error("Expected error when adding nil config")
	}
}