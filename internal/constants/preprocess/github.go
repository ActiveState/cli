package preprocess

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/google/go-github/v29/github"
	"golang.org/x/oauth2"
)

const labelPrefix = "version: "
const masterBranch = "master"

// Client provides methods for getting label values from the Github API
type Client struct {
	client *github.Client
}

// NewGithubClient returns an initialized Github client
func NewGithubClient(token string) *Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	return &Client{
		client: github.NewClient(tc),
	}
}

// IncrementType returns the increment value string (major, minor, patch) of a
// pull request label for the current pull request or the most recently
// merged pull request
func (g *Client) IncrementType(branch string) (string, error) {
	if branch == masterBranch {
		return g.versionLabelMaster()
	}

	prNum, err := pullRequestNumber()
	if err != nil {
		return "", err
	}
	if prNum == 0 {
		return patch, nil
	}

	return g.versionLabelPullRequest(prNum)
}

func (g *Client) versionLabelMaster() (string, error) {
	pullRequests, err := g.pullRequestList(&github.PullRequestListOptions{
		State:     "closed",
		Sort:      "updated",
		Direction: "desc",
	})
	if err != nil {
		return "", err
	}

	for _, pullRequest := range pullRequests {
		merged, err := g.isMerged(pullRequest)
		if err != nil {
			return "", err
		}
		if !merged {
			continue
		}
		label := getLabel(pullRequest.Labels)
		if label == "" {
			return "", errors.New("no pull request label was found")
		}

		return strings.TrimPrefix(label, labelPrefix), nil
	}

	return "", errors.New("could not find version label from previously merged pull request")
}

func (g *Client) versionLabelPullRequest(number int) (string, error) {
	pullRequest, err := g.pullRequest(number)
	if err != nil {
		return "", err
	}

	label := getLabel(pullRequest.Labels)
	target := strings.TrimPrefix(pullRequest.GetBase().GetLabel(), fmt.Sprintf("%s:", constants.LibraryName))
	if target != masterBranch && label == "" {
		return patch, nil
	}

	if label == "" {
		return "", errors.New("no pull request label found")
	}

	return strings.TrimPrefix(label, labelPrefix), nil
}

func (g *Client) pullRequestList(options *github.PullRequestListOptions) ([]*github.PullRequest, error) {
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

func (g *Client) pullRequest(number int) (*github.PullRequest, error) {
	pullRequest, _, err := g.client.PullRequests.Get(context.Background(), constants.LibraryOwner, constants.LibraryName, number)
	if err != nil {
		return nil, err
	}

	return pullRequest, nil
}

func (g *Client) isMerged(pullRequest *github.PullRequest) (bool, error) {
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

func pullRequestNumber() (int, error) {
	// CircleCI
	prInfo := os.Getenv("CI_PULL_REQUEST")
	if prInfo != "" {
		return pullRequestNumberCircle(prInfo)
	}

	// Azure
	prInfo = os.Getenv("SYSTEM_PULLREQUEST_PULLREQUESTNUMBER")
	if prInfo != "" {
		return pullRequestNumberAzure(prInfo)
	}

	// Pull request info not set, we are on a branch but no PR has been created
	// Should still be allowed to build in this state hence we do not return
	// an error here.
	return 0, nil
}

func pullRequestNumberCircle(info string) (int, error) {
	regex := regexp.MustCompile("/pull/[0-9]+")
	match := regex.FindString(info)
	if match == "" {
		return 0, fmt.Errorf("could not determine pull request number from: %s", info)
	}
	num := strings.TrimPrefix(match, "/pull/")
	prNumber, err := strconv.Atoi(num)
	if err != nil {
		return 0, err
	}

	return prNumber, nil
}

func pullRequestNumberAzure(info string) (int, error) {
	regex := regexp.MustCompile("[0-9]+")
	if !regex.MatchString(info) {
		return 0, fmt.Errorf("pull request number contains more non-digits, recieved: %s", info)
	}

	prNumber, err := strconv.Atoi(info)
	if err != nil {
		return 0, err
	}

	return prNumber, nil
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
