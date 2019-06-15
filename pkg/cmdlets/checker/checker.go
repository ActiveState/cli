package checker

import (
	"strconv"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// RunCommitsBehindNotifier checks for the commits behind count based on the
// provided project and displays the results to the user in a helpful manner.
func RunCommitsBehindNotifier(p *project.Project) {
	count, fail := model.CommitsBehindLatest(p.Owner(), p.Name(), p.CommitID())
	if fail != nil {
		if fail.Type.Matches(model.FailCommitCountUnknowable) {
			print.Info(locale.Tr("runtime_update_available_unknown_count", p.Owner(), p.Name()))
			return
		}

		logging.Error(locale.T("err_could_not_get_commit_behind_count"))
		return
	}
	if count > 0 {
		ct := strconv.Itoa(count)
		print.Info(locale.Tr("runtime_update_available", ct, p.Owner(), p.Name()))
	}
}
