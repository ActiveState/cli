package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/osutils"
	wc "github.com/ActiveState/cli/scripts/internal/workflow-controllers"
	wh "github.com/ActiveState/cli/scripts/internal/workflow-helpers"
	"github.com/andygrunwald/go-jira"
	"github.com/blang/semver"
	"github.com/google/go-github/v45/github"
	"golang.org/x/net/context"
)

type Meta struct {
	Repo          *github.Repository
	ActivePR      *github.PullRequest
	ActiveStory   *jira.Issue
	ActiveVersion semver.Version
}

type MergeIntends []MergeIntend

type MergeIntend struct {
	SourceBranch string
	TargetBranch string
}

func (m MergeIntend) String() string {
	return fmt.Sprintf("Merge %s into %s", m.SourceBranch, m.TargetBranch)
}

func (m MergeIntends) String() string {
	v := ""
	for _, vv := range m {
		v += fmt.Sprintf("\n%s", vv.String())
	}
	return v
}

func main() {
	if err := run(); err != nil {
		wc.Print("Error: %s\n", errs.JoinMessage(err))
		os.Exit(1)
	}
}

func run() error {
	finish := wc.PrintStart("Initializing clients")
	// Initialize Clients
	ghClient := wh.InitGHClient()
	jiraClient, err := wh.InitJiraClient()
	if err != nil {
		return errs.Wrap(err, "failed to initialize Jira client")
	}
	finish()

	// Grab input
	if len(os.Args) != 2 {
		return errs.New("Usage: propagate-pr <pr-number>")
	}
	prNumber, err := strconv.Atoi(os.Args[1])
	if err != nil {
		return errs.Wrap(err, "pr number should be numeric")
	}

	finish = wc.PrintStart("Fetching meta for PR %d", prNumber)
	// Collect meta information about the PR and all it's related resources
	meta, err := fetchMeta(ghClient, jiraClient, prNumber)
	if err != nil {
		return errs.Wrap(err, "failed to fetch meta")
	}
	finish()

	if meta.ActiveVersion.EQ(wh.VersionMaster) {
		wc.Print("Target version is master, no propagation required")
		return nil
	}

	// Find open version PRs
	finish = wc.PrintStart("Finding open version PRs that need to adopt this PR")
	versionPRs, err := wh.FetchVersionPRs(ghClient, wh.AssertGT, meta.ActiveVersion, -1)
	if err != nil {
		return errs.Wrap(err, "failed to fetch version PRs")
	}
	finish()

	// Parse merge intends
	intend := MergeIntends{}
	currentBranch := meta.ActivePR.GetBase().GetRef()
	targetBranches := []string{}
	for _, pr := range versionPRs {
		if pr.GetState() != "open" {
			return errs.Wrap(err, "Version PR %d is not open, does the source PR have the right fixVersion associated?", pr.GetNumber())
		}
		intend = append(intend, MergeIntend{
			SourceBranch: currentBranch,
			TargetBranch: pr.GetHead().GetRef(),
		})
		targetBranches = append(targetBranches, pr.GetHead().GetRef())
		currentBranch = pr.GetHead().GetRef()
	}

	// Always end with master
	intend = append(intend, MergeIntend{
		SourceBranch: currentBranch,
		TargetBranch: wh.MasterBranch,
	})
	targetBranches = append(targetBranches, wh.MasterBranch)

	wc.Print("Found %d branches that need to adopt this PR: %s\n", len(intend), strings.Join(targetBranches, ", "))

	// Iterate over the merge intends and merge them
	for i, v := range intend {
		finish = wc.PrintStart("Merging %s into %s", v.SourceBranch, v.TargetBranch)

		if os.Getenv("DRYRUN") == "true" {
			wc.Print("DRY RUN: Skipping merge")
			finish()
			continue
		}

		root := environment.GetRootPathUnsafe()
		stdout, stderr, err := osutils.ExecSimpleFromDir(root, "git", []string{"checkout", v.TargetBranch}, nil)
		if err != nil {
			return errs.Wrap(err, "failed to checkout %s, stdout:\n%s\nstderr:\n%s", v.TargetBranch, stdout, stderr)
		}

		stdout, stderr, err = osutils.ExecSimpleFromDir(root, "git", []string{
			"merge", v.SourceBranch,
			"--no-edit", "-m",
			fmt.Sprintf("Merge branch %s to adopt changes from PR #%d", v.SourceBranch, prNumber),
		}, nil)
		if err != nil {
			return errs.Wrap(err,
				"failed to merge %s into %s. please manually merge the following branches: %s"+
					"\nstdout:\n%s\nstderr:\n%s",
				v.SourceBranch, v.TargetBranch, intend[i:].String(), stdout, stderr)
		}

		stdout, stderr, err = osutils.ExecSimpleFromDir(root, "git", []string{"push"}, nil)
		if err != nil {
			return errs.Wrap(err,
				"failed to merge %s into %s. please manually merge the following branches: %s"+
					"\nstdout:\n%s\nstderr:\n%s",
				v.SourceBranch, v.TargetBranch, intend[i:].String(), stdout, stderr)
		}

		finish()
	}

	return nil
}

