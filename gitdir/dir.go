package gitdir

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Dir represents different utilities for a git working tree
type Dir struct {
	Dir string
	Env []string
}

func New(dirRel string) (*Dir, error) {
	dir, err := filepath.Abs(dirRel)
	if err != nil {
		return nil, fmt.Errorf("cannot make '%s' absolute", dirRel)
	}
	fileInfo, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("invalid directory '%s': %w", dir, err)
	}
	// IsDir is short for fileInfo.Mode().IsDir()
	if !fileInfo.IsDir() {
		// file is a directory
		return nil, fmt.Errorf("file '%s' not a directory", dir)
	}
	return &Dir{Dir: dir}, nil
}

func (wd *Dir) Command(command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.Dir = wd.Dir
	cmd.Stderr = os.Stderr
	cmd.Env = wd.Env
	return cmd
}

func (wd *Dir) GitInit() error {
	err := wd.Command("git", "status").Run()
	if err != nil {
		err = wd.Command("git", "init").Run()
	}
	return err
}

// StartExperimentalBranch creates or recreates a Git branch, force-overwriting it if it already exists.
//
// This function performs the following actions:
//  1. Attempts to create a new branch named 'branch' from 'target'. If 'branch' already exists, it will be
//     forcefully overwritten (`git branch -f`).
//  2. Checks out the newly created or existing 'branch'.
//  3. **DANGER: Resets the working tree and the current branch to the 'target' commit, discarding any
//     local changes in the working directory and staging area on the 'branch'.**
//
// Parameters:
//
//	branch: The name of the branch to create or recreate.
//	target: The commit, branch, or tag from which to start the 'branch' (e.g., "main", "HEAD~1", "v1.0").
//
// Returns:
//
//	An error if any Git command fails, otherwise nil.
func (wd *Dir) StartExperimentalBranch(branch, target string) error {

	// checking the current branch first
	currentBranch, err := wd.currentBranch()
	if err != nil {
		return fmt.Errorf("check for current branch: %w", err)
	}

	if currentBranch == branch {
		// if on experimental branch, reset is enough, regardless current state
		// WARNING: This is a dangerous operation.
		// It resets the current branch (which is now 'branch') to the 'target' commit,
		// discarding any local changes in the working directory and staging area.
		if err := wd.Command("git", "reset", "--hard", target).Run(); err != nil {
			return fmt.Errorf("resetting to %s: %w", branch, err)
		}
		return nil
	}

	// checking if the working tree is not in a clean state
	out, err := wd.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return fmt.Errorf("checking the working tree state: %w", err)
	}

	if len(out) > 0 {
		// nonempty output
		return fmt.Errorf("working tree of %s is not in a clean state and on a branch %s different from %s, resolve it first",
			wd.Dir, currentBranch, branch)
	}

	// Forcefully create or recreate the branch from the target.
	// This command will overwrite 'branch' if it already exists.
	if err := wd.Command("git", "branch", "-f", branch, target).Run(); err != nil {
		return fmt.Errorf("creating branch %s: %w", branch, err)
	}

	// Checkout the specified branch.
	if err := wd.Command("git", "checkout", branch).Run(); err != nil {
		return fmt.Errorf("checking out %s: %w", branch, err)
	}

	return nil
}

// currentBranch returns the name of the current branch.
// It returns an empty string and an error if not on a branch (detached HEAD) or on error.
func (wd *Dir) currentBranch() (string, error) {
	cmd := wd.Command("git", "symbolic-ref", "--short", "HEAD")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Check if it's a "not a symbolic ref" error, which means detached HEAD
		if strings.Contains(stderr.String(), "not a symbolic ref") {
			return "", nil // Not on a branch
		}
		return "", err // Other error
	}

	return strings.TrimSpace(stdout.String()), nil
}

func (wd *Dir) ShaExists(sha string) bool {
	cmd := wd.Command("git", "cat-file", "-t", sha)
	cmd.Stderr = nil
	return cmd.Run() == nil
}
