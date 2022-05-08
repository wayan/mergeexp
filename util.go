package mergeexp

import (
	"os/exec"
	"strings"
    "errors"
)

func SplitLines(out []byte) []string {
	/* must skip last, empty line */
	var lines []string
	for _, line := range strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func OutputString(cmd *exec.Cmd) (string, error) {
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func OutputLines(cmd *exec.Cmd) ([]string, error) {
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return SplitLines(out), nil
}

func OutputLine(cmd *exec.Cmd) (string, error) {
	lines, err := OutputLines(cmd)
	if err != nil {
		return "", err
	}
    if len(lines) == 0 {
		return "",  errors.New("No first line") 
    }
	return lines[0], nil
}
