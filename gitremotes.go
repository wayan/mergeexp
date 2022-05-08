package mergeexp

import (
	"fmt"
)

const GitRemotesPrefix = "macaque"

type stringmap = map[string]string

type GitRemotes struct{ *MergeExp }

func (me *MergeExp) GitRemotes() *GitRemotes {
	return &GitRemotes{MergeExp: me}
}

func (r *GitRemotes) Load() (stringmap, stringmap, error) {
	lines, err := OutputLines(r.Command("git", "remote"))
	if err != nil {
		return nil, nil, err
	}

	remoteUrl := make(stringmap)
	urlRemote := make(stringmap)
	for _, remote := range lines {
		url, err := OutputLine(r.Command("git", "remote", "get-url", remote))
		if err != nil {
			return nil, nil, err
		}

		remoteUrl[remote] = url
		urlRemote[url] = remote
	}
	return remoteUrl, urlRemote, nil
}

func (r *GitRemotes) CreateRemote(url string, suggestion string) (string, error) {
	remoteUrl, urlRemote, err := r.Load()
	if err != nil {
		return "", err
	}

	if urlRemote[url] != "" {
		/* already created */
		return urlRemote[url], nil
	}

	var create func(string, int) (string, error)
	create = func(prefix string, i int) (string, error) {
		remote := prefix
		if i > 0 {
			remote = fmt.Sprintf("%s-%d", prefix, i)
		}

		/* remote does not exist, we create it */
		if remoteUrl[remote] == "" {
			return r.add(remote, url)
		}

		/* trying next i */
		return create(prefix, i+1)
	}

	if suggestion == "" {
		return create(GitRemotesPrefix, 1)
	}
	return create(suggestion, 0)
}

func (r *GitRemotes) GetRemote(url string) (string, error) {
	_, urlRemote, err := r.Load()
	if err != nil {
		return "", err
	}
	return urlRemote[url], nil
}

func (r *GitRemotes) add(remote string, url string) (string, error) {
	err := r.Command("git", "remote", "add", remote, url).Run()
	if err != nil {
		return "", err
	}
	return remote, nil
}
