package bootstrap

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/fluxcd/source-controller/pkg/git"
	"github.com/fluxcd/source-controller/pkg/git/strategy"
	"github.com/spf13/cobra"
)

var ErrNeedRepo = errors.New("a control-plane repository needs to be provided")
var ErrDestNoDir = errors.New("destination is not a directory")

//go:embed cp-manifests/*
var cpManifests embed.FS

type bootstrapCommand struct {
	repoURL string
}

func NewCommand() *cobra.Command {
	bcmd := bootstrapCommand{}
	var repoURL string

	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "install Flux and management cluster components",
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoURL == "" {
				return ErrNeedRepo
			}
			bcmd.repoURL = repoURL
			return bcmd.Run(context.Background())
		},
	}

	cmd.Flags().StringVar(&repoURL, "repository", "", "the repository URL to push control-plane manifests to")

	return cmd

}

func (b bootstrapCommand) Run(ctx context.Context) error {
	tmpGit, err := os.MkdirTemp("", "fluxmc-bootstrap.*")
	if err != nil {
		return fmt.Errorf("could not create temporary directory: %w", err)
	}
	// defer os.RemoveAll(tmpGit)

	checkoutStrategy, err := strategy.CheckoutStrategyForImplementation(ctx, git.Implementation("go-git"), git.CheckoutOptions{
		Branch: "main",
	})
	if err != nil {
		return fmt.Errorf("could not determine checkout strategy: %w", err)
	}
	_, err = checkoutStrategy.Checkout(ctx, tmpGit, b.repoURL, nil)
	if err != nil {
		return fmt.Errorf("could not checkout git repository: %w", err)
	}
	if err := persistContentLocally(cpManifests, "cp-manifests", filepath.Join(tmpGit, "control-plane")); err != nil {
		return fmt.Errorf("could not copy bootstrap files to local clone: %w", err)
	}

	return fmt.Errorf("not implemented, yet")
}

func persistContentLocally(sourceFS fs.FS, sourceDir string, destDir string) error {
	if walkErr := fs.WalkDir(sourceFS, sourceDir, func(path string, d fs.DirEntry, e error) error {
		if d.IsDir() {
			return nil
		}

		content, err := fs.ReadFile(sourceFS, path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", d.Name(), err)
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative file path: %w", err)
		}

		destPath := filepath.Join(destDir, relPath)
		if err := ensureDirectory(filepath.Dir(destPath)); err != nil {
			return err
		}
		if err := os.WriteFile(destPath, content, 0o600); err != nil {
			return fmt.Errorf("could not write file %s: %w", destPath, err)
		}

		return nil
	}); walkErr != nil {
		return fmt.Errorf("failed to walk local repo: %w", walkErr)
	}

	return nil
}

func ensureDirectory(dir string) error {
	stat, err := os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("could not stat %s: %w", dir, err)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("could not create dir %s: %w", dir, err)
		}
		return nil
	}
	if !stat.IsDir() {
		return ErrDestNoDir
	}
	return nil
}
