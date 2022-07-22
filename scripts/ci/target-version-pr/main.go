package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/download"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/scripts/internal/versionpr"
	wh "github.com/ActiveState/cli/scripts/internal/workflow-helpers"
	"github.com/andygrunwald/go-jira"
	"github.com/blang/semver"
	"github.com/google/go-github/v45/github"
	"golang.org/x/net/context"
)

func main() {
	if err := run(); err != nil {
		print("Error: %s\n", errs.JoinMessage(err))
	}
}

type Meta struct {
	Repo              *github.Repository
	ActivePR          *github.PullRequest
	ActiveStory       *jira.Issue
	ActiveVersion     semver.Version
	ActiveJiraVersion string
	VersionPRName     string
	VersionBranchName string
	VersionPR         *github.PullRequest
}

func (m Meta) GetVersion() semver.Version {
	return m.ActiveVersion
}

func (m Meta) GetJiraVersion() string {
	return m.ActiveJiraVersion
}

func (m Meta) GetVersionBranchName() string {
	return m.VersionBranchName
}

func (m Meta) GetVersionPRName() string {
	return m.VersionPRName
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

	// Create version PR if it doesn't exist yet
	if meta.VersionPR == nil {
		finish = printStart("Creating version PR for fixVersion %s", meta.ActiveVersion)
		err := versionpr.Create(ghClient, jiraClient, meta, printStart, print)
		if err != nil {
			return errs.Wrap(err, "failed to create version PR")
		}
		finish()
	}

	// Set the target branch for our PR
	finish = printStart("Setting target branch to %s", meta.VersionBranchName)
	if os.Getenv("DRYRUN") != "true" {
		if err := wh.UpdatePRTargetBranch(ghClient, meta.ActivePR.GetNumber(), meta.VersionBranchName); err != nil {
			return errs.Wrap(err, "failed to update PR target branch")
		}
	} else {
		print("DRYRUN: would update PR target branch to %s", meta.VersionBranchName)
	}
	finish()

	print("All Done")

	return nil
}

func fetchMeta(ghClient *github.Client, jiraClient *jira.Client, prNumber int) (Meta, error) {
	// Grab latest version on release channel to use as cutoff
	finish := printStart("Fetching latest version on release channel")
	latestReleaseversionBytes, err := download.Get("https://raw.githubusercontent.com/ActiveState/cli/release/version.txt")
	latestReleaseversion, err := semver.Parse(strings.TrimSpace(string(latestReleaseversionBytes)))
	if err != nil {
		return Meta{}, errs.Wrap(err, "failed to parse version blob")
	}
	print("Latest version on release channel: %s", latestReleaseversion)
	finish()

	// Grab PR information about the PR that this automation is being ran on
	finish = printStart("Fetching Active PR %d", prNumber)
	prBeingHandled, _, err := ghClient.PullRequests.Get(context.Background(), "ActiveState", "cli", prNumber)
	if err != nil {
		return Meta{}, errs.Wrap(err, "failed to get PR")
	}
	print("PR retrieved: %s", prBeingHandled.GetTitle())
	finish()

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

	// Ensure we have a valid fixVersion
	if fixVersion.LTE(latestReleaseversion) || p.PBool(jiraVersion.Archived) || p.PBool(jiraVersion.Released) {
		return Meta{}, errs.New("Target fixVersion is either archived, released or is less than the latest release version")
	}
	print("Extracted fixVersion: %s", fixVersion)
	finish()

	versionPRName := wh.VersionedPRTitle(fixVersion)

	// Retrieve Relevant Fixversion Pr
	finish = printStart("Fetching Version PR")
	versionPR, err := wh.FetchPRByTitle(ghClient, versionPRName)
	if err != nil {
		return Meta{}, errs.Wrap(err, "failed to get target PR")
	}
	if versionPR != nil && versionPR.GetState() != "open" {
		return Meta{}, errs.New("PR status for %s is not open, make sure your jira fixVersion is targeting an unreleased version", versionPR.GetTitle())
	}
	finish()

	result := Meta{
		Repo:              &github.Repository{},
		ActivePR:          prBeingHandled,
		ActiveStory:       jiraIssue,
		ActiveVersion:     fixVersion,
		ActiveJiraVersion: jiraVersion.Name,
		VersionPRName:     versionPRName,
		VersionBranchName: wh.VersionedBranchName(fixVersion),
		VersionPR:         versionPR,
	}

	return result, nil
}

var printDepth = 0

func print(msg string, args ...interface{}) {
	prefix := ""
	if printDepth > 0 {
		prefix = "|- "
	}
	fmt.Printf(strings.Repeat("  ", printDepth) + prefix + fmt.Sprintf(msg+"\n", args...))
}

func printStart(description string, args ...interface{}) func() {
	print(description+"..", args...)
	printDepth++
	return func() {
		printDepth--
		print("Done\n")
	}
}
