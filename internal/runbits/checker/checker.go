package checker

import (
	"errors"
	"strconv"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// RunCommitsBehindNotifier checks for the commits behind count based on the
// provided project and displays the results to the user in a helpful manner.
func RunCommitsBehindNotifier(p *project.Project, out output.Outputer, auth *authentication.Auth) {
	count, err := CommitsBehind(p, auth)
	if err != nil {
		if errors.Is(err, model.ErrCommitCountUnknowable) {
			out.Notice(output.Title(locale.Tr("runtime_update_notice_unknown_count")))
			out.Notice(locale.Tr("runtime_update_help", p.Owner(), p.Name()))
			return
		}

		logging.Warning(locale.T("err_could_not_get_commit_behind_count"))
		return
	}
	if count > 0 {
		ct := strconv.Itoa(count)
		out.Notice(output.Title(locale.Tr("runtime_update_notice_known_count", ct)))
		out.Notice(locale.Tr("runtime_update_help", p.Owner(), p.Name()))
	}
}

func CommitsBehind(p *project.Project, auth *authentication.Auth) (int, error) {
	if p.IsHeadless() {
		return 0, nil
	}

	latestCommitID, err := model.BranchCommitID(p.Owner(), p.Name(), p.BranchName())
	if err != nil {
		return 0, locale.WrapError(err, "Could not get branch information for {{.V0}}/{{.V1}}", p.Owner(), p.Name())
	}

	if latestCommitID == nil {
		return 0, locale.NewError("err_latest_commit", "Latest commit ID is nil")
	}

	commitID, err := localcommit.Get(p.Dir())
	if err != nil {
		return 0, errs.Wrap(err, "Unable to get local commit")
	}

	return model.CommitsBehind(*latestCommitID, commitID, auth)
}
