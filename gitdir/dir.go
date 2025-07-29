package gitdir

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
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

func (wd *Dir) RunBashWithPrompt(prompt string) error {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "mergeex*.sh")
	if err != nil {
		return fmt.Errorf("cannot create temporary file: %w", err)
	}

	// Remember to clean up the file afterwards
	defer os.Remove(tmpFile.Name())

	// Example writing to the file
	text := []byte(fmt.Sprintf("PS1='%s'\n", prompt))
	if _, err = tmpFile.Write(text); err != nil {
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}

	// Close the file
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing tmp file: %w", err)
	}

	cmd := wd.Command("bash", "--init-file", tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (wd *Dir) StartBranch(branch, target string) error {
	if err := wd.Command("git", "rev-parse", "--verify", branch).Run(); err != nil {
		if err := wd.Command("git", "branch", "-f", branch, target).Run(); err != nil {
			return err
		}
	}
	if err := wd.Command("git", "checkout", branch).Run(); err != nil {
		return err
	}
	// dangerous
	if err := wd.Command("git", "reset", "--hard", target).Run(); err != nil {
		return err
	}
	return nil
}

func (wd *Dir) ShaExists(sha string) bool {
	cmd := wd.Command("git", "cat-file", "-t", sha)
	cmd.Stderr = nil
	return cmd.Run() == nil
}
