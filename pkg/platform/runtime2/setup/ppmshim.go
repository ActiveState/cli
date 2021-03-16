package setup

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/strutils"
	"github.com/gobuffalo/packr"
)

// installPPMShim installs an executable shell script and a BAT file that is executed instead of PPM in the specified path.
// It calls the `state _ppm` sub-command printing deprecation messages.
func installPPMShim(binPath string) error {
	// remove old ppm command if it existed before
	ppmExeName := "ppm"
	if runtime.GOOS == "windows" {
		ppmExeName = "ppm.exe"
	}
	ppmExe := filepath.Join(binPath, ppmExeName)
	if fileutils.FileExists(ppmExe) {
		err := os.Remove(ppmExe)
		if err != nil {
			return errs.Wrap(err, "failed to remove existing ppm %s", ppmExe)
		}
	}

	box := packr.NewBox("../../../../assets/ppm")
	ppmBytes := box.Bytes("ppm.sh")
	shim := filepath.Join(binPath, "ppm")
	// remove shim if it existed before, so we can overwrite (ok to drop error here)
	_ = os.Remove(shim)

	exe, err := os.Executable()
	if err != nil {
		return errs.Wrap(err, "Could not get executable")
	}

	tplParams := map[string]interface{}{"exe": exe}
	ppmStr, err := strutils.ParseTemplate(string(ppmBytes), tplParams)
	if err != nil {
		return errs.Wrap(err, "Could not parse ppm.sh template")
	}

	err = ioutil.WriteFile(shim, []byte(ppmStr), 0755)
	if err != nil {
		return errs.Wrap(err, "failed to write shim command %s", shim)
	}
	if runtime.GOOS == "windows" {
		ppmBatBytes := box.Bytes("ppm.bat")
		shim := filepath.Join(binPath, "ppm.bat")
		// remove shim if it existed before, so we can overwrite (ok to drop error here)
		_ = os.Remove(shim)

		ppmBatStr, err := strutils.ParseTemplate(string(ppmBatBytes), tplParams)
		if err != nil {
			return errs.Wrap(err, "Could not parse ppm.sh template")
		}

		err = ioutil.WriteFile(shim, []byte(ppmBatStr), 0755)
		if err != nil {
			return errs.Wrap(err, "failed to write shim command %s", shim)
		}
	}

	return nil
}
