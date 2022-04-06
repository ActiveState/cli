package clean

import (
	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/svcctl"
)

func stopServices(cfg configurable, out output.Outputer, ipComm svcctl.IPCommunicator, ignoreErrors bool) error {
	cleanForceTip := locale.Tl("clean_force_tip", "You can re-run the command with the [ACTIONABLE]--force[/RESET] flag.")

	// On Windows we need to halt the state tray and the state service before we can remove them
	svcInfo := appinfo.SvcApp()
	trayInfo := appinfo.TrayApp()

	// Todo: https://www.pivotaltracker.com/story/show/177585085
	// Yes this is awkward right now
	if err := installation.StopTrayApp(cfg); err != nil {
		if !ignoreErrors {
			return errs.AddTips(
				locale.WrapError(err, "clean_stop_tray_failure", "Cleanup interrupted, because a running {{.V0}} process could not be stopped.", trayInfo.Name()),
				cleanForceTip)
		}
		out.Error(locale.Tl("clean_stop_tray_warning", "Failed to stop running {{.V0}} process. Continuing anyways, because --force flag was provided.", trayInfo.Name()))
	}

	// Stop state-svc before accessing its files
	if fileutils.FileExists(svcInfo.Exec()) {
		code, _, err := exeutils.Execute(svcInfo.Exec(), []string{"stop"}, nil)
		if err != nil {
			if !ignoreErrors {
				return errs.AddTips(
					locale.WrapError(err, "clean_stop_svc_failure", "Cleanup interrupted, because a running {{.V0}} process could not be stopped.", svcInfo.Name()),
					cleanForceTip)
			}
			out.Error(locale.Tl("clean_stop_svc_warning", "Failed to stop running {{.V0}} process. Continuing anyway because --force flag was provided.", svcInfo.Name()))
		}
		if code != 0 {
			if !ignoreErrors {
				return errs.AddTips(
					locale.WrapError(err, "clean_stop_svc_failure_code", "Cleanup interrupted, because a running {{.V0}} process could not be stopped (invalid exit code).", svcInfo.Name()),
					cleanForceTip)
			}
			out.Error(locale.Tl("clean_stop_svc_warning_code", "Failed to stop running {{.V0}} process (invalid exit code). Continuing anyway because --force flag was provided.", svcInfo.Name()))
		}

		if err := svcctl.StopServer(ipComm); err != nil {
			if !ignoreErrors {
				return errs.AddTips(
					locale.WrapError(err, "clean_stop_svc_failure_wait", "Cleanup interrupted, because a running {{.V0}} process failed to stop due to a timeout.", svcInfo.Name()),
					cleanForceTip)
			}
			out.Error(locale.Tl("clean_stop_svc_warning_code", "Failed to stop running {{.V0}} process due to a timeout. Continuing anyway because --force flag was provided.", svcInfo.Name()))
		}
	}
	return nil
}
