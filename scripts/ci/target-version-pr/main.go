package main

import (
	"os"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	wc "github.com/ActiveState/cli/scripts/internal/workflow-controllers"
	wh "github.com/ActiveState/cli/scripts/internal/workflow-helpers"
	"github.com/andygrunwald/go-jira"
	"github.com/blang/semver"
	"github.com/google/go-github/v45/github"
	"golang.org/x/net/context"
)

func main() {
	if err := run(); err != nil {
		wc.Print("Error: %s\n", errs.JoinMessage(err))
		os.Exit(1)
	}
}

type Meta struct {
	Repo              *github.Repository
	ActivePR          *github.PullRequest
	ActiveStory       *jira.Issue
	ActiveVersion     wh.Version
	ActiveJiraVersion string
	VersionPRName     string
	TargetBranchName  string
	VersionPR         *github.PullRequest
	IsVersionPR       bool
}

func (m Meta) GetVersion() semver.Version {
	return m.ActiveVersion.Version
}

func (m Meta) GetJiraVersion() string {
	return m.ActiveJiraVersion
}

func (m Meta) GetVersionBranchName() string {
	return m.TargetBranchName
}

func (m Meta) GetVersionPRName() string {
	return m.VersionPRName
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
		return errs.New("Usage: target-version-branch <pr-number>")
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

	// Create version PR if it doesn't exist yet
	if !meta.IsVersionPR && meta.VersionPR == nil && !meta.ActiveVersion.EQ(wh.VersionMaster) {
		finish = wc.PrintStart("Creating version PR for fixVersion %s", meta.ActiveVersion)
		err := wc.CreateVersionPR(ghClient, jiraClient, meta)
		if err != nil {
			return errs.Wrap(err, "failed to create version PR")
		}
		finish()
	}

	// Set the target branch for our PR
	finish = wc.PrintStart("Setting target branch to %s", meta.TargetBranchName)
	if strings.HasSuffix(meta.ActivePR.GetBase().GetRef(), meta.TargetBranchName) {
		wc.Print("PR already targets version branch %s", meta.TargetBranchName)
	} else {
		if os.Getenv("DRYRUN") != "true" {
			if err := wh.UpdatePRTargetBranch(ghClient, meta.ActivePR.GetNumber(), meta.TargetBranchName); err != nil {
				return errs.Wrap(err, "failed to update PR target branch")
			}
		} else {
			wc.Print("DRYRUN: would update PR target branch to %s", meta.TargetBranchName)
		}
	}
	finish()

	// Set the fixVersion
	if !meta.IsVersionPR {
		finish = wc.PrintStart("Setting fixVersion to %s", meta.ActiveVersion)
		if len(meta.ActiveStory.Fields.FixVersions) == 0 || meta.ActiveStory.Fields.FixVersions[0].ID != meta.ActiveVersion.JiraID {
			if os.Getenv("DRYRUN") != "true" {
				if err := wh.UpdateJiraFixVersion(jiraClient, meta.ActiveStory, meta.ActiveVersion.JiraID); err != nil {
					return errs.Wrap(err, "failed to update Jira fixVersion")
				}
			} else {
				wc.Print("DRYRUN: would set fixVersion to %s", meta.ActiveVersion.String())
			}
		} else {
			wc.Print("Jira issue already has fixVersion %s", meta.ActiveVersion.String())
		}
		finish()
	}

	wc.Print("All Done")

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

	if wh.IsVersionBranch(prBeingHandled.Head.GetRef()) {
		return Meta{
			Repo:             &github.Repository{},
			ActivePR:         prBeingHandled,
			TargetBranchName: "beta",
			IsVersionPR:      true,
		}, nil
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
	fixVersion, jiraVersion, err := wh.ParseTargetFixVersion(jiraIssue, availableVersions)
	if err != nil {
		return Meta{}, errs.Wrap(err, "failed to get fixVersion")
	}
	wc.Print("Extracted fixVersion: %s", fixVersion)
	finish()

	var versionPRName string
	var versionPR *github.PullRequest
	if fixVersion.NE(wh.VersionMaster) {
		versionPRName := wh.VersionedPRTitle(fixVersion.Version)

		// Retrieve Relevant Fixversion Pr
		finish = wc.PrintStart("Fetching Version PR by name: '%s'", versionPRName)
		versionPR, err = wh.FetchPRByTitle(ghClient, versionPRName)
		if err != nil {
			return Meta{}, errs.Wrap(err, "failed to get target PR")
		}
		if versionPR != nil && versionPR.GetState() != "open" {
			return Meta{}, errs.New("PR status for %s is not open, make sure your jira fixVersion is targeting an unreleased version", versionPR.GetTitle())
		}
		if versionPR == nil {
			wc.Print("No version PR found")
		}
		finish()
	}

	result := Meta{
		Repo:              &github.Repository{},
		ActivePR:          prBeingHandled,
		ActiveStory:       jiraIssue,
		ActiveVersion:     fixVersion,
		ActiveJiraVersion: jiraVersion.Name,
		VersionPRName:     versionPRName,
		TargetBranchName:  wh.VersionedBranchName(fixVersion.Version),
		VersionPR:         versionPR,
	}

	return result, nil
}
