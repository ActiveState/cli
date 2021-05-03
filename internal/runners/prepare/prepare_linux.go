package prepare

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils/autostart"
	"github.com/ActiveState/cli/internal/osutils/shortcut"
	"github.com/gobuffalo/packr"
	"github.com/mitchellh/go-homedir"
)

func (r *Prepare) prepareOS() error {
	trayInfo := appinfo.TrayApp()
	name, exec := trayInfo.Name(), trayInfo.Exec()

	if err := autostart.New(name, exec).Enable(); err != nil {
		r.reportError(locale.Tr(
			"err_prepare_autostart",
			"Could not enable auto-start, error received: {{.V0}}.", err.Error(),
		), err)
	}

	if err := r.setupDesktopApplicationFile(name, exec); err != nil {
		r.reportError(locale.Tr(
			"err_prepare_shortcut_linux",
			"Could not create desktop application file: {{.V0}}.", err.Error(),
		), err)
	}

	return nil
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

	box := packr.NewBox("../../../assets")
	iconData := box.Bytes(constants.TrayIconFileSource)

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
