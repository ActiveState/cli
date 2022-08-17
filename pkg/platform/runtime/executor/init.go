package executor

import (
	"io/ioutil"
	"os"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/pkg/platform/runtime/envdef"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
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

type Targeter interface {
	CommitUUID() strfmt.UUID
	Name() string
	Owner() string
	Dir() string
	Headless() bool
}

type Init struct {
	targeter     Targeter
	executorPath string // The location to store the executors
}

func NewInit(targeter Targeter) (*Init, error) {
	binPath, err := ioutil.TempDir("", "executor")
	if err != nil {
		return nil, errs.New("Could not create tempDir: %v", err)
	}
	return NewInitWithBinPath(targeter, binPath), nil
}

func NewInitWithBinPath(targeter Targeter, executorPath string) *Init {
	return &Init{targeter, executorPath}
}

func (i *Init) BinPath() string {
	return i.executorPath
}

func (i *Init) Apply(env map[string]string, exes envdef.ExecutablePaths) error {
	logging.Debug("Creating executors at %s, exes: %v", i.executorPath, exes)

	// We need to cover the use case of someone running perl.exe/python.exe
	// Proper fix scheduled here https://www.pivotaltracker.com/story/show/177845386
	if rt.GOOS == "windows" {
		for _, exe := range exes {
			if !strings.HasSuffix(exe, exeutils.Extension) {
				continue
			}
			exes = append(exes, exe+exeutils.Extension) // Double up on the ext so only the first on gets dropped
		}
	}

	if err := i.Clean(); err != nil {
		return errs.Wrap(err, "Could not clean up old executors")
	}

	if err := fileutils.MkdirUnlessExists(i.executorPath); err != nil {
		return locale.WrapError(err, "err_mkdir", "Could not create directory: {{.V0}}", i.executorPath)
	}

	sockPath := svcctl.NewIPCSockPathFromGlobals().String()
	for _, exe := range exes {
		f := newFile(i.targeter, i.executorPath)
		if err := f.Save(sockPath, env, exe); err != nil {
			return locale.WrapError(err, "err_createexecutor", "Could not create executor for {{.V0}}.", exe)
		}
	}

	return nil
}

func (i *Init) Clean() error {
	if !fileutils.DirExists(i.executorPath) {
		return nil
	}

	files, err := ioutil.ReadDir(i.executorPath)
	if err != nil {
		return errs.Wrap(err, "Could not read dir: %s", i.executorPath)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(i.executorPath, file.Name())
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
