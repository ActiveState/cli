package main

import (
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	wc "github.com/ActiveState/cli/scripts/internal/workflow-controllers"
	wh "github.com/ActiveState/cli/scripts/internal/workflow-helpers"
	"github.com/andygrunwald/go-jira"
	"github.com/blang/semver"
	"github.com/google/go-github/v45/github"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

func main() {
	if err := run(); err != nil {
		wc.Print("Error: %s\n", errs.JoinMessage(err))
	}
}

type Meta struct {
	Version           semver.Version
	JiraVersion       string
	VersionPRName     string
	VersionBranchName string
	VersionPR         *github.PullRequest
}

func (m Meta) GetVersion() semver.Version {
	return m.Version
}

func (m Meta) GetJiraVersion() string {
	return m.JiraVersion
}

func (m Meta) GetVersionBranchName() string {
	return m.VersionBranchName
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
	repo, err := git.PlainOpen(environment.GetRootPathUnsafe())
	if err != nil {
		return errs.Wrap(err, "failed to open local repo")
	}
	finish()

	finish = wc.PrintStart("Checking if local repo is clean")
	// We can't use go-git here as it does not respect autcrlf
	stdout, stderr, err := exeutils.ExecSimpleFromDir(environment.GetRootPathUnsafe(), "git", []string{"status", "-s"}, nil)
	if err != nil {
		return errs.Wrap(err, "failed to check local repo status, stderr: %s", stderr)
	}
	stdout = strings.TrimSpace(stdout)
	if stdout != "" {
		return errs.New("Local repo is not clean, please make sure you have no pending changes. Status received:\n %s", stdout)
	}
	finish()

	finish = wc.PrintStart("Verifying input")
	// Grab input
	if len(os.Args) < 2 {
		return errs.New("Usage: start-story <story-id> [branch-name]")
	}
	jiraIssueID := os.Args[1]

	branchName := jiraIssueID
	if len(os.Args) > 2 {
		branchName = os.Args[1]
		detectedIssueID, err := wh.ParseJiraKey(branchName)
		if err != nil {
			return errs.Wrap(err, "failed to parse Jira key from branch name")
		}
		if strings.ToLower(detectedIssueID) != strings.ToLower(jiraIssueID) {
			return errs.New("Branch name contains story ID %s, but story being targeted is %s", detectedIssueID, jiraIssueID)
		}
	}
	finish()

	finish = wc.PrintStart("Fetching meta for story %s", jiraIssueID)
	// Collect meta information about the PR and all it's related resources
	meta, err := fetchMeta(ghClient, jiraClient, jiraIssueID)
	if err != nil {
		return errs.Wrap(err, "failed to fetch meta")
	}
	finish()

	ref := ""
	if meta.VersionPR == nil {
		ref = meta.VersionPR.Head.GetSHA()
	} else {
		finish := wc.PrintStart("Detecting base ref to fork from")
		ref, err = wc.DetectBaseRef(ghClient, jiraClient, meta)
		if err != nil {
			return errs.Wrap(err, "failed to detect base ref")
		}
		finish()
	}

	finish = wc.PrintStart("Creating branch")
	worktree, err := repo.Worktree()
	if err != nil {
		return errs.Wrap(err, "failed to get worktree")
	}
	if err := worktree.Checkout(&git.CheckoutOptions{Hash: plumbing.NewHash(ref)}); err != nil {
		return errs.Wrap(err, "failed to checkout base ref")
	}
	if err := repo.CreateBranch(&config.Branch{Name: branchName}); err != nil {
		return errs.Wrap(err, "failed to create branch")
	}

	finish()

	wc.Print("All Done")

	return nil
}

func fetchMeta(ghClient *github.Client, jiraClient *jira.Client, jiraIssueID string) (Meta, error) {
	// Retrieve Relevant Jira Issue for PR being handled
	finish := wc.PrintStart("Fetching Jira issue")
	jiraIssue, err := wh.FetchJiraIssue(jiraClient, jiraIssueID)
	if err != nil {
		return Meta{}, errs.Wrap(err, "failed to get Jira issue")
	}
	finish()

	finish = wc.PrintStart("Parsing version")
	version, jiraVersion, err := wh.ParseTargetFixVersion(jiraIssue, true)
	if err != nil {
		return Meta{}, errs.Wrap(err, "failed to parse version")
	}
	finish()

	versionPRName := wh.VersionedPRTitle(version)

	// Retrieve Relevant Fixversion Pr
	finish = wc.PrintStart("Checking if Version PR with title '%s' exists", versionPRName)
	versionPR, err := wh.FetchPRByTitle(ghClient, versionPRName)
	if err != nil {
		return Meta{}, errs.Wrap(err, "failed to get target PR")
	}
	wc.Print("Exists: %v", versionPR != nil)
	finish()

	return Meta{
		Version:           version,
		JiraVersion:       jiraVersion.Name,
		VersionPR:         versionPR,
		VersionPRName:     versionPRName,
		VersionBranchName: wh.VersionedBranchName(version),
	}, nil
}
