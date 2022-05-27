package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/andygrunwald/go-jira"
	"github.com/google/go-github/v45/github"
	"github.com/thoas/go-funk"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

// cutoff tells the script not to look at PRs before this date.
// You want to set this to the date when development on the given version started
var cutoff, _ = time.Parse("2006-Jan-02", "2022-Apr-25")

var issueKeyRx = regexp.MustCompile(`(?i)(DX-\d+)`)

func main() {
	if err := run(); err != nil {
		fmt.Printf("Failed: %s", errs.JoinMessage(err))
		os.Exit(1)
	}
	fmt.Println("Done")
}

func run() error {
	if len(os.Args) < 2 {
		return errs.New("Invalid arguments, please provide the version tag as the first argument")
	}
	targetVersion := os.Args[1]

	// Init jira client
	tp := &jira.BasicAuthTransport{
		Username: os.Getenv("JIRA_USERNAME"),
		Password: os.Getenv("JIRA_TOKEN"),
	}
	jiraClient, _ := jira.NewClient(tp.Client(), "https://activestatef.atlassian.net/")

	// Init github client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	ghClient := github.NewClient(tc)
	_ = ghClient

	// Grab target stories from jira
	issues, _, err := jiraClient.Issue.Search(fmt.Sprintf(`project = "DX" AND fixVersion=%s ORDER BY created DESC`, targetVersion), nil)
	if err != nil {
		return errs.Wrap(err, "Could not find issues")
	}

	// Create map of issue ID's by iterating over issues and extracting the ID property
	var jiraIssueIDs = make(map[string]struct{})
	for _, issue := range issues {
		jiraIssueIDs[strings.ToUpper(issue.Key)] = struct{}{}
	}

	fmt.Printf("Found %d issues: %v\n", len(issues), funk.Keys(jiraIssueIDs))

	// Grab github PRs to compare against jira stories, cause Jira's API does not tell us what the linker PR is
	prs, err := fetchPRs(ghClient)
	if err != nil {
		return errs.Wrap(err, "Could not find PRs")
	}

	fmt.Printf("Found %d PRs\n", len(prs))

	missingIDs := []string{}
	resultCommits := []string{}
	for _, pr := range prs {
		if pr.UpdatedAt.Before(cutoff) {
			continue // Before our adoption of jira
		}
		if pr.MergeCommitSHA == nil || *pr.MergeCommitSHA == "" || pr.Base.GetRef() != "master" {
			continue
		}

		commit := pr.GetMergeCommitSHA()[0:7]

		jiraIssueID := extractJiraIssueID(pr)
		if jiraIssueID == nil {
			missingIDs = append(missingIDs, fmt.Sprintf("%s (branch: %s, commit: %s): %s", *pr.Title, pr.Head.GetRef(), commit, pr.Links.GetHTML().GetHRef()))
			continue
		}

		_, ok := jiraIssueIDs[*jiraIssueID]
		if !ok {
			continue
		}

		fmt.Printf("Adding %s (branch: %s, commit: %s) as it matches Jira issue %s\n", *pr.Title, pr.Head.GetRef(), commit, *jiraIssueID)
		resultCommits = append(resultCommits, commit)
	}

	fmt.Printf("\nMissing Jira ID:\n%s\n\nCommits to merge: %s\n", strings.Join(missingIDs, "\n"), strings.Join(orderCommits(resultCommits), " "))

	return nil
}

// fetchPRs fetches all PRs and iterates over all available pages
func fetchPRs(ghClient *github.Client) ([]*github.PullRequest, error) {
	page := 1
	result := []*github.PullRequest{}
	for x := 0; x < 10; x++ { // Hard limit of 1000 most recent PRs
		// Grab github PRs to compare against jira stories, cause Jira's API does not tell us what the linker PR is
		prs, _, err := ghClient.PullRequests.List(context.Background(), "ActiveState", "cli", &github.PullRequestListOptions{
			State:     "closed",
			Base:      "master",
			Sort:      "updated",
			Direction: "desc",
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: 100,
			},
		})
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

// Order the commits by calling out to `git merge-base --is-ancestor commit1 commit2`
func orderCommits(hashes []string) []string {
	ordered := []string{}
	for _, hash := range hashes {
		handled := false
		for oidx, ohash := range ordered {
			code, _, err := exeutils.Execute("git", []string{"merge-base", "--is-ancestor", hash, ohash}, nil)
			if err != nil && !errs.Matches(err, &exec.ExitError{}) {
				panic(err)
			}
			if code == 0 {
				ordered = sliceutils.InsertStringAt(ordered, oidx, hash)
				handled = true
				break
			}
		}
		if !handled {
			ordered = append(ordered, hash)
		}
	}
	return ordered
}

// extractJiraIssueID tries to extract the jira issue ID from either the PR title or the branch name
func extractJiraIssueID(pr *github.PullRequest) *string {
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
