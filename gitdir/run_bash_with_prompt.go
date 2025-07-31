package gitdir

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func (wd *Dir) RunBashWithPrompt(prompt string) error {
	// escaping apostrophe
	prompt = strings.ReplaceAll(prompt, "'", "'\\''")

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
