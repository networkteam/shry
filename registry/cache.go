package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
)

// Cache manages a local cache of Git registry clones
type Cache struct {
	// Base directory for all registry clones
	baseDir string
}

// NewCache creates a new Cache instance with the given base directory
func NewCache(baseDir string) (*Cache, error) {
	// Ensure the base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &Cache{
		baseDir: baseDir,
	}, nil
}

// GetRegistry returns a Registry instance for the given registry URL and reference
func (c *Cache) GetRegistry(url string, ref string) (*Registry, error) {
	// Create a safe directory name from the URL
	dirName := strings.ReplaceAll(url, "/", "_")
	repoPath := filepath.Join(c.baseDir, dirName)

	// Check if repository already exists
	bareRepo, err := git.PlainOpen(repoPath)
	if err == nil {
		// Repository exists, update it
		remote, err := bareRepo.Remote("origin")
		if err != nil {
			return nil, fmt.Errorf("failed to get remote: %w", err)
		}

		if err := remote.Fetch(&git.FetchOptions{}); err != nil && err != git.NoErrAlreadyUpToDate {
			return nil, fmt.Errorf("failed to fetch latest changes: %w", err)
		}
	} else if err == git.ErrRepositoryNotExists {
		// Clone the repository as bare
		bareRepo, err = git.PlainClone(repoPath, true, &git.CloneOptions{
			URL:      fmt.Sprintf("https://%s", url),
			Progress: os.Stdout,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to clone repository: %w", err)
		}
	} else {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Resolve the reference from the bare repository
	var referenceName plumbing.ReferenceName
	if ref != "" {
		hash, err := bareRepo.ResolveRevision(plumbing.Revision(ref))
		if err != nil {
			return nil, fmt.Errorf("failed to resolve reference %s: %w", ref, err)
		}
		referenceName = plumbing.NewHashReference("HEAD", *hash).Name()
	} else {
		referenceName = plumbing.HEAD
	}

	// Create a new repository with in-memory storage and filesystem
	fs := memfs.New()
	repo, err := git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
		URL:           repoPath,
		SingleBranch:  true,
		Depth:         1,
		ReferenceName: referenceName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Create Registry instance
	registry, err := NewRegistry(repo, fs)
	if err != nil {
		return nil, err
	}

	return registry, nil
}

// Clear removes all cached repositories
func (c *Cache) Clear() error {
	return os.RemoveAll(c.baseDir)
}
