package uninstall

import (
	"os"
	"runtime"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/instanceid"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
)

type Params struct {
	Path      string
	UserScope bool
}

type Uninstall struct {
	output    output.Outputer
	subshell  subshell.SubShell
	cfg       *config.Instance
	analytics analytics.Dispatcher
}

type primeable interface {
	primer.Outputer
	primer.Subsheller
	primer.Configurer
	primer.Analyticer
}

func NewDeployUninstall(prime primeable) *Uninstall {
	return &Uninstall{prime.Output(), prime.Subshell(), prime.Config(), prime.Analytics()}
}

func (u *Uninstall) Run(params *Params) error {
	if runtime.GOOS == "windows" && !params.UserScope {
		isAdmin, err := osutils.IsAdmin()
		if err != nil {
			multilog.Error("Could not check for windows administrator privileges: %v", err)
		}
		if !isAdmin {
			return locale.NewError(
				"err_deploy_uninstall_admin_privileges_required",
				"Administrator rights are required for this command to modify the system PATH.  If you want to uninstall from the user environment, please adjust the command line flags.")
		}
	}

	path := params.Path
	var cwd string
	if path == "" {
		var err error
		cwd, err = osutils.Getwd()
		if err != nil {
			return locale.WrapExternalError(
				err,
				"err_deploy_uninstall_cannot_get_cwd",
				"Cannot determine current working directory. Please supply '[ACTIONABLE]--path[/RESET]' argument")
		}
		path = cwd
	}

	logging.Debug("Attempting to uninstall deployment at %s", path)
	store := store.New(path)
	if !store.HasMarker() {
		return errs.AddTips(
			locale.NewInputError("err_deploy_uninstall_not_deployed", "There is no deployed runtime at '{{.V0}}' to uninstall.", path),
			locale.Tl("err_deploy_uninstall_not_deployed_tip", "Either change the current directory to a deployment or supply '--path <path>' arguments."))
	}

	if runtime.GOOS == "windows" && path == cwd {
		return locale.NewInputError(
			"err_deploy_uninstall_cannot_chdir",
			"Cannot remove deployment in current working directory. Please cd elsewhere and run this command again with the '--path' flag.")
	}

	namespace, commitID := sourceAnalyticsInformation(store)

	err := u.subshell.CleanUserEnv(u.cfg, sscommon.DeployID, params.UserScope)
	if err != nil {
		return locale.WrapError(err, "err_deploy_uninstall_env", "Failed to remove deploy directory from PATH")
	}

	err = os.RemoveAll(path)
	if err != nil {
		return locale.WrapError(err, "err_deploy_uninstall", "Unable to remove deployed runtime at '{{.V0}}'", path)
	}

	u.analytics.Event(constants.CatRuntimeUsage, constants.ActRuntimeDelete, &dimensions.Values{
		Trigger:          ptr.To(target.TriggerDeploy.String()),
		CommitID:         ptr.To(commitID),
		ProjectNameSpace: ptr.To(namespace),
		InstanceID:       ptr.To(instanceid.ID()),
	})

	u.output.Notice(locale.T("deploy_uninstall_success"))

	return nil
}

func sourceAnalyticsInformation(store *store.Store) (string, string) {
	namespace, err := store.Namespace()
	if err != nil {
		logging.Error("Could not read namespace from marker file: %v", err)
	}

	commitID, err := store.CommitID()
	if err != nil {
		logging.Error("Could not read commit ID from marker file: %v", err)
	}

	return namespace, commitID
}