func fetchMeta(ghClient *github.Client, jiraClient *jira.Client, prNumber int) (Meta, error) {
	// Grab PR information about the PR that this automation is being ran on
	finish := wc.PrintStart("Fetching Active PR %d", prNumber)
	prBeingHandled, _, err := ghClient.PullRequests.Get(context.Background(), "ActiveState", "cli", prNumber)
	if err != nil {
		return Meta{}, errs.Wrap(err, "failed to get PR")
	}
	wc.Print("PR retrieved: %s", prBeingHandled.GetTitle())
	finish()

	if prBeingHandled.GetState() != "closed" && !prBeingHandled.GetMerged() {
		if os.Getenv("DRYRUN") != "true" {
			return Meta{}, errs.New("Active PR should be merged before it can be propagated.")
		}
	}

	finish = wc.PrintStart("Extracting Jira Issue ID from Active PR: %s", prBeingHandled.GetTitle())
	jiraIssueID, err := wh.ExtractJiraIssueID(prBeingHandled)
	if err != nil {
		return Meta{}, errs.Wrap(err, "PR does not have Jira issue ID associated with it: %s", prBeingHandled.Links.GetHTML().GetHRef())
	}
	wc.Print("Extracted Jira Issue ID: %s", jiraIssueID)
	finish()

	// Retrieve Relevant Jira Issue for PR being handled
	finish = wc.PrintStart("Fetching Jira issue")
	jiraIssue, err := wh.FetchJiraIssue(jiraClient, jiraIssueID)
	if err != nil {
		return Meta{}, errs.Wrap(err, "failed to get Jira issue")
	}
	finish()

	finish = wc.PrintStart("Fetching Jira Versions")
	availableVersions, err := wh.FetchAvailableVersions(jiraClient)
	if err != nil {
		return Meta{}, errs.Wrap(err, "Failed to fetch JIRA issue")
	}
	finish()

	// Retrieve Relevant Fixversion
	finish = wc.PrintStart("Extracting target fixVersion from Jira issue")
	fixVersion, _, err := wh.ParseTargetFixVersion(jiraIssue, availableVersions)
	if err != nil {
		return Meta{}, errs.Wrap(err, "failed to get fixVersion")
	}
	wc.Print("Extracted fixVersion: %s", fixVersion)
	finish()

	if err := wh.ValidVersionBranch(prBeingHandled.GetBase().GetRef(), fixVersion.Version); err != nil {
		return Meta{}, errs.Wrap(err, "Failed to validate that the target branch for the active PR is correct.")
	}

	result := Meta{
		Repo:          &github.Repository{},
		ActivePR:      prBeingHandled,
		ActiveStory:   jiraIssue,
		ActiveVersion: fixVersion.Version,
	}

	return result, nil
}
