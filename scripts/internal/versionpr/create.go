package versionpr

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

type progressPrinterFunc func(string, ...interface{}) func()
type printerFunc func(string, ...interface{})

func Create(ghClient *github.Client, jiraClient *jira.Client, meta Meta, progressPrinter progressPrinterFunc, printer printerFunc) error {
	// Check if master is safe to fork from
	finish := progressPrinter("Checking if master is safe to fork from")
	var prevVersionRef *string
	versionsGT, err := wh.BranchHasVersionsGT(ghClient, jiraClient, wh.MasterBranch, meta.GetVersion())
	if err != nil {
		return errs.Wrap(err, "failed to check if can fork master")
	}

	// Calculate SHA for master
	if !versionsGT {
		printer("Master is safe")
		finish2 := progressPrinter("Getting master HEAD SHA")
		branch, _, err := ghClient.Repositories.GetBranch(context.Background(), "ActiveState", "cli", wh.MasterBranch, false)
		if err != nil {
			return errs.Wrap(err, "failed to get branch info")
		}
		prevVersionRef = branch.GetCommit().SHA
		printer("Master SHA: " + *prevVersionRef)
		finish2()
	} else {
		printer("Master is unsafe as it has versions greater than %s", meta.GetVersion())
	}
	finish()

	// Master is unsafe, detect closest matching PR instead
	if prevVersionRef == nil {
		finish = progressPrinter("Finding nearest matching version PR to fork from")
		prevVersionPR, err := wh.FetchVersionPR(ghClient, wh.AssertLT, meta.GetVersion())
		if err != nil {
			return errs.Wrap(err, "failed to find fork branch")
		}

		prevVersionRef = prevVersionPR.Head.SHA
		printer("Nearest matching PR: %s (%d), branch: %s, SHA: %s",
			prevVersionPR.GetTitle(), prevVersionPR.GetNumber(), prevVersionPR.Head.GetRef(), *prevVersionRef)
		finish()
	}

	// Create branch
	finish = progressPrinter("Creating version branch, name: %s, forked from: %s", meta.GetVersionBranchName(), *prevVersionRef)
	dryRun := os.Getenv("DRYRUN") == "true"
	if !dryRun {
		if err := wh.CreateBranch(ghClient, meta.GetVersionBranchName(), *prevVersionRef); err != nil {
			return errs.Wrap(err, "failed to create branch")
		}
	} else {
		printer("DRYRUN: skip")
	}
	finish()

	// Create commit with version.txt change
	finish = progressPrinter("Creating commit with version.txt change")
	parentSha, err := wh.CreateFileUpdateCommit(ghClient, meta.GetVersionBranchName(), "version.txt", meta.GetVersion().String())
	if err != nil {
		return errs.Wrap(err, "failed to create commit")
	}
	printer("Created commit SHA: %s", parentSha)
	finish()

	// Prepare PR Body
	body := fmt.Sprintf(`[View %s tickets on Jira](%s)`,
		meta.GetVersion(),
		"https://activestatef.atlassian.net/jira/software/c/projects/DX/issues/?jql="+
			url.QueryEscape(fmt.Sprintf(`project = "DX" AND fixVersion=%s ORDER BY created DESC`, meta.GetJiraVersion())),
	)

	// Create PR
	finish = progressPrinter("Creating version PR, name: %s", meta.GetVersionPRName())
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
