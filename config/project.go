package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// ProjectConfigFile is the name of the project configuration file
	ProjectConfigFile = ".shry.yaml"
)

// ProjectConfig represents a project configuration
type ProjectConfig struct {
	// ProjectDir is the directory containing the .shry.yaml file
	ProjectDir string `yaml:"-"`
	// Registry path (relative or absolute)
	Registry string `yaml:"registry"`
	// Platform this project is for
	Platform string `yaml:"platform"`
	// Variables to substitute for component templates
	Variables map[string]any `yaml:"variables"`
}

// findNearestProjectConfigDir finds the nearest .shry.yaml file by walking up the directory tree
func findNearestProjectConfigDir(startPath string) (string, error) {
	dir := startPath
	for {
		configPath := filepath.Join(dir, ProjectConfigFile)
		if _, err := os.Stat(configPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no project config found in %s or any parent directory", startPath)
		}
		dir = parent
	}
}

// FindNearestProjectConfig finds and loads the nearest project configuration
func FindNearestProjectConfig() (*ProjectConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting current directory: %w", err)
	}

	projectDir, err := findNearestProjectConfigDir(cwd)
	if err != nil {
		return nil, err
	}

	return LoadProjectConfig(projectDir)
}

// LoadProjectConfig loads a project configuration from a file
func LoadProjectConfig(path string) (*ProjectConfig, error) {
	// Read the project configuration file
	file, err := os.Open(filepath.Join(path, ProjectConfigFile))
	if err != nil {
		return nil, fmt.Errorf("opening project config: %w", err)
	}
	defer file.Close()

	// Parse the YAML configuration
	var config ProjectConfig
	if err := yaml.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("parsing project config: %w", err)
	}

	// Set the project directory
	config.ProjectDir = path

	// Validate required fields
	if config.Registry == "" {
		return nil, fmt.Errorf("project registry is required")
	}
	if config.Platform == "" {
		return nil, fmt.Errorf("project platform is required")
	}

	return &config, nil
}
