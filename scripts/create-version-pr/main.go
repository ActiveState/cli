package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/scripts/internal/versionpr"
	wh "github.com/ActiveState/cli/scripts/internal/workflow-helpers"
	"github.com/andygrunwald/go-jira"
	"github.com/blang/semver"
	"github.com/google/go-github/v45/github"
)

func main() {
	if err := run(); err != nil {
		print("Error: %s\n", errs.JoinMessage(err))
	}
}

type Meta struct {
	Version           semver.Version
	JiraVersion       string
	VersionPRName     string
	VersionBranchName string
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
		return errs.New("Usage: create-version-branch <version>")
	}
	versionName := os.Args[1]

	finish = printStart("Fetching meta for version %s", versionName)
	// Collect meta information about the PR and all it's related resources
	meta, err := fetchMeta(ghClient, jiraClient, versionName)
	if err != nil {
		return errs.Wrap(err, "failed to fetch meta")
	}
	finish()

	finish = printStart("Creating version PR for version %s", meta.Version)
	if err := versionpr.Create(ghClient, jiraClient, meta, printStart, print); err != nil {
		return errs.Wrap(err, "failed to create version PR")
	}
	finish()

	print("All Done")

	return nil
}

func fetchMeta(ghClient *github.Client, jiraClient *jira.Client, versionName string) (Meta, error) {
	version, err := wh.ParseJiraVersion(versionName)
	if err != nil {
		return Meta{}, errs.Wrap(err, "failed to parse version")
	}

	finish := printStart("Fetching Jira Project info")
	project, _, err := jiraClient.Project.Get("DX")
	if err != nil {
		return Meta{}, errs.Wrap(err, "failed to fetch Jira project")
	}
	finish()

	versionPRName := wh.VersionedPRTitle(version)

	// Retrieve Relevant Fixversion Pr
	finish = printStart("Checking if Version PR with title '%s' exists", versionPRName)
	versionPR, err := wh.FetchPRByTitle(ghClient, versionPRName)
	if err != nil {
		return Meta{}, errs.Wrap(err, "failed to get target PR")
	}
	if versionPR != nil {
		return Meta{}, errs.New("Version PR already exists: %s", versionPR.GetLinks().GetHTML())
	}
	finish()

	finish = printStart("Fetching Jira version info")
	for _, v := range project.Versions {
		if v.Name == versionName {
			finish()
			return Meta{
				Version:           version,
				JiraVersion:       v.Name,
				VersionPRName:     versionPRName,
				VersionBranchName: wh.VersionedBranchName(version),
			}, nil
		}
	}

	return Meta{}, errs.New("failed to find Jira version matching: %s", versionName)
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
