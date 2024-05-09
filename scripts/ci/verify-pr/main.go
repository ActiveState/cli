package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/httputil"
	wc "github.com/ActiveState/cli/scripts/internal/workflow-controllers"
	wh "github.com/ActiveState/cli/scripts/internal/workflow-helpers"
	"github.com/andygrunwald/go-jira"
	"github.com/blang/semver"
	"github.com/google/go-github/v45/github"
	"golang.org/x/net/context"
)

/*
- Pushes to fixVersion PR should verify that it has all the intended PRs for that version
- PRs should always have a fixVersion value
*/

func main() {
	if err := run(); err != nil {
		wc.Print("Error: %s\n", errs.JoinMessage(err))
		os.Exit(1)
	}
}

func run() error {
	// Validate Input
	{
		// Verify input args
		if len(os.Args) != 2 {
			return errs.New("Usage: verify-pr <pr-number>")
		}
	}

	prID, err := strconv.Atoi(os.Args[1])
	if err != nil {
		return errs.Wrap(err, "PR number should be numeric")
	}

	finish := wc.PrintStart("Initializing clients")
	ghClient := wh.InitGHClient()
	jiraClient, err := wh.InitJiraClient()
	if err != nil {
		return errs.Wrap(err, "Failed to initialize JIRA client")
	}
	finish()

	finish = wc.PrintStart("Fetching PR")
	pr, _, err := ghClient.PullRequests.Get(context.Background(), "ActiveState", "cli", prID)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch PR")
	}
	finish()

	if wh.IsVersionBranch(pr.GetHead().GetRef()) {
		finish = wc.PrintStart("Verifying Version PR")
		if err := verifyVersionRC(ghClient, jiraClient, pr); err != nil {
			return errs.Wrap(err, "Failed to Version PR")
		}
		finish()
	}
	finish = wc.PrintStart("Verifying PR")
	if err := verifyPR(jiraClient, pr); err != nil {
		return errs.Wrap(err, "Failed to verify PR")
	}
	finish()

	return nil
}

func verifyVersionRC(ghClient *github.Client, jiraClient *jira.Client, pr *github.PullRequest) error {
	if pr.GetBase().GetRef() != wh.StagingBranch {
		return errs.New("PR should be targeting the staging branch: '%s'", wh.StagingBranch)
	}

	finish := wc.PrintStart("Parsing version from PR title")
	version, err := wh.VersionFromPRTitle(pr.GetTitle())
	if err != nil {
		return errs.Wrap(err, "Failed to parse version from PR title")
	}
	wc.Print("Version: %s", version)
	finish()

	finish = wc.PrintStart("Fetching Jira issues targeting %s", version)
	issues, _, err := jiraClient.Issue.Search(fmt.Sprintf(
		`project = "DX" AND fixVersion=v%s ORDER BY statusCategoryChangedDate ASC`,
		version), nil)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch JIRA issues, does the version 'v%s' exist on Jira?", version)
	}

	found := map[string]bool{}
	jiraIDs := map[string]jira.Issue{}
	for _, issue := range issues {
		if issue.Fields == nil || issue.Fields.Status == nil {
			return errs.New("Jira fields and/or status properties are nil, this should never happen..")
		}
		jiraIDs[strings.ToUpper(issue.Key)] = issue
		found[strings.ToUpper(issue.Key)] = false
	}
	finish()

	finish = wc.PrintStart("Fetching previous version PR")
	prevVersionPR, err := wh.FetchVersionPR(ghClient, wh.AssertLT, version)
	if err != nil {
		return errs.Wrap(err,
			"Failed to find previous version PR for %s.", version.String())
	}
	wc.Print("Got: %s\n", prevVersionPR.GetTitle())
	finish()

	finish = wc.PrintStart("Verifying we have all the commits from the previous version PR, comparing %s to %s", pr.Head.GetRef(), prevVersionPR.Head.GetRef())
	behind, err := wh.GetCommitsBehind(ghClient, prevVersionPR.Head.GetRef(), pr.Head.GetRef())
	if err != nil {
		return errs.Wrap(err, "Failed to compare to previous version PR")
	}
	if len(behind) > 0 {
		commits := []string{}
		for _, c := range behind {
			commits = append(commits, c.GetSHA()+": "+c.GetCommit().GetMessage())
		}
		return errs.New("PR is behind the previous version PR (%s) by %d commits, missing commits:\n%s",
			prevVersionPR.GetTitle(), len(behind), strings.Join(commits, "\n"))
	}
	finish()

	finish = wc.PrintStart("Fetching commits for PR %d", pr.GetNumber())
	commits, err := wh.FetchCommitsByRef(ghClient, pr.GetHead().GetSHA(), func(commit *github.RepositoryCommit) bool {
		return commit.GetSHA() == prevVersionPR.GetHead().GetSHA()
	})
	if err != nil {
		return errs.Wrap(err, "Failed to fetch commits")
	}
	finish()

	finish = wc.PrintStart("Matching commits against jira issues")
	for _, commit := range commits {
		key, err := wh.ParseJiraKey(commit.GetCommit().GetMessage())
		if err != nil {
			continue
		}
		key = strings.ToUpper(key) // ParseJiraKey already does this, but it's implicit

		if _, ok := jiraIDs[key]; ok {
			found[key] = true
		}
	}

	notFound := []string{}
	notFoundCritical := []string{}
	for jiraID, isFound := range found {
		if !isFound {
			issue := jiraIDs[jiraID]
			if wh.IsMergedStatus(issue.Fields.Status.Name) {
				notFoundCritical = append(notFoundCritical, issue.Key+": "+jiraIDs[jiraID].Fields.Summary)
			} else {
				notFound = append(notFound, issue.Key+": "+jiraIDs[jiraID].Fields.Summary)
			}
		}
	}

	sort.Strings(notFound)
	sort.Strings(notFoundCritical)

	if len(notFound) > 0 {
		return errs.New("PR not ready as it's still missing commits for the following JIRA issues:\n"+
			"Pending story completion:\n%s\n\n"+
			"Missing stories:\n%s", strings.Join(notFound, "\n"), strings.Join(notFoundCritical, "\n"))
	}
	finish()

	return nil
}

