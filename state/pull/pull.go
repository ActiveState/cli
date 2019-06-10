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

	cid, fail := defaultCommitID(project.Get())
	if fail != nil {
		failures.Handle(fail, locale.T("err_pull_get_commit_id"))
		return
	}

	updated, fail := updateCommitID(projectfile.Get(), cid)
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

func defaultCommitID(p *project.Project) (string, *failures.Failure) {
	proj, fail := model.FetchProjectByName(p.Owner(), p.Name())
	if fail != nil {
		return "", fail
	}

	branch, fail := model.DefaultBranchForProject(proj)
	if fail != nil {
		return "", fail
	}

	return branch.CommitID.String(), nil
}

func updateCommitID(p *projectfile.Project, cid string) (bool, *failures.Failure) {
	// add method to projectfile.Project type to regex replace value by key
	return false, nil
}
