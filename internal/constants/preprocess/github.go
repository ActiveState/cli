package preprocess

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/google/go-github/v29/github"
	"golang.org/x/oauth2"
)

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

func (g *githubClient) getVersionLabelMaster() (string, error) {
	pullRequests, err := g.getPullRequestList(&github.PullRequestListOptions{
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

		return label, nil
	}

	return "", errors.New("could not find version label from previously merged pull request")
}

func (g *githubClient) getVersionLabelPullRequest(number int) (string, error) {
	pullRequest, err := g.getPullRequest(number)
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

	return label, nil
}

func (g *githubClient) getPullRequestList(options *github.PullRequestListOptions) ([]*github.PullRequest, error) {
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

func (g *githubClient) getPullRequest(number int) (*github.PullRequest, error) {
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
