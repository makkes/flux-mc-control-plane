package gogit_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/require"

	"github.com/makkes/fluxmc/common/gitops/gogit"
)

func startGitServer() (func(), string, error) {
	ctrName := "gitserver"
	ctrPort := "8765"
	rmCmd := exec.Command("docker", "rm", "-f", ctrName)
	rmCmd.Stdout = os.Stdout
	rmCmd.Stderr = os.Stderr
	if err := rmCmd.Run(); err != nil {
		return func() {}, "", fmt.Errorf("could not remove dangling container: %w", err)
	}

	srvCmd := exec.Command("docker", "run", "-d", "--name", ctrName, "-p", ctrPort+":80", "makkes/gitserver:v0.0.6")
	srvCmd.Stdout = os.Stdout
	srvCmd.Stderr = os.Stderr
	if err := srvCmd.Run(); err != nil {
		return func() {}, "", fmt.Errorf("could not run container: %w", err)
	}

	cleanupFn := func() {
		for _, subCmd := range []string{"stop", "rm"} {
			cmd := exec.Command("docker", subCmd, ctrName)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				panic(fmt.Errorf("could not %s container: %s", subCmd, err))
			}
		}
	}

	ticker := time.Tick(100 * time.Millisecond)
	timeout := time.After(3 * time.Second)
	for {
		select {
		case <-timeout:
			return func() {}, "", fmt.Errorf("timeout waiting for container to get ready")
		default:
			<-ticker
			resp, err := http.Get("http://localhost:8765")
			if err == nil {
				resp.Body.Close()
				// treat as ready for any status code in [200, 500)
				if resp.StatusCode/100 < 5 && resp.StatusCode/100 >= 2 {
					return cleanupFn, ctrPort, nil
				}
			}
		}
	}
}

func TestCloneOnCancelledContext(t *testing.T) {
	srcDir := t.TempDir()
	targetDir := t.TempDir()
	require.NoError(t, copy.Copy("testdata/repo.git", srcDir), "could not copy repo")
	repo, err := gogit.NewGoGitRepo(srcDir, gogit.Depth(1))
	require.NoError(t, err, "could not create git repo")
	require.NotNil(t, repo, "repo is supposed to be non-nil")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = repo.Clone(ctx, targetDir)
	require.Error(t, err, "expected Clone to fail")
	require.Regexp(t, "context canceled$", err.Error(), "unexpected error")
}

func TestPushOnCancelledContext(t *testing.T) {
	srcDir := t.TempDir()
	targetDir := t.TempDir()
	require.NoError(t, copy.Copy("testdata/repo.git", srcDir), "could not copy repo")
	repo, err := gogit.NewGoGitRepo(srcDir, gogit.Depth(1))
	require.NoError(t, err, "could not create git repo")
	require.NotNil(t, repo, "repo is supposed to be non-nil")

	ctx, cancel := context.WithCancel(context.Background())
	err = repo.Clone(ctx, targetDir)
	require.NoError(t, err, "could not clone git repo")

	cancel()

	err = repo.Push(ctx)
	require.Error(t, err, "expected Clone to fail")
	require.Regexp(t, "^context canceled$", err.Error(), "unexpected error")
}

func TestCommitAndPush(t *testing.T) {
	srcDir := t.TempDir()
	targetDir := t.TempDir()
	require.NoError(t, copy.Copy("testdata/repo.git", srcDir), "could not copy repo")
	repo, err := gogit.NewGoGitRepo(srcDir, gogit.Depth(1))
	require.NoError(t, err, "could not create git repo")
	require.NotNil(t, repo, "repo is supposed to be non-nil")

	err = repo.Clone(context.Background(), targetDir)
	require.NoError(t, err, "could not clone git repo")

	fname := fmt.Sprintf("test-%d", time.Now().UnixNano())
	require.NoError(t, ioutil.WriteFile(path.Join(targetDir, fname), []byte("the file"), 0600), "could not write file")

	err = repo.Add(fname)
	require.NoError(t, err, "could not add file to repo")

	err = repo.Commit("adding a file", "max", "max@example.org")
	require.NoError(t, err, "could not create commit")

	err = repo.Push(context.Background())
	require.NoError(t, err, "could not push to remote")
}

