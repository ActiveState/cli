package executors

import (
	"io/ioutil"
	"os"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/ActiveState/cli/pkg/project"

	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/pkg/platform/runtime/envdef"
	"github.com/ActiveState/cli/pkg/platform/runtime/executors/execmeta"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

type Targeter interface {
	CommitUUID() strfmt.UUID
	Name() string
	Owner() string
	Dir() string
}

type Executors struct {
	executorPath string // The location to store the executors

	altExecSrcPath string // Path to alternate executor for testing. Executor() will use global func if not set.
}

func New(executorPath string) *Executors {
	return &Executors{
		executorPath: executorPath,
	}
}

func (es *Executors) ExecutorSrc() (string, error) {
	if es.altExecSrcPath != "" {
		return es.altExecSrcPath, nil
	}
	return installation.ExecutorExec()
}

func (es *Executors) Apply(sockPath string, targeter Targeter, env map[string]string, exes envdef.ExecutablePaths) error {
	logging.Debug("Creating executors at %s, exes: %v", es.executorPath, exes)

	executors := make(map[string]string) // map[alias]dest
	for _, dest := range exes {
		executors[makeAlias(dest)] = dest
	}

	if err := es.Clean(); err != nil {
		return errs.Wrap(err, "Could not clean up old executors")
	}

	if err := fileutils.MkdirUnlessExists(es.executorPath); err != nil {
		return locale.WrapError(err, "err_mkdir", "Could not create directory: {{.V0}}", es.executorPath)
	}

	ns := project.NewNamespace(targeter.Owner(), targeter.Name(), "")
	t := execmeta.Target{
		CommitUUID: targeter.CommitUUID().String(),
		Namespace:  ns.String(),
		Dir:        targeter.Dir(),
	}
	m := execmeta.New(sockPath, osutils.EnvMapToSlice(env), t, executors)
	if err := m.WriteToDisk(es.executorPath); err != nil {
		return err
	}

	executorSrc, err := es.ExecutorSrc()
	if err != nil {
		return locale.WrapError(err, "err_state_exec")
	}

	for executor := range executors {
		if err := copyExecutor(es.executorPath, executor, executorSrc); err != nil {
			return locale.WrapError(err, "err_createexecutor", "Could not create executor for {{.V0}}.", executor)
		}
	}

	return nil
}

func makeAlias(destination string) string {
	alias := filepath.Base(destination)

	if rt.GOOS == "windows" {
		ext := filepath.Ext(alias)
		if ext != "" && ext != osutils.ExeExtension { // for non-.exe executables like pip.bat
			alias = strings.TrimSuffix(alias, ext) + osutils.ExeExtension // setup alias pip.exe -> pip.bat
		}
	}

	return alias
}

func (es *Executors) Clean() error {
	if !fileutils.DirExists(es.executorPath) {
		return nil
	}

	files, err := ioutil.ReadDir(es.executorPath)
	if err != nil {
		return errs.Wrap(err, "Could not read dir: %s", es.executorPath)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(es.executorPath, file.Name())
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
	return strings.Contains(string(fileContents), "state-exec") ||
		execmeta.IsMetaFile(fileContents) ||
		legacyIsOwnedByUs(fileContents)
}

func copyExecutor(destDir, executor, srcExec string) error {
	name := filepath.Base(executor)
	target := filepath.Clean(filepath.Join(destDir, name))

	logging.Debug("Creating executor for %s at %s", name, target)

	if fileutils.TargetExists(target) {
		b, err := fileutils.ReadFile(target)
		if err != nil {
			return locale.WrapError(err, "err_createexecutor_exists_noread", "Could not create executor as target already exists and could not be read: {{.V0}}.", target)
		}
		if !isOwnedByUs(b) {
			return locale.WrapError(err, "err_createexecutor_exists", "Could not create executor as target already exists: {{.V0}}.", target)
		}
	}

	if err := fileutils.CopyFile(srcExec, target); err != nil {
		return locale.WrapError(err, "err_copyexecutor_fail", "Could not copy {{.V0}} to {{.V1}}", srcExec, target)
	}

	if err := os.Chmod(target, 0755); err != nil {
		return locale.WrapError(err, "err_setexecmode_fail", "Could not set mode of {{.V0}}", target)
	}

	return nil
}

// denoter constants are to ensure we clean up old executors, but are deprecated as of this comment
const (
	legacyExecutorDenoter = "!DO NOT EDIT! State Tool Executor !DO NOT EDIT!"
	legacyShimDenoter     = "!DO NOT EDIT! State Tool Shim !DO NOT EDIT!"
)

func legacyIsOwnedByUs(fileContents []byte) bool {
	s := string(fileContents)
	return strings.Contains(s, legacyExecutorDenoter) || strings.Contains(s, legacyShimDenoter)
}
