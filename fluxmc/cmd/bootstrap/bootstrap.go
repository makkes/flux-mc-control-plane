package bootstrap

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	scapi "github.com/fluxcd/source-controller/api/v1beta1"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/makkes/fluxmc/pkg/kubernetes"
)

var ErrNeedRepo = errors.New("a control-plane repository needs to be provided")
var ErrDestNoDir = errors.New("destination is not a directory")

//go:embed cp-manifests/*
var cpManifests embed.FS

type bootstrapCommand struct {
	repoURL string
	c       client.Client
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
			c, err := kubernetes.NewClient()
			if err != nil {
				return fmt.Errorf("could not create Kubernetes client: %w", err)
			}
			bcmd.c = c
			return bcmd.Run(context.Background())
		},
	}

	cmd.Flags().StringVar(&repoURL, "repository", "", "the repository URL to push control-plane manifests to")

	return cmd

}

func (b bootstrapCommand) Run(ctx context.Context) error {
	if err := b.pushControlPlaneManifests(ctx); err != nil {
		return fmt.Errorf("could not push control plane manifests: %w", err)
	}

	gitRepo := b.generateGitRepository(ctx)
	return b.c.Create(ctx, &gitRepo, &client.CreateOptions{})
}

func (b bootstrapCommand) generateGitRepository(ctx context.Context) scapi.GitRepository {
	return scapi.GitRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "control-plane",
			Namespace: "flux-system",
		},
		Spec: scapi.GitRepositorySpec{
			Interval: metav1.Duration{Duration: 1 * time.Minute},
			Reference: &scapi.GitRepositoryRef{
				Branch: "main",
			},
			URL: b.repoURL,
		},
	}
}

func (b bootstrapCommand) pushControlPlaneManifests(ctx context.Context) error {
	tmpGit, err := os.MkdirTemp("", "fluxmc-bootstrap.*")
	if err != nil {
		return fmt.Errorf("could not create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpGit)

	repo, err := git.PlainCloneContext(ctx, tmpGit, false, &git.CloneOptions{
		URL:           b.repoURL,
		ReferenceName: plumbing.NewBranchReferenceName("main"),
	})
	if err != nil {
		return fmt.Errorf("could not clone git repository: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("could not get work tree: %w", err)
	}

	if err := addContent(cpManifests, "cp-manifests", "control-plane", wt); err != nil {
		return fmt.Errorf("could not copy bootstrap files to local clone: %w", err)
	}

	if _, err := wt.Commit("add bootstrap manifests", &git.CommitOptions{
		Author: &object.Signature{
			Name: "Flux MC",
		},
	}); err != nil {
		return fmt.Errorf("could not commit bootstrap content: %w", err)
	}

	repo.Push(&git.PushOptions{})

	return nil
}

func addPath(wt *git.Worktree, path string) error {
	if _, err := wt.Add(path); err != nil {
		return fmt.Errorf("could not add %s: %w", path, err)
	}
	return nil
}

func addContent(sourceFS fs.FS, sourceDir string, destDir string, wt *git.Worktree) error {
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

		destPath := filepath.Join(wt.Filesystem.Root(), destDir, relPath)
		if err := ensureDirectory(filepath.Dir(destPath)); err != nil {
			return err
		}
		if err := os.WriteFile(destPath, content, 0o600); err != nil {
			return fmt.Errorf("could not write file %s: %w", destPath, err)
		}

		relDest, err := filepath.Rel(wt.Filesystem.Root(), destPath)
		if err != nil {
			return fmt.Errorf("failed to get repo-relative file path: %w", err)
		}
		if err := addPath(wt, relDest); err != nil {
			return err
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
