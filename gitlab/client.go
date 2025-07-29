package gitlab

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	restClient *resty.Client
}

func NewClient(rc *resty.Client) *Client {
	return &Client{restClient: rc}
}

func (c *Client) Req(ctx context.Context) *resty.Request {
	return c.restClient.R().SetContext(ctx)
}

var ProjectNotFound = errors.New("gitlab project not found")

func (c *Client) MergeRequests(ctx context.Context, targetProjectId int, labels ...string) ([]MergeRequest, error) {
	var mrs []MergeRequest

	// building initial URL
	// Define query parameters using url.Values
	query := url.Values{}
	query.Add("state", "opened")
	// wip = work in progress, i.e. Drafts
	query.Add("wip", "no")
	if len(labels) > 0 {
		query.Add("labels", strings.Join(labels, ","))
	}
	u := (&url.URL{
		Path:     fmt.Sprintf("/projects/%d/merge_requests", targetProjectId),
		RawQuery: query.Encode(), // Encode the query parameters
	}).String()

	// paging
	for u != "" {
		var mrsPage []MergeRequest

		req := c.Req(ctx).SetResult(&mrsPage)
		resp, err := req.Get(u)
		if err != nil {
			return nil, fmt.Errorf("gitlab failed: %w", err)
		}
		if !resp.IsSuccess() {
			return nil, fmt.Errorf("fetch for GitLab merge requests failed with %s status", resp.Status())
		}

		mrs = append(mrs, mrsPage...)
		u = extractNextURL(resp.Header().Get("Link"))
	}

	return mrs, nil
}

// extractNextURL parses Link header in format RFC 8288: Web Linking and return URL for next link if any
// header may look like
// Link: <https://gitlab.example.com/api/v4/projects/9/issues/8/notes?per_page=3&page=1>; rel="prev", <https://gitlab.example.com/api/v4/projects/9/issues/8/notes?per_page=3&page=3>; rel="next", <https://gitlab.example.com/api/v4/projects/9/issues/8/notes?per_page=3&page=1>; rel="first", <https://gitlab.example.com/api/v4/projects/9/issues/8/notes?per_page=3&page=3>; rel="last"
func extractNextURL(link string) string {
	// Link header can contain multiple links separated by commas
	for _, linkPart := range strings.Split(link, ",") {
		// Each link part looks like: <url>; rel="relation"
		if strings.Contains(linkPart, `rel="next"`) {
			// Extract the URL part: it's between '<' and '>'
			urlStart := strings.Index(linkPart, "<")
			urlEnd := strings.Index(linkPart, ">")

			if urlStart > -1 && urlEnd > -1 && urlStart < urlEnd {
				return linkPart[urlStart+1 : urlEnd]
			}
		}
	}
	return ""
}

// returns SHA of branch name of project
func (c *Client) BranchSHA(ctx context.Context, projectID int, name string) (string, error) {
	var branches []struct {
		Commit struct {
			ID string `json:"id"`
		} `json:"commit"`
	}

	res, err := c.Req(ctx).
		SetQueryParam("regex", "^"+name+"$").
		SetResult(&branches).
		Get(fmt.Sprintf("projects/%d/repository/branches", projectID))
	if err != nil {
		return "", fmt.Errorf("gitlab call failed: %w", err)
	}
	if !res.IsSuccess() {
		if res.StatusCode() == http.StatusNotFound {
			return "", ProjectNotFound
		}
		return "", fmt.Errorf("gitlab call returned: %d", res.StatusCode())
	}

	if len(branches) == 0 {
		return "", nil
	} else if len(branches) > 1 {
		return "", fmt.Errorf("%d branches named '%s'", len(branches), name)
	}
	return branches[0].Commit.ID, nil
}

func (c *Client) ProjectSSHUrl(ctx context.Context, projectID int) (string, error) {
	var project struct {
		SshUrl string `json:"ssh_url_to_repo"`
	}
	res, err := c.Req(ctx).
		SetResult(&project).
		Get(fmt.Sprintf("projects/%d", projectID))
	if err != nil {
		return "", fmt.Errorf("gitlab call failed: %w", err)
	}
	if !res.IsSuccess() {
		if res.StatusCode() == http.StatusNotFound {
			return "", ProjectNotFound
		}
		return "", fmt.Errorf("gitlab call returned: %d", res.StatusCode())
	}

	return project.SshUrl, nil
}
