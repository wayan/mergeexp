package mergeexp

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
)

const BitBucketApiRoot = "https://api.bitbucket.org/2.0/"

// repositories/gudang/gts-ocp/pullrequests?state=OPEN"

type BitBucketRest struct{ *MergeExp }

type PullRequest struct {
	Id             int
	SourceBranch   string
	SourceFullname string
}

func (me *MergeExp) BitBucketRest() *BitBucketRest {
	// for BitBucketRest to work, these parameters must be present
	if me.BitBucketUsername == "" {
		log.Fatal("Missing BitBucket user")
	}
	if me.BitBucketPassword == "" {
		log.Fatal("Missing BitBucket app password")
	}

	return &BitBucketRest{MergeExp: me}
}

func (bb *BitBucketRest) PullRequestsUrl(fullname string) string {
	root := bb.BitBucketApiRoot
	if root == "" {
		root = BitBucketApiRoot
	}
	return root + "repositories/" + fullname + "/pullrequests?state=OPEN"
}

func (bb *BitBucketRest) Fetch(url string, entity interface{}) error {
	req, _ := http.NewRequest("GET", url, nil)

	bb.Logger.Info("Fetching " + url)

	if bb.BitBucketUsername == "" {
		log.Fatal("Missing BitBucket user")
	}
	if bb.BitBucketPassword == "" {
		log.Fatal("Missing BitBucket app password")
	}

	req.SetBasicAuth(bb.BitBucketUsername, bb.BitBucketPassword)
	resp, err := bb.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("BitBucket Fail: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("BitBucket returned unexpected code %d %s", resp.StatusCode, string(b))
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	err = json.Unmarshal(b, entity)
	if err != nil {
		return fmt.Errorf("Parsing Fail: %w", err)
	}

	return nil
}

type restPullRequest struct {
	Id    int `json:"id"`
	Links struct {
		Comments struct {
			Href string `json:"href"`
		} `json:"comments"`
	} `json:"links"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"destination"`
	Source struct {
		Repository struct {
			Links struct {
				Self struct {
					Href string `json:"href"`
				} `json:"self"`
			} `json:"links"`
			FullName string `json:"full_name"`
		} `json:"repository"`
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"source"`
}

func (bb *BitBucketRest) SearchPullRequests(fullname string, destinationBranches []string, tags []string) ([]*PullRequest, error) {
	/* recursive function */
	var fetch func(string, []*PullRequest) ([]*PullRequest, error)

	fetch = func(url string, pullRequests []*PullRequest) ([]*PullRequest, error) {
		var prs struct {
			Values []restPullRequest `json:"values"`
			Next   string            `json:"next"`
		}

		err := bb.Fetch(url, &prs)
		if err != nil {
			return nil, fmt.Errorf("Fetching pull requests failed: %w", err)
		}

		for _, rpr := range prs.Values {
			ok, err := bb.testPullRequest(rpr, destinationBranches, tags)
			if err != nil {
				return nil, err
			}
			if ok {
				fullname := rpr.Source.Repository.FullName
				pr := PullRequest{
					Id:             rpr.Id,
					SourceBranch:   rpr.Source.Branch.Name,
					SourceFullname: fullname,
				}
				pullRequests = append(pullRequests, &pr)
			}
		}
		if prs.Next != "" {
			return fetch(prs.Next, pullRequests)
		}
		return pullRequests, nil
	}
	return fetch(bb.PullRequestsUrl(fullname), make([]*PullRequest, 0))
}

func (bb *BitBucketRest) testPullRequest(rpr restPullRequest, destinationBranches []string, tags []string) (bool, error) {
	testBranch := func(branch string) bool {
		for _, db := range destinationBranches {
			if branch == db {
				return true
			}
		}
		return false
	}

	if !testBranch(rpr.Destination.Branch.Name) {
		return false, nil
	}

	return bb.testDeploymentTags(rpr.Links.Comments.Href, tags)
}

func (bb *BitBucketRest) testDeploymentTags(commentsUrl string, tags []string) (bool, error) {
	var testc func(string, bool) (bool, error)

	/* the last comment is the one which is valid */
	testc = func(url string, ok bool) (bool, error) {
		var comments struct {
			Values []struct {
				Content struct {
					Raw string `json:"raw"`
				} `json:"content"`
			} `json:"values"`
			Next string `json:"next"`
		}

		err := bb.Fetch(url, &comments)
		if err != nil {
			return false, fmt.Errorf("Fetching comments failed: %w", err)
		}

		for _, v := range comments.Values {
			if decided, deployed := TestComment(v.Content.Raw, tags); decided {
				ok = deployed
			}
		}

		if comments.Next != "" {
			return testc(comments.Next, ok)
		}

		return ok, nil
	}

	return testc(commentsUrl, false)
}

func TestComment(comment string, tags []string) (bool, bool) {
	re := regexp.MustCompile("\\bdeployment:\\s*(?:(no)-)?(\\S+)")
	for _, match := range re.FindAllStringSubmatch(comment, -1) {
		tag := match[2]
		for _, t := range tags {
			if t == tag {
				return true, match[1] != "no"

			}
		}
	}
	return false, false
}
