package clean

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/installmgr"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/svcctl"
)

func stopServices(cfg configurable, out output.Outputer, ipComm svcctl.IPCommunicator, ignoreErrors bool) error {
	cleanForceTip := locale.Tl("clean_force_tip", "You can re-run the command with the [ACTIONABLE]--force[/RESET] flag.")

	// On Windows we need to halt the state tray and the state service before we can remove them
	svcExec, err := installation.NewExec(installation.ServiceApp)
	if err != nil {
		return locale.WrapError(err, "err_service_info")
	}

	// Todo: https://www.pivotaltracker.com/story/show/177585085
	// Yes this is awkward right now
	if err := installmgr.StopTrayApp(cfg); err != nil {
		if !ignoreErrors {
			return errs.AddTips(
				locale.WrapError(err, "clean_stop_tray_failure", "Cleanup interrupted, because a running {{.V0}} process could not be stopped.", constants.TrayAppName),
				cleanForceTip)
		}
		out.Error(locale.Tl("clean_stop_tray_warning", "Failed to stop running {{.V0}} process. Continuing anyways, because --force flag was provided.", constants.TrayAppName))
	}

	// Stop state-svc before accessing its files
	if fileutils.FileExists(svcExec) {
		code, _, err := exeutils.Execute(svcExec, []string{"stop"}, nil)
		if err != nil {
			if !ignoreErrors {
				return errs.AddTips(
					locale.WrapError(err, "clean_stop_svc_failure", "Cleanup interrupted, because a running {{.V0}} process could not be stopped.", constants.SvcAppName),
					cleanForceTip)
			}
			out.Error(locale.Tl("clean_stop_svc_warning", "Failed to stop running {{.V0}} process. Continuing anyway because --force flag was provided.", constants.SvcAppName))
		}
		if code != 0 {
			if !ignoreErrors {
				return errs.AddTips(
					locale.WrapError(err, "clean_stop_svc_failure_code", "Cleanup interrupted, because a running {{.V0}} process could not be stopped (invalid exit code).", constants.SvcAppName),
					cleanForceTip)
			}
			out.Error(locale.Tl("clean_stop_svc_warning_code", "Failed to stop running {{.V0}} process (invalid exit code). Continuing anyway because --force flag was provided.", constants.SvcAppName))
		}

		if err := svcctl.StopServer(ipComm); err != nil {
			if !ignoreErrors {
				return errs.AddTips(
					locale.WrapError(err, "clean_stop_svc_failure_wait", "Cleanup interrupted, because a running {{.V0}} process failed to stop due to a timeout.", constants.SvcAppName),
					cleanForceTip)
			}
			out.Error(locale.Tl("clean_stop_svc_warning_code", "Failed to stop running {{.V0}} process due to a timeout. Continuing anyway because --force flag was provided.", constants.SvcAppName))
		}
	}
	return nil
}
