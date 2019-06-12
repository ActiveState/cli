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

	proj := project.Get()
	latestID, fail := latestCommitID(proj.Owner(), proj.Name())
	if fail != nil {
		failures.Handle(fail, locale.T("err_pull_get_commit_id"))
		return
	}

	projFile := projectfile.Get()
	updated, fail := updateCommitID(projFile.SetCommit, "proj.CommitID()", latestID)
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

func latestCommitID(owner, project string) (string, *failures.Failure) {
	cid, fail := model.LatestCommitID(owner, project)
	if fail != nil {
		return "", fail
	}

	var id string
	if cid != nil {
		id = cid.String()
	}

	return id, nil
}

type setCommitFunc func(string) *failures.Failure

func updateCommitID(setCommit setCommitFunc, oldID, newID string) (bool, *failures.Failure) {
	if newID != "" && oldID != newID {
		return true, setCommit(newID)
	}

	return false, nil
}
