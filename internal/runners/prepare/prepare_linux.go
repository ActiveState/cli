package prepare

import (
	"path/filepath"

	"github.com/ActiveState/cli/cmd/state-tray/pkg/autostart"
	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils/shortcut"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/gobuffalo/packr"
	"github.com/mitchellh/go-homedir"
)

func (r *Prepare) prepareOS() error {
	if rtutils.BuiltViaCI { // disabled while we're still testing this functionality
		return nil
	}

	if err := autostart.New().Enable(); err != nil {
		r.reportError(locale.Tr("err_prepare_autostart", "Could not enable auto-start, error received: {{.V0}}.", err.Error()), err)
	}

	if err := r.prepareDesktopShortcut(); err != nil {
		r.reportError(locale.Tr("err_prepare_shortcut", "Could not create start menu shortcut, error received: {{.V0}}.", err.Error()), err)
	}

	return nil
}

func (r *Prepare) prepareDesktopShortcut() error {
	dir, err := dirPath(constants.ApplicationDir)
	if err != nil {
		return errs.Wrap(err, "Could not find application directory")
	}
	path := filepath.Join(dir, constants.TrayLaunchFileName)

	scut, err := shortcut.New(appinfo.TrayApp().Exec(), path)
	if err != nil {
		return errs.Wrap(err, "Could not construct shortcut")
	}

	iconsDir, err := dirPath(constants.IconsDir)
	if err != nil {
		return errs.Wrap(err, "")
	}
	iconsPath := filepath.Join(iconsDir, constants.TrayIconFileName)

	box := packr.NewBox("../../../assets")
	iconData := box.Bytes(constants.TrayIconFileSource)

	scutOpts := shortcut.ShortcutSaveOpts{
		GenericName: constants.TrayGenericName,
		Comment:     constants.TrayComment,
		Keywords:    constants.TrayKeywords,
		IconData:    iconData,
		IconPath:    iconsPath,
	}
	if err := scut.Save(constants.TrayAppName, scutOpts); err != nil {
		return errs.Wrap(err, "Could not save shortcut")
	}

	return nil
}

func dirPath(dir string) (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	return filepath.Join(homeDir, dir), nil
}
