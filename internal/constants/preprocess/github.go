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

type githubClient struct {
	client *github.Client
}

func newGitHubService() *githubClient {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_REPO_TOKEN")},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	return &githubClient{
		client: github.NewClient(tc),
	}
}

func (g *githubClient) incrementValue(branchName string) (string, error) {
	if branchName == "master" {
		return g.versionLabelMaster()
	}

	prNum, err := pullRequestNumber()
	if err != nil {
		return "", err
	}
	return g.versionLabelPullRequest(prNum)
}

func (g *githubClient) versionLabelMaster() (string, error) {
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

func (g *githubClient) versionLabelPullRequest(number int) (string, error) {
	pullRequest, err := g.pullRequest(number)
	if err != nil {
		return "", err
	}

	label := getLabel(pullRequest.Labels)
	target := strings.TrimPrefix(pullRequest.GetBase().GetLabel(), fmt.Sprintf("%s:", constants.LibraryName))
	if target != "master" && label == "" {
		return patch, nil
	}

	if label == "" {
		return "", errors.New("no pull request label found")
	}

	return strings.TrimPrefix(label, labelPrefix), nil
}

func (g *githubClient) pullRequestList(options *github.PullRequestListOptions) ([]*github.PullRequest, error) {
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

func (g *githubClient) pullRequest(number int) (*github.PullRequest, error) {
	pullRequest, _, err := g.client.PullRequests.Get(context.Background(), constants.LibraryOwner, constants.LibraryName, number)
	if err != nil {
		return nil, err
	}

	return pullRequest, nil
}

func (g *githubClient) isMerged(pullRequest *github.PullRequest) (bool, error) {
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
