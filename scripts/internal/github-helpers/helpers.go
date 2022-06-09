package github_helpers

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/internal/testhelpers/secrethelper"
	"github.com/google/go-github/v45/github"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

var issueKeyRx = regexp.MustCompile(`(?i)(DX-\d+)`)

// ExtractJiraIssueID tries to extract the jira issue ID from either the PR title or the branch name
func ExtractJiraIssueID(pr *github.PullRequest) *string {
	if pr.Title == nil {
		panic(fmt.Sprintf("PR title is nil: %#v", pr))
	}
	if pr.Head == nil || pr.Head.Ref == nil {
		panic(fmt.Sprintf("Head or head ref is nil: %#v", pr))
	}

	// Extract from title
	matches := issueKeyRx.FindStringSubmatch(*pr.Title)
	if len(matches) == 2 {
		return p.StrP(strings.ToUpper(matches[1]))
	}

	// Extract from branch
	matches = issueKeyRx.FindStringSubmatch(*pr.Head.Ref)
	if len(matches) == 2 {
		return p.StrP(strings.ToUpper(matches[1]))
	}

	return nil
}

func InitClient() *github.Client {
	token := secrethelper.GetSecretIfEmpty(os.Getenv("GITHUB_TOKEN"), "user.GITHUB_TOKEN")

	// Init github client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

// FetchPRs fetches all PRs and iterates over all available pages
func FetchPRs(ghClient *github.Client, cutoff time.Time, opts *github.PullRequestListOptions) ([]*github.PullRequest, error) {
	page := 1
	result := []*github.PullRequest{}

	if opts == nil {
		opts = &github.PullRequestListOptions{
			State:     "closed",
			Base:      "master",
			Sort:      "updated",
			Direction: "desc",
		}
	}

	for x := 0; x < 10; x++ { // Hard limit of 1000 most recent PRs
		opts.ListOptions = github.ListOptions{
			Page:    page,
			PerPage: 100,
		}
		// Grab github PRs to compare against jira stories, cause Jira's API does not tell us what the linker PR is
		prs, _, err := ghClient.PullRequests.List(context.Background(), "ActiveState", "cli", opts)
		if err != nil {
			return nil, errs.Wrap(err, "Could not find PRs")
		}
		fmt.Printf("Processing %d PRs on page %d\n", len(prs), page)
		if len(prs) < 100 {
			break
		}
		if len(prs) > 0 && prs[0].UpdatedAt.Before(cutoff) {
			break // The rest of the PRs are too old to care about
		}
		result = append(result, prs...)
		page++
	}

	return result, nil
}

func FetchCommitsByShaRange(ghClient *github.Client, startSha string, stopSha string) ([]*github.RepositoryCommit, error) {
	result := []*github.RepositoryCommit{}
	page := 0
	perPage := 100

	for x := 0; x < 100; x++ { // hard limit of 100,000 commits
		commits, _, err := ghClient.Repositories.ListCommits(context.Background(), "ActiveState", "cli", &github.CommitsListOptions{
			SHA: startSha,
			ListOptions: github.ListOptions{
				Page:    0,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, errs.Wrap(err, "ListCommits failed")
		}

		for _, commit := range commits {
			if commit.GetSHA() == stopSha {
				return result, nil
			}
			result = append(result, commit)
		}

		if len(commits) < perPage {
			break // Last page
		}

		page++
	}

	return result, nil
}
