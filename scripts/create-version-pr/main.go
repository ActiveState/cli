package main

import (
	"os"

	"github.com/ActiveState/cli/internal/errs"
	wc "github.com/ActiveState/cli/scripts/internal/workflow-controllers"
	wh "github.com/ActiveState/cli/scripts/internal/workflow-helpers"
	"github.com/andygrunwald/go-jira"
	"github.com/blang/semver"
	"github.com/google/go-github/v45/github"
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
	finish()

	// Grab input
	if len(os.Args) != 2 {
		return errs.New("Usage: create-version-branch <version>")
	}
	versionName := os.Args[1]

	finish = wc.PrintStart("Fetching meta for version %s", versionName)
	// Collect meta information about the PR and all it's related resources
	meta, err := fetchMeta(ghClient, jiraClient, versionName)
	if err != nil {
		return errs.Wrap(err, "failed to fetch meta")
	}
	finish()

	finish = wc.PrintStart("Creating version PR for version %s", meta.Version)
	if err := wc.CreateVersionPR(ghClient, jiraClient, meta); err != nil {
		return errs.Wrap(err, "failed to create version PR")
	}
	finish()

	wc.Print("All Done")

	return nil
}

func fetchMeta(ghClient *github.Client, jiraClient *jira.Client, versionName string) (Meta, error) {
	version, err := wh.ParseJiraVersion(versionName)
	if err != nil {
		return Meta{}, errs.Wrap(err, "failed to parse version")
	}

	finish := wc.PrintStart("Fetching Jira Project info")
	project, _, err := jiraClient.Project.Get("DX")
	if err != nil {
		return Meta{}, errs.Wrap(err, "failed to fetch Jira project")
	}
	finish()

	versionPRName := wh.VersionedPRTitle(version)

	// Retrieve Relevant Fixversion Pr
	finish = wc.PrintStart("Checking if Version PR with title '%s' exists", versionPRName)
	versionPR, err := wh.FetchPRByTitle(ghClient, versionPRName)
	if err != nil {
		return Meta{}, errs.Wrap(err, "failed to get target PR")
	}
	if versionPR != nil {
		return Meta{}, errs.New("Version PR already exists: %s", versionPR.GetLinks().GetHTML().GetHRef())
	}
	finish()

	finish = wc.PrintStart("Fetching Jira version info")
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
