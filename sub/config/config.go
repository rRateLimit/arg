package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

// Config represents rate limiter configuration
type Config struct {
	Rate            int           `json:"rate"`
	Burst           int           `json:"burst"`
	Window          time.Duration `json:"window,omitempty"`
	Name            string        `json:"name,omitempty"`
	Enabled         bool          `json:"enabled"`
	PerKeyLimits    bool          `json:"per_key_limits,omitempty"`
	ErrorMessage    string        `json:"error_message,omitempty"`
	ExcludedPaths   []string      `json:"excluded_paths,omitempty"`
	ExcludedIPs     []string      `json:"excluded_ips,omitempty"`
	CustomHeaders   map[string]string `json:"custom_headers,omitempty"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Rate:         10,
		Burst:        20,
		Window:       time.Second,
		Enabled:      true,
		ErrorMessage: "Too Many Requests",
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Rate <= 0 {
		return errors.New("rate must be positive")
	}
	if c.Burst <= 0 {
		return errors.New("burst must be positive")
	}
	if c.Burst < c.Rate {
		return errors.New("burst must be greater than or equal to rate")
	}
	if c.Window < 0 {
		return errors.New("window must be non-negative")
	}
	return nil
}

// LoadFromFile loads configuration from a JSON file
func LoadFromFile(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()
	
	return LoadFromReader(file)
}

// LoadFromReader loads configuration from an io.Reader
func LoadFromReader(r io.Reader) (*Config, error) {
	config := DefaultConfig()
	
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}
	
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	
	return config, nil
}

// SaveToFile saves configuration to a JSON file
func (c *Config) SaveToFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()
	
	return c.SaveToWriter(file)
}

// SaveToWriter saves configuration to an io.Writer
func (c *Config) SaveToWriter(w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}
	
	return nil
}

// Clone creates a deep copy of the configuration
func (c *Config) Clone() *Config {
	clone := *c
	
	if c.ExcludedPaths != nil {
		clone.ExcludedPaths = make([]string, len(c.ExcludedPaths))
		copy(clone.ExcludedPaths, c.ExcludedPaths)
	}
	
	if c.ExcludedIPs != nil {
		clone.ExcludedIPs = make([]string, len(c.ExcludedIPs))
		copy(clone.ExcludedIPs, c.ExcludedIPs)
	}
	
	if c.CustomHeaders != nil {
		clone.CustomHeaders = make(map[string]string, len(c.CustomHeaders))
		for k, v := range c.CustomHeaders {
			clone.CustomHeaders[k] = v
		}
	}
	
	return &clone
}

// ConfigSet represents a collection of named configurations
type ConfigSet struct {
	configs map[string]*Config
}

// NewConfigSet creates a new configuration set
func NewConfigSet() *ConfigSet {
	return &ConfigSet{
		configs: make(map[string]*Config),
	}
}

// Add adds a configuration to the set
func (cs *ConfigSet) Add(name string, config *Config) error {
	if name == "" {
		return errors.New("config name cannot be empty")
	}
	if config == nil {
		return errors.New("config cannot be nil")
	}
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config for %s: %w", name, err)
	}
	
	cs.configs[name] = config
	return nil
}

// Get retrieves a configuration by name
func (cs *ConfigSet) Get(name string) (*Config, bool) {
	config, exists := cs.configs[name]
	return config, exists
}

// Remove removes a configuration from the set
func (cs *ConfigSet) Remove(name string) {
	delete(cs.configs, name)
}

// Names returns all configuration names in the set
func (cs *ConfigSet) Names() []string {
	names := make([]string, 0, len(cs.configs))
	for name := range cs.configs {
		names = append(names, name)
	}
	return names
}

// LoadFromFile loads a configuration set from a JSON file
func (cs *ConfigSet) LoadFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open config set file: %w", err)
	}
	defer file.Close()
	
	var configs map[string]*Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&configs); err != nil {
		return fmt.Errorf("failed to decode config set: %w", err)
	}
	
	for name, config := range configs {
		if err := cs.Add(name, config); err != nil {
			return fmt.Errorf("failed to add config %s: %w", name, err)
		}
	}
	
	return nil
}

// SaveToFile saves the configuration set to a JSON file
func (cs *ConfigSet) SaveToFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create config set file: %w", err)
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	
	if err := encoder.Encode(cs.configs); err != nil {
		return fmt.Errorf("failed to encode config set: %w", err)
	}
	
	return nil
}

// Builder provides a fluent interface for building configurations
type Builder struct {
	config *Config
}

// NewBuilder creates a new configuration builder
func NewBuilder() *Builder {
	return &Builder{
		config: DefaultConfig(),
	}
}

// WithRate sets the rate
func (b *Builder) WithRate(rate int) *Builder {
	b.config.Rate = rate
	return b
}

// WithBurst sets the burst
func (b *Builder) WithBurst(burst int) *Builder {
	b.config.Burst = burst
	return b
}

// WithWindow sets the window duration
func (b *Builder) WithWindow(window time.Duration) *Builder {
	b.config.Window = window
	return b
}

// WithName sets the name
func (b *Builder) WithName(name string) *Builder {
	b.config.Name = name
	return b
}

// WithEnabled sets the enabled state
func (b *Builder) WithEnabled(enabled bool) *Builder {
	b.config.Enabled = enabled
	return b
}

// WithPerKeyLimits enables per-key limits
func (b *Builder) WithPerKeyLimits(enabled bool) *Builder {
	b.config.PerKeyLimits = enabled
	return b
}

// WithErrorMessage sets the error message
func (b *Builder) WithErrorMessage(message string) *Builder {
	b.config.ErrorMessage = message
	return b
}

// WithExcludedPaths sets the excluded paths
func (b *Builder) WithExcludedPaths(paths ...string) *Builder {
	b.config.ExcludedPaths = paths
	return b
}

// WithExcludedIPs sets the excluded IPs
func (b *Builder) WithExcludedIPs(ips ...string) *Builder {
	b.config.ExcludedIPs = ips
	return b
}

// WithCustomHeaders sets custom headers
func (b *Builder) WithCustomHeaders(headers map[string]string) *Builder {
	b.config.CustomHeaders = headers
	return b
}

// Build validates and returns the configuration
func (b *Builder) Build() (*Config, error) {
	if err := b.config.Validate(); err != nil {
		return nil, err
	}
	return b.config.Clone(), nil
}