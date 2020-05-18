package preprocess

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/constants/version"
	"github.com/google/go-github/v29/github"
	"golang.org/x/oauth2"
)

const labelPrefix = "version: "
const branchPrefix = "ActiveState:"
const masterBranch = "master"

// GithubIncrementStateStore provides methods for getting label values from the Github API
type GithubIncrementStateStore struct {
	client *github.Client
}

// NewGithubIncrementStateStore returns an initialized Github client
func NewGithubIncrementStateStore(token string) *GithubIncrementStateStore {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	return &GithubIncrementStateStore{
		client: github.NewClient(tc),
	}
}

// IncrementType returns the increment value string (major, minor, patch) by
// reading the appropriate version file associated with the most recently
// merged pull request.
func (g *GithubIncrementStateStore) IncrementType() (string, error) {
	pullRequests, err := g.pullRequestList(&github.PullRequestListOptions{
		State:     "closed",
		Sort:      "updated",
		Direction: "desc",
	})
	if err != nil {
		return "", err
	}

	var branchName string
	for _, pullRequest := range pullRequests {
		merged, err := g.isMerged(pullRequest)
		if err != nil {
			return "", err
		}
		if !merged {
			continue
		}
		branchName = strings.TrimPrefix(pullRequest.Head.GetLabel(), branchPrefix)
		break
	}
	if branchName == "" {
		return "", errors.New("could not determine branch name from previosly merged pull requests")
	}

	return getVersionString(branchName)
}

func (g *GithubIncrementStateStore) versionLabelPullRequest(number int) (string, error) {
	pullRequest, err := g.pullRequest(number)
	if err != nil {
		return "", err
	}

	label := getLabel(pullRequest.Labels)
	target := strings.TrimPrefix(pullRequest.GetBase().GetLabel(), fmt.Sprintf("%s:", constants.LibraryOwner))
	if target != masterBranch && label == "" {
		return version.Patch, nil
	}

	if label == "" {
		return "", errors.New("no pull request label found")
	}

	return strings.TrimPrefix(label, labelPrefix), nil
}

func (g *GithubIncrementStateStore) pullRequestList(options *github.PullRequestListOptions) ([]*github.PullRequest, error) {
	pullReqests, _, err := g.client.PullRequests.List(
		context.Background(),
		constants.LibraryOwner,
		constants.LibraryName,
		options,
	)
	if err != nil {
		return nil, err
	}

	return pullReqests, nil
}

func (g *GithubIncrementStateStore) pullRequest(number int) (*github.PullRequest, error) {
	pullRequest, _, err := g.client.PullRequests.Get(context.Background(), constants.LibraryOwner, constants.LibraryName, number)
	if err != nil {
		return nil, err
	}

	return pullRequest, nil
}

func (g *GithubIncrementStateStore) isMerged(pullRequest *github.PullRequest) (bool, error) {
	if pullRequest.Number == nil {
		return false, errors.New("could not check if pull request has been merged, invalid pull request received")
	}
	merged, _, err := g.client.PullRequests.IsMerged(
		context.Background(),
		constants.LibraryOwner,
		constants.LibraryName,
		*pullRequest.Number,
	)
	if err != nil {
		return false, err
	}

	return merged, nil
}

func getLabel(labels []*github.Label) string {
	regex := regexp.MustCompile("version: (major|minor|patch)")

	for _, label := range labels {
		if label.Name != nil && regex.MatchString(*label.Name) {
			return *label.Name
		}
	}

	return ""
}
