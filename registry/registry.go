package registry

import (
	"fmt"
	"io"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/networkteam/shry/config"
)

// Registry represents a component registry that can be either a Git repository or a local directory
type Registry struct {
	repo *git.Repository // nil for local directories
	fs   billy.Filesystem
}

// newRegistry creates a new Registry instance
func newRegistry(repo *git.Repository, fs billy.Filesystem) *Registry {
	return &Registry{
		repo: repo,
		fs:   fs,
	}
}

// IsGit returns true if this is a Git-based registry
func (r *Registry) IsGit() bool {
	return r.repo != nil
}

// ReadFile reads a file from the registry
func (r *Registry) ReadFile(path string) ([]byte, error) {
	file, err := r.fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	return io.ReadAll(file)
}

// ScanComponents scans the registry for components
func (r *Registry) ScanComponents() (map[string]map[string]*config.Component, error) {
	return config.ScanComponents(r.fs, ".")
}

// ResolveComponent resolves a component by name for the given platform and verifies its variables
func (r *Registry) ResolveComponent(platform, name string) (*config.Component, error) {
	// Scan components
	components, err := r.ScanComponents()
	if err != nil {
		return nil, fmt.Errorf("scanning components: %w", err)
	}

	// Lookup platform components
	platformComponents, exists := components[platform]
	if !exists {
		return nil, fmt.Errorf("no components found for platform %s", platform)
	}

	// Lookup component
	component, exists := platformComponents[name]
	if !exists {
		return nil, fmt.Errorf("component %s not found for platform %s", name, platform)
	}

	return component, nil
}
