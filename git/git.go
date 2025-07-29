package git

import (
	"fmt"
	"iter"
	"slices"
	"strings"
	"unicode"

	"github.com/wayan/mergeexp/gitdir"
)

// chaotic mixture of git related utilities

func LsRemote(gd *gitdir.Dir, args ...string) (string, error) {
	args = append([]string{"ls-remote"}, args...)
	cmd := gd.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("fetching remote failed: %w", err)
	}
	s := string(out)
	idxWhite := strings.IndexFunc(s, unicode.IsSpace)
	if idxWhite < 0 {
		return "", fmt.Errorf("unexpected output from ls-remote: %s", s)
	}

	return s[:idxWhite], nil
}

func parseOutputTable(output []byte) iter.Seq[[]string] {
	s := string(output)

	return func(yield func([]string) bool) {
		for s != "" {
			line := s
			rest := ""
			if idx := strings.Index(s, "\n"); idx >= 0 {
				line = s[:idx]
				rest = s[idx+1:]
			}
			if !yield(strings.Fields(line)) {
				return
			}
			s = rest
		}
	}
}

func HighestVersionTag(gd *gitdir.Dir, url string) (*VersionTag, error) {
	cmd := gd.Command("git", "ls-remote", "--tags", "--sort=v:refname", url)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("fetching remote failed: %w", err)
	}

	// parsing the output
	var tags []VersionTag
	for row := range parseOutputTable(out) {
		if len(row) < 2 {
			continue
		}
		if vt := parseVersionTag(row[1]); vt != nil {
			vt.SHA = row[0]
			tags = append(tags, *vt)
		}
		// 2.5.0^{}
	}
	if len(tags) == 0 {
		return nil, nil
	}
	slices.SortFunc(tags, compareVersionTags)
	return &(tags[len(tags)-1]), nil
}
