package runtime

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/go-openapi/strfmt"
)

func solveWithProgress(commitUUID strfmt.UUID, owner, project string, auth *authentication.Auth, out output.Outputer) (*bpModel.Commit, error) {
	out.Notice(locale.T("setup_runtime"))
	solveSpinner := output.StartSpinner(out, locale.T("progress_solve"), constants.TerminalAnimationInterval)

	bpm := bpModel.NewBuildPlannerModel(auth)
	commit, err := bpm.FetchCommit(commitUUID, owner, project, nil)
	if err != nil {
		solveSpinner.Stop(locale.T("progress_fail"))
		return nil, errs.Wrap(err, "Failed to fetch build result")
	}

	solveSpinner.Stop(locale.T("progress_success"))

	return commit, nil
}
