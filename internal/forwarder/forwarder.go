package forwarder

import (
	"io/ioutil"
	"os"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/gobuffalo/packr"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/strutils"
)

// forwardDenoter is used to communicate to the user that this file is generated as well as for us to track
// ownership of the file. Changing this will break updates to older forwards that might need to be updated.
const forwardDenoter = "!DO NOT EDIT! State Tool Forwarder !DO NOT EDIT!"

// shimDenoter is our old denoter that we want to make sure we clean up
const shimDenoter = "!DO NOT EDIT! State Tool Shim !DO NOT EDIT!"

// forwardTarget tracks the target executable of the forward and is used to determine whether an existing
// forward needs to be updating.
// Update this if you want to blow away older targets (ie. you made changes to the template)
const forwardTarget = "Target: "

type Forward struct {
	projectPath string
	binPath     string
}

func New(projectPath string) (*Forward, error) {
	binPath, err := ioutil.TempDir("", "forward-rt")
	if err != nil {
		return nil, errs.New("Could not create tempDir: %v", err)
	}
	return NewWithBinPath(projectPath, binPath), nil
}

func NewWithBinPath(projectPath, binPath string) *Forward {
	return &Forward{projectPath, binPath}
}

func (f *Forward) BinPath() string {
	return f.binPath
}

func (f *Forward) Update(exes runtime.Executables) error {
	logging.Debug("Creating forwarders at %s, exes: %v", f.binPath, exes)

	if err := f.Cleanup(exes); err != nil {
		return errs.Wrap(err, "Could not clean up old forwards")
	}

	for _, exe := range exes {
		if err := f.createForward(exe); err != nil {
			return locale.WrapError(err, "err_createforward", "Could not create forwarder for {{.V0}}.", exe)
		}
	}

	return nil
}

func (f *Forward) Cleanup(keep []string) error {
	if !fileutils.DirExists(f.binPath) {
		return nil
	}

	files, err := ioutil.ReadDir(f.binPath)
	if err != nil {
		return errs.Wrap(err, "Could not read dir: %s", f.binPath)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if containsBase(keep, file.Name()) {
			continue
		}

		filePath := filepath.Join(f.binPath, file.Name())
		b, err := fileutils.ReadFile(filePath)
		if err != nil {
			return locale.WrapError(err, "err_cleanforward_noread", "Could not read potential forward file: {{.V0}}.", file.Name())
		}
		if !isOwnedByUs(b) {
			continue
		}

		if err := os.Remove(filePath); err != nil {
			return locale.WrapError(err, "err_cleanforward_remove", "Could not remove forwarder: {{.V0}}", file.Name())
		}
	}

	return nil
}

func (f *Forward) createForward(exe string) error {
	name := nameForwarder(filepath.Base(exe))
	target := filepath.Clean(filepath.Join(f.binPath, name))

	logging.Debug("Creating forward for %s at %s", exe, target)

	denoteTarget := forwardTarget + exe

	if fileutils.TargetExists(target) {
		b, err := fileutils.ReadFile(target)
		if err != nil {
			return locale.WrapError(err, "err_createforward_exists_noread", "Could not create forwarder as target already exists and could not be read: {{.V0}}.", target)
		}
		if !isOwnedByUs(b) {
			return locale.WrapError(err, "err_createforward_exists", "Could not create forwarder as target already exists: {{.V0}}.", target)
		}
		if strings.Contains(string(b), denoteTarget) {
			return nil
		}
	}

	tplParams := map[string]interface{}{
		"state":       appinfo.StateApp().Exec(),
		"exe":         filepath.Base(exe),
		"projectPath": f.projectPath,
		"denote":      []string{forwardDenoter, denoteTarget},
	}
	box := packr.NewBox("../../assets/forwarders")
	boxFile := "forwarder.sh"
	if rt.GOOS == "windows" {
		boxFile = "forwarder.bat"
	}
	fwBytes := box.Bytes(boxFile)
	fwStr, err := strutils.ParseTemplate(string(fwBytes), tplParams)
	if err != nil {
		return errs.Wrap(err, "Could not parse %s template", boxFile)
	}

	if err = ioutil.WriteFile(target, []byte(fwStr), 0755); err != nil {
		return locale.WrapError(err, "Could not create forwarder for {{.V0}} at {{.V1}}.", exe, target)
	}

	return nil
}

func containsBase(sourcePaths []string, targetPath string) bool {
	for _, p := range sourcePaths {
		if filepath.Base(p) == filepath.Base(targetPath) {
			return true
		}
	}
	return false
}

func isOwnedByUs(fileContents []byte) bool {
	if strings.Contains(string(fileContents), forwardDenoter) ||
		strings.Contains(string(fileContents), shimDenoter) {
		return true
	}
	return false
}
