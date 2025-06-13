package registry

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp/sideband"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/networkteam/shry/config"
)

// Cache manages a local cache of Git registry clones and local directories
type Cache struct {
	// Base directory for all registry clones
	baseDir string
	// Global configuration for authentication
	globalConfig *config.GlobalConfig
	// Verbose mode
	Verbose bool
}

// NewCache creates a new Cache instance with the given base directory
func NewCache(baseDir string, globalConfig *config.GlobalConfig) (*Cache, error) {
	return &Cache{
		baseDir:      baseDir,
		globalConfig: globalConfig,
	}, nil
}

// GetRegistry returns a Registry instance for the given registry URL and reference
func (c *Cache) GetRegistry(url string, ref string, projectRoot string) (*Registry, error) {
	// Check if this is a local path
	if !isGitURL(url) {
		// Resolve the path relative to the project root
		absPath := url
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(projectRoot, absPath)
		}
		absPath, err := filepath.Abs(absPath)
		if err != nil {
			return nil, fmt.Errorf("resolving registry path: %w", err)
		}

		// Create a filesystem for the local path
		fs := osfs.New(absPath)
		return &Registry{fs: fs}, nil
	}

	// Handle Git repository
	// Create a safe directory name from the URL
	dirName := strings.ReplaceAll(url, "/", "_")
	repoPath := filepath.Join(c.baseDir, dirName)

	// Get authentication method
	auth, err := c.globalConfig.GetAuth(url)
	if err != nil {
		return nil, fmt.Errorf("getting auth: %w", err)
	}

	var progress sideband.Progress
	if c.Verbose {
		progress = os.Stderr
	}

	// Check if repository already exists
	bareRepo, err := git.PlainOpen(repoPath)
	if err == nil {
		// Repository exists, update it
		remote, err := bareRepo.Remote("origin")
		if err != nil {
			return nil, fmt.Errorf("failed to get remote: %w", err)
		}

		if err := remote.Fetch(&git.FetchOptions{
			Auth:     auth,
			Progress: progress,
			RefSpecs: []gitconfig.RefSpec{"+refs/heads/*:refs/heads/*"},
			Prune:    true,
		}); err != nil && err != git.NoErrAlreadyUpToDate {
			return nil, fmt.Errorf("failed to fetch latest changes: %w", err)
		}
		slog.Debug("Updated cache repository", "url", url, "ref", ref)
	} else if err == git.ErrRepositoryNotExists {
		var progress sideband.Progress
		if c.Verbose {
			progress = os.Stderr
		}
		// Clone the repository as bare
		bareRepo, err = git.PlainClone(repoPath, true, &git.CloneOptions{
			URL:      fmt.Sprintf("https://%s", url),
			Progress: progress,
			Auth:     auth,
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
		Progress:      progress,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	slog.Debug("Cloned cache repository to in-memory worktree", "url", url, "ref", ref, "referenceName", referenceName)

	return newRegistry(repo, fs), nil
}

// Clear removes all cached repositories
func (c *Cache) Clear() error {
	return os.RemoveAll(c.baseDir)
}

// isGitURL checks if the given URL is a Git URL
func isGitURL(url string) bool {
	return !(filepath.IsAbs(url) || filepath.VolumeName(url) != "" || url[0] == '.' || url[0] == '/')
}

// ListRegistries returns a list of registry names in the cache directory
func (c *Cache) ListRegistries() ([]string, error) {
	// Read the cache directory
	entries, err := os.ReadDir(c.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading cache directory: %w", err)
	}

	var registries []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Try to get the remote URL
		repoPath := filepath.Join(c.baseDir, entry.Name())
		repo, err := git.PlainOpen(repoPath)
		if err != nil {
			continue
		}

		remote, err := repo.Remote("origin")
		if err != nil {
			continue
		}

		// Get the remote URL from the config
		remoteURL := remote.Config().URLs[0]
		if strings.HasPrefix(remoteURL, "https://") {
			remoteURL = remoteURL[8:]
		}

		registries = append(registries, remoteURL)
	}

	return registries, nil
}
