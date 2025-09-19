package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"gopkg.in/yaml.v3"
)

const (
	// GlobalConfigFile is the name of the global configuration file
	GlobalConfigFile = "config.yaml"
)

// GlobalConfig represents the global configuration
type GlobalConfig struct {
	// ConfigPath is the path to the global configuration file
	ConfigPath string `yaml:"-"`
	// Registries contains authentication information for each registry
	Registries map[string]RegistryConfig `yaml:"registries"`
}

// RegistryConfig contains authentication information for a registry
type RegistryConfig struct {
	// HTTP authentication
	HTTP *HTTPAuth `yaml:"http,omitempty"`
	// SSH authentication
	SSH *SSHAuth `yaml:"ssh,omitempty"`
}

// HTTPAuth contains HTTP authentication information
type HTTPAuth struct {
	// Username for HTTP authentication
	Username string `yaml:"username"`
	// Password or token for HTTP authentication
	Password string `yaml:"password"`
}

// SSHAuth contains SSH authentication information
type SSHAuth struct {
	// Path to the private key file
	PrivateKeyPath string `yaml:"privateKeyPath"`
	// Password for the private key (if encrypted)
	Password string `yaml:"password,omitempty"`
}

// LoadGlobalConfig loads the global configuration
func LoadGlobalConfig(configPath string) (*GlobalConfig, error) {
	// Read the global configuration file
	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config if it doesn't exist
			config := &GlobalConfig{
				Registries: make(map[string]RegistryConfig),
				ConfigPath: configPath,
			}
			return config, nil
		}
		return nil, fmt.Errorf("opening global config: %w", err)
	}
	defer file.Close()

	// Parse the YAML configuration
	var config GlobalConfig
	if err := yaml.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("parsing global config: %w", err)
	}

	config.ConfigPath = configPath

	return &config, nil
}

// Save saves the global configuration
func (c *GlobalConfig) Save() error {
	configPath := c.ConfigPath

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("creating global config directory: %w", err)
	}

	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("creating global config: %w", err)
	}
	defer file.Close()

	// Encode the configuration as YAML
	if err := yaml.NewEncoder(file).Encode(c); err != nil {
		return fmt.Errorf("writing global config: %w", err)
	}

	return nil
}

// GetAuth returns the authentication method for the given registry URL
func (c *GlobalConfig) GetAuth(registryURL string) (transport.AuthMethod, error) {
	// Look up registry configuration
	config, exists := c.Registries[registryURL]
	if !exists {
		return nil, nil
	}

	// Try HTTP authentication first
	if config.HTTP != nil {
		return &http.BasicAuth{
			Username: config.HTTP.Username,
			Password: config.HTTP.Password,
		}, nil
	}

	// Try SSH authentication
	if config.SSH != nil {
		// Read private key
		key, err := os.ReadFile(config.SSH.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("reading private key: %w", err)
		}

		// Create SSH auth
		auth, err := ssh.NewPublicKeys("git", key, config.SSH.Password)
		if err != nil {
			return nil, fmt.Errorf("creating SSH auth: %w", err)
		}

		return auth, nil
	}

	return nil, nil
}

// RegistryLocations returns a sorted list of registry locations
func (c *GlobalConfig) RegistryLocations() ([]string, error) {
	locations := make([]string, 0, len(c.Registries))
	for location := range c.Registries {
		locations = append(locations, location)
	}
	slices.Sort(locations)
	return locations, nil
}
