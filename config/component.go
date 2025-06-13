package config

import (
	"fmt"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/networkteam/shry/template"
	"gopkg.in/yaml.v3"
)

const (
	// ComponentConfigFile is the name of the component configuration file
	ComponentConfigFile = "shry.yaml"
)

// ComponentKey represents a unique key for a component in a registry
type ComponentKey struct {
	Platform string
	Name     string
}

// Component represents a component configuration
type Component struct {
	// Path is the directory path of the component in the filesystem
	Path string `yaml:"-"`
	// Name of the component, will be used to reference the component (must be unique within the registry per platform)
	Name string `yaml:"name"`
	// Optional title (e.g. image-card vs. "Image Card")
	Title string `yaml:"title,omitempty"`
	// Optional description
	Description string `yaml:"description,omitempty"`
	// Platform this component is for (required)
	Platform string `yaml:"platform"`
	// Optional preview image and demo URL
	Preview struct {
		Image string `yaml:"image,omitempty"`
		Demo  string `yaml:"demo,omitempty"`
	} `yaml:"preview,omitempty"`
	// Default variables for the component (optional)
	Variables map[string]any `yaml:"variables,omitempty"`
	// Files to copy when adding the component to a project
	Files []File `yaml:"files"`
}

// File represents a file to be copied when adding a component
type File struct {
	// Src file relative to the component directory
	Src string `yaml:"src"`
	// Destination path (filename with variables)
	Dst string `yaml:"dst"`
}

// LoadComponent loads a component configuration from a filesystem
func LoadComponent(fs billy.Filesystem, path string) (*Component, error) {
	// Read the component configuration file
	file, err := fs.Open(filepath.Join(path, ComponentConfigFile))
	if err != nil {
		return nil, fmt.Errorf("opening component config: %w", err)
	}
	defer file.Close()

	// Parse the YAML configuration
	var component Component
	if err := yaml.NewDecoder(file).Decode(&component); err != nil {
		return nil, fmt.Errorf("parsing component config: %w", err)
	}

	// Set the component path
	component.Path = path

	// Validate required fields
	if component.Name == "" {
		return nil, fmt.Errorf("component name is required")
	}
	if component.Platform == "" {
		return nil, fmt.Errorf("component platform is required")
	}

	return &component, nil
}

// ScanComponents scans a directory recursively for component configurations
func ScanComponents(fs billy.Filesystem, path string) (map[string]map[string]*Component, error) {
	components := make(map[string]map[string]*Component)

	// Helper function to scan a directory
	var scanDir func(dir string) error
	scanDir = func(dir string) error {
		entries, err := fs.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("failed to read directory %s: %w", dir, err)
		}

		for _, entry := range entries {
			entryPath := filepath.Join(dir, entry.Name())

			if entry.IsDir() {
				// Check if this directory contains a component configuration
				if _, err := fs.Stat(filepath.Join(entryPath, ComponentConfigFile)); err == nil {
					// Load the component configuration
					component, err := LoadComponent(fs, entryPath)
					if err != nil {
						return fmt.Errorf("failed to load component in %s: %w", entryPath, err)
					}

					// Initialize platform map if it doesn't exist
					if _, exists := components[component.Platform]; !exists {
						components[component.Platform] = make(map[string]*Component)
					}

					// Check for duplicate component name within the platform
					if _, exists := components[component.Platform][component.Name]; exists {
						return fmt.Errorf("duplicate component %s found in platform %s at %s", component.Name, component.Platform, entryPath)
					}

					components[component.Platform][component.Name] = component
				} else {
					// Recursively scan subdirectories
					if err := scanDir(entryPath); err != nil {
						return err
					}
				}
			}
		}

		return nil
	}

	// Start scanning from the given path
	if err := scanDir(path); err != nil {
		return nil, err
	}

	return components, nil
}

// ResolveFiles resolves all variables in the component's files
func (c *Component) ResolveFiles(variables map[string]any) ([]File, error) {
	var resolvedFiles []File

	// Collect all variables from file paths and content
	requiredVars := make(map[string]bool)
	for _, file := range c.Files {
		// Check variables in destination path
		for _, varName := range template.FindVariables(file.Dst) {
			requiredVars[varName] = true
		}
	}

	// Verify all required variables are defined
	for varName := range requiredVars {
		if _, exists := variables[varName]; !exists {
			return nil, fmt.Errorf("required variable %s not defined", varName)
		}
	}

	// Resolve variables in each file
	for _, file := range c.Files {
		// Resolve destination path
		dst, err := template.Resolve(file.Dst, variables)
		if err != nil {
			return nil, fmt.Errorf("resolving destination path %s: %w", file.Dst, err)
		}

		resolvedFiles = append(resolvedFiles, File{
			Src: file.Src,
			Dst: dst,
		})
	}

	return resolvedFiles, nil
}
