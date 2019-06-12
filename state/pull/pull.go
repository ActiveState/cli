package pull

import (
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// Command is the pull command's definition.
var Command = &commands.Command{
	Name:        "pull",
	Description: "pull_latest",
	Run:         Execute,
}

// Execute the pull command.
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")

	latestID, fail := latestCommitID(project.Get())
	if fail != nil {
		failures.Handle(fail, locale.T("err_pull_get_commit_id"))
		return
	}

	updated, fail := updateCommitID(projectfile.Get(), latestID)
	if fail != nil {
		failures.Handle(fail, locale.T("err_pull_update_commit_id"))
		return
	}

	locKey := "pull_is_updated"
	if !updated {
		locKey = "pull_not_updated"
	}
	print.Line(locale.T(locKey))
}

func latestCommitID(p *project.Project) (string, *failures.Failure) {
	proj, fail := model.FetchProjectByName(p.Owner(), p.Name())
	if fail != nil {
		return "", fail
	}

	branch, fail := model.DefaultBranchForProject(proj)
	if fail != nil {
		return "", fail
	}

	var cid string
	if branch.CommitID != nil {
		cid = branch.CommitID.String()
	}

	return cid, nil
}

func updateCommitID(p *projectfile.Project, newID string) (bool, *failures.Failure) {
	//break // halt on build
	oldID := ""

	if oldID == "" || oldID != newID {
		return true, p.SetCommit(newID)
	}

	return false, nil
}