func verifyPR(jiraClient *jira.Client, pr *github.PullRequest) error {
	finish := wc.PrintStart("Parsing Jira issue from PR title")
	jiraIssueID, err := wh.ExtractJiraIssueID(pr)
	if err != nil {
		return errs.Wrap(err, "Failed to extract JIRA issue ID from PR")
	}
	wc.Print("JIRA Issue: %s\n", jiraIssueID)
	finish()

	finish = wc.PrintStart("Fetching Jira issue %s", jiraIssueID)
	jiraIssue, err := wh.FetchJiraIssue(jiraClient, jiraIssueID)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch JIRA issue")
	}
	finish()

	finish = wc.PrintStart("Fetching Jira Versions")
	availableVersions, err := wh.FetchAvailableVersions(jiraClient)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch JIRA issue")
	}
	finish()

	// Grab latest version on release channel to use as cutoff
	finish = wc.PrintStart("Fetching latest version on release channel")
	latestReleaseversionBytes, err := httputil.Get("https://raw.githubusercontent.com/ActiveState/cli/release/version.txt")
	if err != nil {
		return errs.Wrap(err, "failed to fetch latest release version")
	}
	latestReleaseversion, err := semver.Parse(strings.TrimSpace(string(latestReleaseversionBytes)))
	if err != nil {
		return errs.Wrap(err, "failed to parse version blob")
	}
	wc.Print("Latest version on release channel: %s", latestReleaseversion)
	finish()

	finish = wc.PrintStart("Verifying fixVersion")
	version, _, err := wh.ParseTargetFixVersion(jiraIssue, availableVersions)
	if err != nil {
		return errs.Wrap(err, "Failed to parse fixVersion")
	}
	finish()

	if !version.EQ(wh.VersionMaster) {
		// Ensure we have a valid version
		if version.LTE(latestReleaseversion) {
			return errs.New("Target fixVersion is either is less than the latest release version")
		}
	}

	finish = wc.PrintStart("Validating target branch")
	if err := wh.ValidVersionBranch(pr.GetBase().GetRef(), version.Version); err != nil {
		return errs.Wrap(err, "Invalid target branch, ensure your PR is targeting a versioned branch")
	}
	finish()

	return nil
}
