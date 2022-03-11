package prepare

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils/autostart"
	"github.com/ActiveState/cli/internal/osutils/shortcut"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/mitchellh/go-homedir"
)

func (r *Prepare) prepareOS() {
	trayInfo := appinfo.TrayApp()
	name, exec := trayInfo.Name(), trayInfo.Exec()

	if err := r.setupDesktopApplicationFile(name, exec); err != nil {
		r.reportError(locale.Tr(
			"err_prepare_shortcut_linux",
			"Could not create desktop application file: {{.V0}}.", err.Error(),
		), err)
	}

	return
}

func (r *Prepare) setupDesktopApplicationFile(name, exec string) error {
	dir, err := prependHomeDir(constants.ApplicationDir)
	if err != nil {
		return errs.Wrap(err, "Could not find application directory")
	}
	path := filepath.Join(dir, constants.TrayLaunchFileName)

	iconsDir, err := prependHomeDir(constants.IconsDir)
	if err != nil {
		return errs.Wrap(err, "Could not find icons directory")
	}
	iconsPath := filepath.Join(iconsDir, constants.TrayIconFileName)

	iconData, err := assets.ReadFileBytes(constants.TrayIconFileSource)
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}

	scutOpts := shortcut.SaveOpts{
		Name:        name,
		GenericName: constants.TrayGenericName,
		Comment:     constants.TrayComment,
		Keywords:    constants.TrayKeywords,
		IconData:    iconData,
		IconPath:    iconsPath,
	}
	if _, err := shortcut.Save(exec, path, scutOpts); err != nil {
		return errs.Wrap(err, "Could not save shortcut")
	}

	return nil
}

func prependHomeDir(path string) (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	return filepath.Join(homeDir, path), nil
}

// InstalledPreparedFiles returns the files installed by state _prepare
func InstalledPreparedFiles(cfg autostart.Configurable) []string {
	var files []string
	trayInfo := appinfo.TrayApp()
	name, exec := trayInfo.Name(), trayInfo.Exec()

	shortcut, err := autostart.New(name, exec, cfg).Path()
	if err != nil {
		logging.Error("Failed to determine shortcut path for removal: %v", err)
		rollbar.Error("Failed to determine shortcut path for removal: %v", err)
	} else if shortcut != "" {
		files = append(files, shortcut)
	}

	dir, err := prependHomeDir(constants.ApplicationDir)
	if err != nil {
		logging.Error("Failed to set application dir: %v", err)
		rollbar.Error("Failed to set application dir: %v", err)
	} else {
		files = append(files, filepath.Join(dir, constants.TrayLaunchFileName))
	}

	iconsDir, err := prependHomeDir(constants.IconsDir)
	if err != nil {
		logging.Error("Could not find icons directory: %v", err)
		rollbar.Error("Could not find icons directory: %v", err)
	} else {
		files = append(files, filepath.Join(iconsDir, constants.TrayIconFileName))
	}
	return files
}
