package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils/p"
	wh "github.com/ActiveState/cli/scripts/internal/workflow-helpers"
	"github.com/andygrunwald/go-jira"
	"github.com/blang/semver"
	"github.com/google/go-github/v45/github"
	"golang.org/x/net/context"
)

type Meta struct {
	Repo              *github.Repository
	ActivePR          *github.PullRequest
	ActiveStory       *jira.Issue
	ActiveVersion     semver.Version
	ActiveJiraVersion string
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
		print("Error: %s\n", errs.JoinMessage(err))
	}
}

func run() error {
	finish := printStart("Initializing clients")
	// Initialize Clients
	ghClient := wh.InitGHClient()
	jiraClient, err := wh.InitJiraClient()
	if err != nil {
		return errs.Wrap(err, "failed to initialize Jira client")
	}
	finish()

	// Grab input
	if len(os.Args) != 2 {
		return errs.New("Usage: target-version-branch <pr-number>")
	}
	prNumber, err := strconv.Atoi(os.Args[1])
	if err != nil {
		return errs.Wrap(err, "pr number should be numeric")
	}

	finish = printStart("Fetching meta for PR %d", prNumber)
	// Collect meta information about the PR and all it's related resources
	meta, err := fetchMeta(ghClient, jiraClient, prNumber)
	if err != nil {
		return errs.Wrap(err, "failed to fetch meta")
	}
	finish()

	// Find open version PRs
	finish = printStart("Finding open version PRs that need to adopt this PR")
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

	print("Found %d branches that need to adopt this PR: %s\n", len(intend), strings.Join(targetBranches, ", "))

	// Iterate over the merge intends and merge them
	for i, v := range intend {
		finish = printStart("Merging %s into %s", v.SourceBranch, v.TargetBranch)

		if os.Getenv("DRYRUN") == "true" {
			print("DRY RUN: Skipping merge")
			finish()
			continue
		}

		// Perform the actual merge
		_, _, err := ghClient.Repositories.Merge(context.Background(), "ActiveState", "cli", &github.RepositoryMergeRequest{
			Base: &v.TargetBranch,
			Head: &v.SourceBranch,
			CommitMessage: p.StrP(fmt.Sprintf(
				"Merge branch %s into %s to adopt PR %d for story %s",
				v.SourceBranch, v.TargetBranch, meta.ActivePR.GetNumber(), meta.ActiveStory.Key,
			)),
		})
		if err != nil {
			return errs.Wrap(err, "Failed to merge PR, please manually merge the following branches: %s", intend[i:].String())
		}
		finish()
	}

	return nil
}

func fetchMeta(ghClient *github.Client, jiraClient *jira.Client, prNumber int) (Meta, error) {
	// Grab PR information about the PR that this automation is being ran on
	finish := printStart("Fetching Active PR %d", prNumber)
	prBeingHandled, _, err := ghClient.PullRequests.Get(context.Background(), "ActiveState", "cli", prNumber)
	if err != nil {
		return Meta{}, errs.Wrap(err, "failed to get PR")
	}
	print("PR retrieved: %s", prBeingHandled.GetTitle())
	finish()

	if prBeingHandled.GetState() != "closed" && !prBeingHandled.GetMerged() {
		return Meta{}, errs.New("Active PR should be merged before it can be propagated.")
	}

	if err := wh.ValidVersionBranch(prBeingHandled.GetBase().GetRef()); err != nil {
		return Meta{}, errs.Wrap(err, "Failed to validate that the target branch for the active PR is a valid version branch.")
	}

	finish = printStart("Extracting Jira Issue ID from Active PR: %s", prBeingHandled.GetTitle())
	jiraIssueID := wh.ExtractJiraIssueID(prBeingHandled)
	if jiraIssueID == nil {
		return Meta{}, errs.New("PR does not have Jira issue ID associated with it: %s", prBeingHandled.Links.GetHTML().GetHRef())
	}
	print("Extracted Jira Issue ID: %s", *jiraIssueID)
	finish()

	// Retrieve Relevant Jira Issue for PR being handled
	finish = printStart("Fetching Jira issue")
	jiraIssue, err := wh.FetchJiraIssue(jiraClient, *jiraIssueID)
	if err != nil {
		return Meta{}, errs.Wrap(err, "failed to get Jira issue")
	}
	finish()

	// Retrieve Relevant Fixversion
	finish = printStart("Extracting target fixVersion from Jira issue")
	fixVersion, jiraVersion, err := wh.ParseTargetFixVersion(jiraIssue, true)
	if err != nil {
		return Meta{}, errs.Wrap(err, "failed to get fixVersion")
	}
	print("Extracted fixVersion: %s", fixVersion)
	finish()

	result := Meta{
		Repo:              &github.Repository{},
		ActivePR:          prBeingHandled,
		ActiveStory:       jiraIssue,
		ActiveVersion:     fixVersion,
		ActiveJiraVersion: jiraVersion.Name,
	}

	return result, nil
}

var printDepth = 0

func print(msg string, args ...interface{}) {
	prefix := ""
	if printDepth > 0 {
		prefix = "|- "
	}
	indent := strings.Repeat("  ", printDepth)
	msg = strings.Replace(msg, "\n", indent+"\n", -1)
	fmt.Printf(indent + prefix + fmt.Sprintf(msg+"\n", args...))
}

func printStart(description string, args ...interface{}) func() {
	print(description+"..", args...)
	printDepth++
	return func() {
		printDepth--
		print("Done\n")
	}
}
