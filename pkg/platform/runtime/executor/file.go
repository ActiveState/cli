package executor

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/strutils"
	"github.com/ActiveState/cli/pkg/project"
)

type file struct {
	dir string
	t   Targeter
}

func newFile(t Targeter, dir string) *file {
	return nil
}

func (f *file) Save(sockPath string, env map[string]string, exe string) error {
	name := NameForExe(filepath.Base(exe))
	target := filepath.Clean(filepath.Join(f.dir, name))

	if strings.HasSuffix(exe, exeutils.Extension+exeutils.Extension) {
		// This is super awkward, but we have a double .exe to temporarily work around an issue that will be fixed
		// more correctly here - https://www.pivotaltracker.com/story/show/177845386
		exe = strings.TrimSuffix(exe, exeutils.Extension)
	}

	logging.Debug("Creating executor for %s at %s", exe, target)

	denoteTarget := executorTarget + exe

	if fileutils.TargetExists(target) {
		b, err := fileutils.ReadFile(target)
		if err != nil {
			return locale.WrapError(err, "err_createexecutor_exists_noread", "Could not create executor as target already exists and could not be read: {{.V0}}.", target)
		}
		if !isOwnedByUs(b) {
			return locale.WrapError(err, "err_createexecutor_exists", "Could not create executor as target already exists: {{.V0}}.", target)
		}
		if strings.Contains(string(b), denoteTarget) {
			return nil
		}
	}

	executorExec, err := installation.ExecutorExec()
	if err != nil {
		return locale.WrapError(err, "err_state_exec")
	}

	tplParams := map[string]interface{}{
		"stateExec":  executorExec,
		"stateSock":  sockPath,
		"targetFile": exe,
		"denote":     []string{executorDenoter, denoteTarget},
		"Env":        env,
		"commitID":   f.t.CommitUUID().String(),
		"nameSpace":  project.NewNamespace(f.t.Owner(), f.t.Name(), f.t.CommitUUID().String()).String(),
		"headless":   fmt.Sprintf("%t", f.t.Headless()),
	}
	boxFile := "executor.sh"
	if rt.GOOS == "windows" {
		boxFile = "executor.bat"
	}
	fwBytes, err := assets.ReadFileBytes(fmt.Sprintf("executors/%s", boxFile))
	if err != nil {
		return errs.Wrap(err, "Failed to read asset")
	}
	fwStr, err := strutils.ParseTemplate(string(fwBytes), tplParams)
	if err != nil {
		return errs.Wrap(err, "Could not parse %s template", boxFile)
	}

	if err = ioutil.WriteFile(target, []byte(fwStr), 0755); err != nil {
		return locale.WrapError(err, "Could not create executor for {{.V0}} at {{.V1}}.", exe, target)
	}

	return nil
}