func TestAddingNonExistentFile(t *testing.T) {
	srcDir := t.TempDir()
	targetDir := t.TempDir()
	require.NoError(t, copy.Copy("testdata/repo.git", srcDir), "could not copy repo")
	repo, err := gogit.NewGoGitRepo(srcDir, gogit.Depth(1))
	require.NoError(t, err, "could not create git repo")
	require.NotNil(t, repo, "repo is supposed to be non-nil")

	err = repo.Clone(context.Background(), targetDir)
	require.NoError(t, err, "could not clone git repo")

	fname := fmt.Sprintf("test-%d", time.Now().UnixNano())
	err = repo.Add(fname)
	require.Error(t, err, "expected a non-nil error")
}

func TestAddingFileByAbsolutePath(t *testing.T) {
	srcDir := t.TempDir()
	targetDir := t.TempDir()
	require.NoError(t, copy.Copy("testdata/repo.git", srcDir), "could not copy repo")
	repo, err := gogit.NewGoGitRepo(srcDir, gogit.Depth(1))
	require.NoError(t, err, "could not create git repo")
	require.NotNil(t, repo, "repo is supposed to be non-nil")

	err = repo.Clone(context.Background(), targetDir)
	require.NoError(t, err, "could not clone git repo")

	fname := fmt.Sprintf("test-%d", time.Now().UnixNano())
	err = repo.Add(path.Join(targetDir, fname))
	require.Error(t, err, "expected a non-nil error")
}

func TestDeletingAFile(t *testing.T) {
	srcDir := t.TempDir()
	targetDir := t.TempDir()
	require.NoError(t, copy.Copy("testdata/repo.git", srcDir), "could not copy repo")
	repo, err := gogit.NewGoGitRepo(srcDir, gogit.Depth(1))
	require.NoError(t, err, "could not create git repo")
	require.NotNil(t, repo, "repo is supposed to be non-nil")

	err = repo.Clone(context.Background(), targetDir)
	require.NoError(t, err, "could not clone git repo")

	err = os.Remove(path.Join(targetDir, "README.md"))
	require.NoError(t, err, "could not remove file")

	err = repo.Add("README.md")
	require.NoError(t, err, "could not add file to index")

	err = repo.Commit("removing README.md", "max", "max@example.org")
	require.NoError(t, err, "could not create commit")

	err = repo.Push(context.Background())
	require.NoError(t, err, "could not push to remote")
}

func TestDeletingADirectory(t *testing.T) {
	srcDir, _ := ioutil.TempDir("", "gitops-test-src-")
	targetDir, _ := ioutil.TempDir("", "gitops-target-dir-")
	require.NoError(t, copy.Copy("testdata/repo.git", srcDir), "could not copy repo")
	repo, err := gogit.NewGoGitRepo(srcDir, gogit.Depth(1))
	require.NoError(t, err, "could not create git repo")
	require.NotNil(t, repo, "repo is supposed to be non-nil")

	err = repo.Clone(context.Background(), targetDir)
	require.NoError(t, err, "could not clone git repo")

	err = repo.Remove("folder1")
	require.NoError(t, err, "could not remove folder")

	err = repo.Commit("removing README.md", "max", "max@example.org")
	require.NoError(t, err, "could not create commit")

	err = repo.Push(context.Background())
	require.NoError(t, err, "could not push to remote")
}

func TestBasicAuth(t *testing.T) {
	cleanupFn, port, err := startGitServer()
	t.Cleanup(cleanupFn)
	require.NoError(t, err, "could not start git server")
	targetDir := t.TempDir()
	repo, err := gogit.NewGoGitRepo("http://localhost:"+port+"/repo.git", gogit.Depth(1), gogit.BasicAuth("git", "git123"))
	require.NoError(t, err, "could not create git repo")
	require.NotNil(t, repo, "repo is supposed to be non-nil")

	err = repo.Clone(context.Background(), targetDir)
	require.NoError(t, err, "could not clone git repo")
}
