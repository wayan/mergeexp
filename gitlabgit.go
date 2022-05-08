package mergeexp

import (
	"fmt"
	"log"
	"regexp"
)

/**/
type GitlabGit struct {*MergeExp }

func (me *MergeExp) GitlabGit() *GitlabGit {
	return &GitlabGit{MergeExp: me}
}

func (gg *GitlabGit) CloneUrl(fullname string) string {
	base := gg.GitlabCloneBase
	if base == "" {
        log.Fatal("Missing GitlabCloneBase");
	}
	return base + ":" + fullname + ".git"
}

func (gg *GitlabGit) RemoteSuggestion(fullname string) string {
	return regexp.MustCompile(`[^-\w]+`).ReplaceAllString(fullname, "-")
}

func (gg *GitlabGit) Fetch(remote string) error {
	c := gg.Command("git", "fetch", "--prune", remote)
	return c.Run()
}

func (gg *GitlabGit) GetRemote(fullname string) (string, error) {
	remotes := gg.MergeExp.GitRemotes()
	url := gg.CloneUrl(fullname)
	return remotes.CreateRemote(
		url,
		gg.RemoteSuggestion(fullname),
	)
}

func (gg *GitlabGit) FetchBranch(fullname string, localname string) (*Branch, error) {
	remote, err := gg.GetRemote(fullname)
	if err != nil {
		return nil, err
	}

	err = gg.Fetch(remote)
	if err != nil {
		return nil, err
	}

	return &Branch{
		Name:      fmt.Sprintf("%s/%s", remote, localname),
		Label:     fmt.Sprintf("Gitlab %s branch %s", fullname, localname),
		Remote:    remote,
		Localname: localname,
	}, nil
}
