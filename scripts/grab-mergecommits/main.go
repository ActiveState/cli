package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/scripts/internal/github-helpers"
	"github.com/ActiveState/cli/scripts/internal/jira-helpers"
	"github.com/thoas/go-funk"
)

// cutoff tells the script not to look at PRs before this date.
// You want to set this to the date when development on the given version started
var cutoff, _ = time.Parse("2006-Jan-02", "2022-Apr-25")

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

	jiraClient := jira_helpers.InitClient()
	ghClient := github_helpers.InitClient()

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
	prs, err := github_helpers.FetchPRs(ghClient, cutoff, nil)
	if err != nil {
		return errs.Wrap(err, "Could not find PRs")
	}

	fmt.Printf("Found %d PRs\n", len(prs))

	missingIDs := []string{}
	resultCommits := []string{}
	for _, pr := range prs {
		if pr.MergeCommitSHA == nil || *pr.MergeCommitSHA == "" || pr.Base.GetRef() != "master" {
			continue
		}

		commit := pr.GetMergeCommitSHA()[0:7]

		jiraIssueID := github_helpers.ExtractJiraIssueID(pr)
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
