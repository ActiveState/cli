package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	github_helpers "github.com/ActiveState/cli/scripts/internal/github-helpers"
	jira_helpers "github.com/ActiveState/cli/scripts/internal/jira-helpers"
	"github.com/codemodus/relay"
	"github.com/google/go-github/v45/github"
	"golang.org/x/net/context"
)

// cutoff tells the script not to look at PRs before this date.
// We're assuming here that no release is under development for more than 3 months
// This saves us having to process all PRs since we started development
var cutoff = time.Now().Add(-(3 * 31 * 24 * time.Hour))

var jiraIssueRx = regexp.MustCompile(`(?i)(DX-\d+)`)

/*
- Pushes to fixVersion PR should verify that it has all the intended PRs for that version
- PRs should always have a fixVersion value
*/

var r = relay.New()

func main() {
	defer relay.Handle()

	// Validate Input
	{
		// Verify input args
		if len(os.Args) != 2 {
			r.Check(errs.New("Usage: verify-ci <id-of-current-pr>"))
			return
		}
	}

	ghClient := github_helpers.InitClient()

	prID, err := strconv.Atoi(os.Args[1])
	r.Check(err)
	pr, _, err := ghClient.PullRequests.Get(context.Background(), "ActiveState", "cli", prID)
	r.Check(err)

	isOnRCPR := strings.Contains(pr.GetTitle(), "-RC")
	if isOnRCPR {
		verifyRC(ghClient, pr)
	}
}

func verifyRC(ghClient *github.Client, pr *github.PullRequest) {
	version := strings.Split(pr.GetTitle(), " ")[0]
	jiraClient := jira_helpers.InitClient()

	issues, _, err := jiraClient.Issue.Search(fmt.Sprintf(
		// Why `statusCategory="In Progress" AND status!="In Progress"`?
		// Because we track "Ready for.." statuses as "In Progress", and it's easier to exclude status="In Progress"
		// than it is to include all the various "Ready for.." statuses.
		`project = "DX" AND fixVersion=%s AND statusCategory="In Progress" AND status!="In Progress" ORDER BY statusCategoryChangedDate ASC`,
		version), nil)
	r.Check(err)

	if len(issues) == 0 {
		r.Check(errs.New("No issues found for version %s", version))
	}

	jiraIDs := map[string]bool{}
	for _, issue := range issues {
		jiraIDs[strings.ToLower(issue.Key)] = false
	}

	commits, err := github_helpers.FetchCommitsByShaRange(ghClient, pr.GetHead().GetSHA(), pr.GetBase().GetSHA())
	r.Check(err)

	for _, commit := range commits {
		match := jiraIssueRx.FindStringSubmatch(commit.Commit.GetMessage())
		if len(match) != 2 {
			continue
		}

		if _, ok := jiraIDs[match[1]]; ok {
			jiraIDs[match[1]] = true
		}
	}

	notFound := []string{}
	for jiraID, isFound := range jiraIDs {
		if !isFound {
			notFound = append(notFound, jiraID)
		}
	}

	if len(notFound) > 0 {
		r.Check(errs.New("Missing JIRA issues: %s", strings.Join(notFound, ", ")))
	}
}
