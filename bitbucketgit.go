package mergeexp

import (
	"fmt"
	//"log"
	"regexp"
)

const BitBucketCloneBase = "git@bitbucket.org"

//const APIRoot    = "https://api.bitbucket.org/2.0/"
// repositories/gudang/gts-ocp/pullrequests?state=OPEN"

/**/
type BitBucketGit struct{ *MergeExp }

func (me *MergeExp) BitBucketGit() *BitBucketGit {
	return &BitBucketGit{MergeExp: me}
}

func (bb *BitBucketGit) CloneUrl(fullname string) string {
	base := bb.BitBucketCloneBase
	if base == "" {
		base = BitBucketCloneBase
	}
	return base + ":" + fullname + ".git"
}

func (bb *BitBucketGit) RemoteSuggestion(fullname string) string {
	return regexp.MustCompile(`[^-\w]+`).ReplaceAllString(fullname, "-")
}

func (bb *BitBucketGit) Fetch(remote string) error {
	c := bb.Command("git", "fetch", "--prune", remote)
	if bb.BitBucketDeploymentKey != "" {
		c.Env = []string{"GIT_SSH_COMMAND=ssh -i " + bb.BitBucketDeploymentKey}
	}
	return c.Run()
}

/* fetches branches from pull requests */
func (bb *BitBucketGit) FetchPRBranches(fullname string, destinationBranches []string, tags []string) ([]Branch, error) {
	prs, err := bb.MergeExp.BitBucketRest().SearchPullRequests(fullname, destinationBranches, tags)
	if err != nil {
		return nil, err
	}

	remoteFor := map[string]string{}
	branches := []Branch{}

	for _, pr := range prs {
		fullname := pr.SourceFullname
		if remoteFor[fullname] == "" {
			remote, err := bb.GetRemote(fullname)
			if err != nil {
				return nil, err
			}
			err = bb.Fetch(remote)
			if err != nil {
				return nil, err
			}
			remoteFor[fullname] = remote
		}
		// label should look like
		// drachonis/gts-ocp/AT-44903-GP-Hlasova-VPN-odstraneni-legacy-validace (pull request #2351)
		branches = append(
			branches,
			Branch{
				Name:  fmt.Sprintf("%s/%s", remoteFor[fullname], pr.SourceBranch),
				Label: fmt.Sprintf("%s/%s (pull request #%d)", fullname, pr.SourceBranch, pr.Id),
			})
	}

	return branches, nil
}

func (bb *BitBucketGit) GetRemote(fullname string) (string, error) {
	remotes := bb.MergeExp.GitRemotes()
	url := bb.CloneUrl(fullname)
	return remotes.CreateRemote(
		url,
		bb.RemoteSuggestion(fullname),
	)
}

func (bb *BitBucketGit) FetchBranch(fullname string, localname string) (*Branch, error) {
	remote, err := bb.GetRemote(fullname)
	if err != nil {
		return nil, err
	}

	err = bb.Fetch(remote)
	if err != nil {
		return nil, err
	}

	return &Branch{
		Name:      fmt.Sprintf("%s/%s", remote, localname),
		Label:     fmt.Sprintf("Bitbucket %s branch %s", fullname, localname),
		Remote:    remote,
		Localname: localname,
	}, nil
}
