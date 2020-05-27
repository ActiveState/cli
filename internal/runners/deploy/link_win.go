// +build windows

package deploy

import (
	"bytes"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gobuffalo/packr"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/scriptfile"
)

func link(fpath, symlink string) error {
	if strings.HasSuffix(symlink, ".exe") {
		symlink = strings.Replace(symlink, ".exe", ".lnk", 1)
	}
	logging.Debug("Creating shortcut, destination: %s symlink: %s", fpath, symlink)

	box := packr.NewBox("../../../assets/scripts/")
	sfile, fail := scriptfile.New(language.PowerShell, "createShortcut", box.String("createShortcut.ps1"))
	if fail != nil {
		return errs.Wrap(fail.ToError(), "Could not create createShortcut.ps1 scriptfile")
	}

	// Some paths may contain spaces so we must quote
	fpath = strconv.Quote(fpath)
	symlink = strconv.Quote(symlink)

	cmd := exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-Command", sfile.Filename(), fpath, symlink)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return locale.WrapError(err, "err_powershell_symlink", "Invoking powershell to create a shortcut failed with error code: {{.V0}}, error: {{.V1}}", err.Error(), out.String())
	}
	return nil
}
