package checker

import (
	"errors"
	"strconv"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// RunCommitsBehindNotifier checks for the commits behind count based on the
// provided project and displays the results to the user in a helpful manner.
func RunCommitsBehindNotifier(out output.Outputer) {
	p, err := project.GetOnce()
	if err != nil {
		logging.Warning("Could not retrieve project, error: %v", err.Error())
		return
	}

	count, err := model.CommitsBehindLatest(p.Owner(), p.Name(), p.CommitID())
	if err != nil {
		if errors.Is(err, model.ErrCommitCountUnknowable) {
			out.Notice(output.Heading(locale.Tr("runtime_update_notice_unknown_count")))
			out.Notice(locale.Tr("runtime_update_help", p.Owner(), p.Name()))
			return
		}

		logging.Warning(locale.T("err_could_not_get_commit_behind_count"))
		return
	}
	if count > 0 {
		ct := strconv.Itoa(count)
		out.Notice(output.Heading(locale.Tr("runtime_update_notice_known_count", ct)))
		out.Notice(locale.Tr("runtime_update_help", p.Owner(), p.Name()))
	}
}
