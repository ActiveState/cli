package wc

import (
	"fmt"
	"net/url"
	"os"

	"github.com/ActiveState/cli/internal/errs"
	wh "github.com/ActiveState/cli/scripts/internal/workflow-helpers"
	"github.com/andygrunwald/go-jira"
	"github.com/blang/semver"
	"github.com/google/go-github/v45/github"
	"golang.org/x/net/context"
)

type Meta interface {
	GetVersion() semver.Version
	GetJiraVersion() string
	GetVersionBranchName() string
	GetVersionPRName() string
}

func DetectBaseRef(ghClient *github.Client, jiraClient *jira.Client, meta Meta) (string, error) {
	// Check if master is safe to fork from
	finish := PrintStart("Checking if master is safe to fork from")
	var ref *string
	versionsGT, err := wh.BranchHasVersionsGT(ghClient, jiraClient, wh.MasterBranch, meta.GetVersion())
	if err != nil {
		return "", errs.Wrap(err, "failed to check if can fork master")
	}

	// Calculate SHA for master
	if !versionsGT {
		Print("Master is safe")
		finish2 := PrintStart("Getting master HEAD SHA")
		branch, _, err := ghClient.Repositories.GetBranch(context.Background(), "ActiveState", "cli", wh.MasterBranch, false)
		if err != nil {
			return "", errs.Wrap(err, "failed to get branch info")
		}
		ref = branch.GetCommit().SHA
		Print("Master SHA: " + *ref)
		finish2()
	} else {
		Print("Master is unsafe as it has versions greater than %s", meta.GetVersion())
	}
	finish()

	// Master is unsafe, detect closest matching PR instead
	if ref == nil {
		finish = PrintStart("Finding nearest matching version PR to fork from")
		prevVersionPR, err := wh.FetchVersionPR(ghClient, wh.AssertLT, meta.GetVersion())
		if err != nil {
			return "", errs.Wrap(err,
				"Failed to find fork branch, please manually create the Version PR "+
					"for '%s' by running the create-version-pr script.",
				meta.GetVersion())
		}

		ref = prevVersionPR.Head.SHA
		Print("Nearest matching PR: %s (%d), branch: %s, SHA: %s",
			prevVersionPR.GetTitle(), prevVersionPR.GetNumber(), prevVersionPR.Head.GetRef(), *ref)
		finish()
	}

	return *ref, nil
}

func CreateVersionPR(ghClient *github.Client, jiraClient *jira.Client, meta Meta) error {
	finish := PrintStart("Detecting base ref to fork from")
	prevVersionRef, err := DetectBaseRef(ghClient, jiraClient, meta)
	if err != nil {
		return errs.Wrap(err, "failed to detect base ref")
	}
	finish()

	// Create branch
	finish = PrintStart("Creating version branch, name: %s, forked from: %s", meta.GetVersionBranchName(), prevVersionRef)
	dryRun := os.Getenv("DRYRUN") == "true"
	if !dryRun {
		if err := wh.CreateBranch(ghClient, meta.GetVersionBranchName(), prevVersionRef); err != nil {
			return errs.Wrap(err, "failed to create branch")
		}
	} else {
		Print("DRYRUN: skip")
	}
	finish()

	// Create commit with version.txt change
	finish = PrintStart("Creating commit with version.txt change")
	parentSha, err := wh.CreateFileUpdateCommit(ghClient, meta.GetVersionBranchName(), "version.txt", meta.GetVersion().String())
	if err != nil {
		return errs.Wrap(err, "failed to create commit")
	}
	Print("Created commit SHA: %s", parentSha)
	finish()

	// Prepare PR Body
	body := fmt.Sprintf(`[View %s tickets on Jira](%s)`,
		meta.GetVersion(),
		"https://activestatef.atlassian.net/jira/software/c/projects/DX/issues/?jql="+
			url.QueryEscape(fmt.Sprintf(`project = "DX" AND fixVersion=%s ORDER BY created DESC`, meta.GetJiraVersion())),
	)

	// Create PR
	finish = PrintStart("Creating version PR, name: %s", meta.GetVersionPRName())
	if !dryRun {
		versionPR, err := wh.CreatePR(ghClient, meta.GetVersionPRName(), meta.GetVersionBranchName(), wh.StagingBranch, body)
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
