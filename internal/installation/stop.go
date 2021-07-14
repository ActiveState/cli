package installation

import (
	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/rtutils"
)

func StopRunning(installPath string) (rerr error) {
	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not get config")
	}
	defer rtutils.Closer(cfg.Close, &rerr)
	
	svcInfo := appinfo.SvcApp(installPath)
	trayInfo := appinfo.TrayApp(installPath)

	// Todo: https://www.pivotaltracker.com/story/show/177585085
	// Yes this is awkward right now
	if err := StopTrayApp(cfg); err != nil {
		return errs.Wrap(err, "Failed to stop %s", trayInfo.Name())
	}

	if fileutils.FileExists(svcInfo.Exec()) {
		exitCode, _, err := exeutils.Execute(svcInfo.Exec(), []string{"stop"}, nil)
		if err != nil {
			return errs.Wrap(err, "Stopping %s returned error", svcInfo.Name())
		}
		if exitCode != 0 {
			return errs.New("Stopping %s exited with code %d", svcInfo.Name(), exitCode)
		}
	}

	return nil
}
