package gogit

import (
	"context"
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"

	"github.com/makkes/fluxmc/common/gitops"
)

var _ gitops.Repository = &GoGitRepo{}

type GoGitRepo struct {
	url   string
	auth  transport.AuthMethod
	depth int
	repo  *git.Repository
}

type Option func(r *GoGitRepo) error

func Depth(d int) Option {
	return func(r *GoGitRepo) error {
		r.depth = d
		return nil
	}
}

func BasicAuth(username, password string) Option {
	return func(r *GoGitRepo) error {
		r.auth = &http.BasicAuth{
			Username: username,
			Password: password,
		}
		return nil
	}
}

func NewGoGitRepo(url string, opts ...Option) (*GoGitRepo, error) {
	r := &GoGitRepo{
		url: url,
	}

	for _, opt := range opts {
		if err := opt(r); err != nil {
			return nil, fmt.Errorf("could not apply option: %w", err)
		}
	}

	return r, nil
}

func (r *GoGitRepo) Clone(ctx context.Context, dir string) error {
	repo, err := git.PlainCloneContext(ctx, dir, false, &git.CloneOptions{
		Auth:            r.auth,
		URL:             r.url,
		Depth:           r.depth,
		InsecureSkipTLS: true,
	})
	if err != nil {
		return fmt.Errorf("could not clone repo: %w", err)
	}

	r.repo = repo

	return nil
}

func (r GoGitRepo) Add(path string) error {
	wt, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("could not get worktree: %w", err)
	}
	_, err = wt.Add(path)
	if err != nil {
		return fmt.Errorf("could not add '%s' to index: %w", path, err)
	}

	return nil
}

func (r GoGitRepo) Remove(path string) error {
	wt, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("could not get worktree: %w", err)
	}

	_, err = wt.Remove(path)
	if err != nil {
		return fmt.Errorf("could not remove '%s': %w", path, err)
	}

	return nil
}

func (r GoGitRepo) Commit(msg, name, email string) error {
	wt, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("could not get worktree: %w", err)
	}
	_, err = wt.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  name,
			Email: email,
			When:  time.Now(),
		},
	})

	return err
}

func (r GoGitRepo) Push(ctx context.Context) error {
	return r.repo.PushContext(ctx, &git.PushOptions{
		InsecureSkipTLS: true,
		Auth:            r.auth,
	})
}
