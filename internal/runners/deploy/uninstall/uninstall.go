package uninstall

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/instanceid"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/rtutils/p"
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
		cwd, err = os.Getwd()
		if err != nil {
			return locale.WrapInputError(
				err,
				"err_deploy_uninstall_cannot_get_cwd",
				"Cannot determine current working directory. Please supply `--path` argument")
		}
		path = cwd
	}

	store := store.New(path)
	if !store.HasMarker() {
		return errs.AddTips(
			locale.NewError("err_deploy_uninstall_not_deployed", "There is no deployed runtime at '{{.V0}}' to uninstall.", path),
			locale.Tl("err_deploy_uninstall_not_deployed_tip", "Either change the current directory to a deployment or supply '--path <path>' arguments."))
	}

	if runtime.GOOS == "windows" && path == cwd {
		return locale.NewInputError(
			"err_deploy_uninstall_cannot_chdir",
			"Cannot remove deployment in current working directory. Please cd elsewhere and run this command again with the '--path' flag.")
	}

	err := u.subshell.CleanUserEnv(u.cfg, sscommon.DeployID, params.UserScope)
	if err != nil {
		return locale.WrapError(err, "err_deploy_uninstall_env", "Failed to remove deploy directory from PATH")
	}

	err = os.RemoveAll(path)
	if err != nil {
		return locale.WrapError(err, "err_deploy_uninstall", "Unable to remove deployed runtime at '{{.V0}}'", path)
	}

	fmt.Println("Has marker:", store.HasMarker())
	contents, err := fileutils.ReadFile(store.MarkerFile())
	fmt.Println("err:", err)
	if err != nil {
		return locale.WrapError(err, "err_deploy_uninstall_marker", "Unable to read marker file at [ACTIONABLE]{{.V0}}[RESET]. Deployment may be corrupted.", store.MarkerFile())
	}

	var namespace string
	var commitID string
	lines := strings.Split(string(contents), "\n")
	if len(lines) < 3 {
		logging.Error("Marker file is incomplete, cannot determine all project information")
	} else {
		commitID = strings.TrimSpace(lines[0])
		namespace = strings.TrimSpace(lines[2])
	}

	u.analytics.Event(constants.CatRuntimeUsage, constants.ActRuntimeDelete, &dimensions.Values{
		Trigger:          p.StrP(target.TriggerDeploy.String()),
		CommitID:         p.StrP(commitID),
		ProjectNameSpace: p.StrP(namespace),
		InstanceID:       p.StrP(instanceid.ID()),
	})

	u.output.Notice(locale.T("deploy_uninstall_success"))

	return nil
}
