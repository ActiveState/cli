package uninstall

import (
	"os"
	"runtime"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
)

type Params struct {
	Path      string
	UserScope bool
}

type Uninstall struct {
	output   output.Outputer
	subshell subshell.SubShell
	cfg      *config.Instance
}

type primeable interface {
	primer.Outputer
	primer.Subsheller
	primer.Configurer
}

func NewDeployUninstall(prime primeable) *Uninstall {
	return &Uninstall{prime.Output(), prime.Subshell(), prime.Config()}
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
	if path == "" {
		if runtime.GOOS == "windows" {
			return locale.NewInputError(
				"err_deploy_uninstall_cannot_chdir",
				"Cannot remove deployment in current working directory. Please cd elsewhere and then supply `--path` argument")
		}
		cwd, err := os.Getwd()
		if err != nil {
			return locale.WrapInputError(
				err,
				"err_deploy_uninstall_cannot_get_cwd",
				"Cannot determine current working directory. Please supply `--path` argument")
		}
		path = cwd
	}

	logging.Debug("Attempting to uninstall deployment at %s", path)
	if !store.New(path).HasMarker() {
		return errs.AddTips(
			locale.NewError("err_deploy_uninstall_not_deployed", "There is no deployed runtime at '{{.V0}}' to uninstall.", path),
			locale.Tl("err_deploy_uninstall_not_deployed_tip", "Either change the current directory to a deployment or supply '--path <path>' arguments."))
	}

	err := os.RemoveAll(path)
	if err != nil {
		return locale.WrapError(err, "err_deploy_uninstall", "Unable to remove deployed runtime at '{{.V0}}'", path)
	}

	err = u.subshell.CleanUserEnv(u.cfg, sscommon.DeployID, params.UserScope)
	if err != nil {
		return locale.WrapError(err, "err_deploy_uninstall_env", "Failed to remove deploy directory from PATH")
	}

	u.output.Notice(locale.T("deploy_uninstall_success"))

	return nil
}
