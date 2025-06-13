package registry

import (
	"fmt"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Registry represents a component registry repository
type Registry struct {
	repo *git.Repository
	fs   billy.Filesystem
}

// NewRegistry creates a new Registry instance from a Git repository
func NewRegistry(repo *git.Repository, fs billy.Filesystem) (*Registry, error) {
	return &Registry{
		repo: repo,
		fs:   fs,
	}, nil
}

// Update pulls the latest changes from the remote repository
func (r *Registry) Update() error {
	wt, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	if err := wt.Pull(&git.PullOptions{}); err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to pull latest changes: %w", err)
	}
	return nil
}

// Checkout switches to the specified reference (branch, tag, or commit)
func (r *Registry) Checkout(ref string) error {
	wt, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Try to resolve the reference
	hash, err := r.repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return fmt.Errorf("failed to resolve reference %s: %w", ref, err)
	}

	if err := wt.Checkout(&git.CheckoutOptions{
		Hash: *hash,
	}); err != nil {
		return fmt.Errorf("failed to checkout reference %s: %w", ref, err)
	}

	return nil
}
