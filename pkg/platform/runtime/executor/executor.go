package executor

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/pkg/platform/runtime/envdef"

	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/strutils"
)

// executorDenoter is used to communicate to the user that this file is generated as well as for us to track
// ownership of the file. Changing this will break updates to older executors that might need to be updated.
const executorDenoter = "!DO NOT EDIT! State Tool Executor !DO NOT EDIT!"

// shimDenoter is our old denoter that we want to make sure we clean up
const shimDenoter = "!DO NOT EDIT! State Tool Shim !DO NOT EDIT!"

// executorTarget tracks the target executable of the executor and is used to determine whether an existing
// executor needs to be updating.
// Update this if you want to blow away older targets (ie. you made changes to the template)
const executorTarget = "Target: "

type Executor struct {
	targetPath   string // The path of a project or a runtime
	executorPath string // The location to store the executors
}

func New(targetPath string) (*Executor, error) {
	binPath, err := ioutil.TempDir("", "executor")
	if err != nil {
		return nil, errs.New("Could not create tempDir: %v", err)
	}
	return NewWithBinPath(targetPath, binPath), nil
}

func NewWithBinPath(targetPath, executorPath string) *Executor {
	return &Executor{targetPath, executorPath}
}

func (f *Executor) BinPath() string {
	return f.executorPath
}

func (f *Executor) Update(sockPath string, exes envdef.ExecutablePaths) error {
	logging.Debug("Creating executors at %s, exes: %v", f.executorPath, exes)

	// We need to cover the use case of someone running perl.exe/python.exe
	// Proper fix scheduled here https://www.pivotaltracker.com/story/show/177845386
	if rt.GOOS == "windows" {
		for _, exe := range exes {
			if !strings.HasSuffix(exe, exeutils.Extension) {
				continue
			}

			if fileutils.FileExists(exe) {
				fixedExe, err := fileutils.CaseSensitivePath(exe)
				if err != nil {
					return locale.WrapError(err, "err_createexecutor_path_fix", "Could not create executor for {{.V0}} (case sensitivity correction failed)", exe)
				}
				exe = strings.ReplaceAll(fixedExe, "c:", "C:")
			}

			exes = append(exes, exe+exeutils.Extension) // Double up on the ext so only the first on gets dropped
		}
	}

	if err := f.Cleanup(); err != nil {
		return errs.Wrap(err, "Could not clean up old executors")
	}

	for _, exe := range exes {
		if rt.GOOS == "windows" && fileutils.FileExists(exe) {
			fixedExe, err := fileutils.CaseSensitivePath(exe)
			if err != nil {
				return locale.WrapError(err, "err_createexecutor_path_fix", "Could not create executor for {{.V0}} (case sensitivity correction failed)", exe)
			}
			exe = strings.ReplaceAll(fixedExe, "c:", "C:")
		}

		if err := f.createExecutor(sockPath, exe); err != nil {
			return locale.WrapError(err, "err_createexecutor", "Could not create executor for {{.V0}}.", exe)
		}
	}

	return nil
}

func (f *Executor) Cleanup() error {
	if !fileutils.DirExists(f.executorPath) {
		return nil
	}

	files, err := ioutil.ReadDir(f.executorPath)
	if err != nil {
		return errs.Wrap(err, "Could not read dir: %s", f.executorPath)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(f.executorPath, file.Name())
		b, err := fileutils.ReadFile(filePath)
		if err != nil {
			return locale.WrapError(err, "err_cleanexecutor_noread", "Could not read potential executor file: {{.V0}}.", file.Name())
		}
		if !isOwnedByUs(b) {
			continue
		}

		if err := os.Remove(filePath); err != nil {
			return locale.WrapError(err, "err_cleanexecutor_remove", "Could not remove executor: {{.V0}}", file.Name())
		}
	}

	return nil
}

func (f *Executor) createExecutor(sockPath, exe string) error {
	name := NameForExe(filepath.Base(exe))
	target := filepath.Clean(filepath.Join(f.executorPath, name))

	if strings.HasSuffix(exe, exeutils.Extension+exeutils.Extension) {
		// This is super awkward, but we have a double .exe to temporarily work around an issue that will be fixed
		// more correctly here - https://www.pivotaltracker.com/story/show/177845386
		exe = strings.TrimSuffix(exe, exeutils.Extension)
	}

	if err := fileutils.MkdirUnlessExists(f.executorPath); err != nil {
		return locale.WrapError(err, "err_mkdir", "Could not create directory: {{.V0}}", f.executorPath)
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

	fmt.Printf(
		"exists... exec: %t, sock: %t, target: %t\n",
		fileutils.FileExists(executorExec),
		fileutils.FileExists(sockPath),
		fileutils.FileExists(exe),
	)
	if !fileutils.FileExists(sockPath) {
		files, _ := ioutil.ReadDir(filepath.Dir(sockPath))
		for _, file := range files {
			fmt.Println(file.Name())
		}
	}

	tplParams := map[string]interface{}{
		"stateExec": executorExec,
		"stateSock": sockPath,
		"target":    exe,
		"denote":    []string{executorDenoter, denoteTarget},
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

func IsExecutor(filePath string) (bool, error) {
	if fileutils.IsDir(filePath) {
		return false, nil
	}

	b, err := fileutils.ReadFile(filePath)
	if err != nil {
		return false, locale.WrapError(err, "err_cleanexecutor_noread", "Could not read potential executor file: {{.V0}}.", filePath)
	}
	return isOwnedByUs(b), nil
}

func isOwnedByUs(fileContents []byte) bool {
	if strings.Contains(string(fileContents), executorDenoter) ||
		strings.Contains(string(fileContents), shimDenoter) {
		return true
	}
	return false
}
