package checker

import (
	"context"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/runbits/commitmediator"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// RunCommitsBehindNotifier checks for the commits behind count based on the
// provided project and displays the results to the user in a helpful manner.
func RunCommitsBehindNotifier(p *project.Project, out output.Outputer) {
	count, err := CommitsBehind(p)
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

func CommitsBehind(p *project.Project) (int, error) {
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

	commitID, err := commitmediator.Get(p)
	if err != nil {
		return 0, errs.Wrap(err, "Unable to get local commit")
	}

	return model.CommitsBehind(*latestCommitID, commitID)
}

func RunUpdateNotifier(an analytics.Dispatcher, svc *model.SvcModel, out output.Outputer) {
	defer profile.Measure("RunUpdateNotifier", time.Now())

	ctx, cancel := context.WithTimeout(context.Background(), model.SvcTimeoutMinimal)
	defer cancel()

	upd, err := svc.CheckUpdate(ctx, constants.BranchName, "")
	if err != nil {
		var timeoutErr net.Error
		if errors.As(err, &timeoutErr) && timeoutErr.Timeout() {
			logging.Debug("CheckUpdate timed out")
			return
		}
		multilog.Error("Could not check for update when running update notifier, error: %v", errs.JoinMessage(err))
		return
	}

	update := updater.NewUpdateInstaller(an, updater.NewAvailableUpdateFromGraph(upd))
	if !update.ShouldInstall() {
		return
	}

	out.Notice(output.Title(locale.Tr("update_available_header")))
	out.Notice(locale.Tr("update_available", constants.Version, update.AvailableUpdate.Version))
}
