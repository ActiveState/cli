package main

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/download"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/rtutils/p"
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
		err := createVersionPR(ghClient, jiraClient, meta)
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

func createVersionPR(ghClient *github.Client, jiraClient *jira.Client, meta Meta) error {
	// Check if master is safe to fork from
	finish := printStart("Checking if master is safe to fork from")
	var prevVersionRef *string
	versionsGT, err := wh.BranchHasVersionsGT(ghClient, jiraClient, wh.MasterBranch, meta.ActiveVersion)
	if err != nil {
		return errs.Wrap(err, "failed to check if can fork master")
	}

	// Calculate SHA for master
	if !versionsGT {
		print("Master is safe")
		finish2 := printStart("Getting master HEAD SHA")
		branch, _, err := ghClient.Repositories.GetBranch(context.Background(), "ActiveState", "cli", wh.MasterBranch, false)
		if err != nil {
			return errs.Wrap(err, "failed to get branch info")
		}
		prevVersionRef = branch.GetCommit().SHA
		print("Master SHA: " + *prevVersionRef)
		finish2()
	} else {
		print("Master is unsafe as it has versions greater than %s", meta.ActiveVersion)
	}
	finish()

	// Master is unsafe, detect closest matching PR instead
	if prevVersionRef == nil {
		finish = printStart("Finding nearest matching version branch to fork from")
		prevVersionPR, err := wh.FetchVersionPRLT(ghClient, meta.ActiveVersion)
		if err != nil {
			return errs.Wrap(err, "failed to find fork branch")
		}

		prevVersionRef = prevVersionPR.Head.SHA
		print("Nearest matching branch: %s, SHA: %s", prevVersionPR.Head.GetRef(), *prevVersionRef)
		finish()
	}

	// Create commit with version.txt change
	finish = printStart("Creating commit with version.txt change")
	parentSha, err := wh.CreateFileUpdateCommit(ghClient, *prevVersionRef, "version.txt", meta.ActiveVersion.String())
	if err != nil {
		return errs.Wrap(err, "failed to create commit")
	}
	print("Created commit SHA: %s", parentSha)
	finish()

	// Create branch
	finish = printStart("Creating version branch, name: %s, forked from: %s", meta.VersionBranchName, parentSha)
	dryRun := os.Getenv("DRYRUN") == "true"
	if !dryRun {
		if err := wh.CreateBranch(ghClient, meta.VersionBranchName, parentSha); err != nil {
			return errs.Wrap(err, "failed to create branch")
		}
	} else {
		print("DRYRUN: skip")
	}
	finish()

	// Prepare PR Body
	body := fmt.Sprintf(`[View %s tickets on Jira](%s)`,
		meta.ActiveVersion,
		"https://activestatef.atlassian.net/jira/software/c/projects/DX/issues/?jql="+
			url.QueryEscape(fmt.Sprintf(`project = "DX" AND fixVersion=%s ORDER BY created DESC`, meta.ActiveJiraVersion)),
	)

	// Create PR
	finish = printStart("Creating version PR, name: %s", meta.VersionPRName)
	if !dryRun {
		versionPR, err := wh.CreatePR(ghClient, meta.VersionPRName, meta.VersionBranchName, wh.StagingBranch, body)
		if err != nil {
			return errs.Wrap(err, "failed to create target PR")
		}

		if err := wh.LabelPR(ghClient, versionPR.GetNumber(), []string{"Test: all"}); err != nil {
			return errs.Wrap(err, "failed to label PR")
		}
	} else {
		fmt.Printf("DRYRUN: would create PR with body:\n%s\n", body)
	}
	finish()

	return nil
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

func execute(args ...string) error {
	print("Executing: %#v\n", args)
	c := exec.Command(args[0], args[1:]...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	if err != nil {
		return errs.Wrap(err, fmt.Sprintf("stdout: %s\nstderr: %s", stdout.String(), stderr.String()))
	}

	code := osutils.CmdExitCode(c)
	if code != 0 {
		return errs.New("%#v returned code %d", args, code)
	}

	return nil
}
