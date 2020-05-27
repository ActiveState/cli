package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
)

// VirtualEnvParams manages the request-specific parameters used to run the
// primary VirtualEnv logic.
type VirtualEnvParams struct {
	Path  string
	Force bool
}

// VirtualEnv manages the core dependencies for the primary VirtualEnv logic.
type VirtualEnv struct {
	pj  *project.Project
	out output.Outputer
}

// NewVirtualEnv is a convenience construction function.
func NewVirtualEnv(pj *project.Project, out output.Outputer) *VirtualEnv {
	return &VirtualEnv{
		pj:  pj,
		out: out,
	}
}

type outputFormat struct {
	Path string `locale:"path,Path"`
}

func (f *outputFormat) MarshalOutput(format output.Format) interface{} {
	switch format {
	case output.PlainFormatName:
		return locale.Tl("virtualenv_created_at", "Virtualenv has been created at {{.V0}}.", f.Path)
	}

	return f
}

// Run executes the primary VirtualEnv logic.
func (v *VirtualEnv) Run(params VirtualEnvParams) error {
	logging.Debug("Execute export virtualenv")

	path := params.Path
	if path == "" {
		path = filepath.Join(config.ConfigPath(), "venvs")
	}

	targetDir := filepath.Join(path, fmt.Sprintf("%s-%s", v.pj.Owner(), v.pj.Name()))
	v.out.Notice(locale.Tl("virtualenv_creating", "[INFO]Creating virtualenv at {{.V0}}.[/RESET]", targetDir))
	if fileutils.DirExists(targetDir) {
		if !params.Force && params.Path != "" {
			return locale.NewInputError(
				"err_virtualenv_targetexists",
				"Target path {{.V0}} already contains a directory named '{{.V1}}-{{.V2}}. Run with '--force' to overwrite.",
				path, v.pj.Owner(), v.pj.Name())
		}

		v.out.Notice(locale.Tl("virtualenv_overwrite", "Removing existing target directory."))
		if err := os.RemoveAll(targetDir); err != nil {
			return locale.WrapInputError(err, "err_virtualenv_overwrite", "Could not remove target directory, error returned: {{.V0}}.", err.Error())
		}
	}
	targetDir = filepath.Join(targetDir, "bin")
	if err := os.MkdirAll(targetDir, fileutils.DirMode); err != nil {
		return locale.WrapInputError(err, "err_virtualenv_mkdir", "Could not create target directory, error returned: {{.V0}}.", err.Error())
	}

	installable, fail := runtime.NewInstallerByParams(runtime.NewInstallerParams(
		config.CachePath(),
		v.pj.CommitUUID(),
		v.pj.Owner(),
		v.pj.Name(),
	))
	if fail != nil {
		return locale.WrapError(fail, "err_virtualenv_installer", "Could not collect Runtime Environment information.")
	}

	installed, fail := installable.IsInstalled()
	if fail != nil {
		return locale.WrapError(fail, "err_virtualenv_isinstalled", "Could not determine whether Runtime Environment has already been installed.")
	}
	if !installed {
		return locale.NewInputError("err_virtualenv_notinstalled", "You have to activate your project at least once before you can export a virtualenv for it.")
	}

	if !fileutils.IsWritable(path) {
		return locale.NewInputError("err_virtualenv_notwritable", "The target path '{{.V0}}' is not writable.", path)
	}

	envGetter, fail := installable.Env()
	if fail != nil {
		return locale.WrapError(fail, "err_virtualenv_env", "Could not retrieve environment information.")
	}
	venv := virtualenvironment.New(envGetter.GetEnv)
	env := venv.GetEnv(false, "")

	// Retrieve artifact binary directory
	pathEnv, ok := env["PATH"]
	if !ok {
		return locale.NewInputError("err_virtualenv_nopath", "Runtime Environment has no PATH.")
	}

	binPaths := strings.Split(pathEnv, string(os.PathListSeparator))
	for _, binPath := range binPaths {
		err := filepath.Walk(binPath, func(fpath string, info os.FileInfo, err error) error {
			if info == nil || info.IsDir() || !fileutils.IsExecutable(fpath) { // check if executable by anyone
				return nil // not executable
			}
			return os.Symlink(fpath, filepath.Join(targetDir, filepath.Base(fpath)))
		})
		if err != nil {
			return locale.WrapError(err, "err_virtualenv_symlink", "Could not create symlinks, error returned: {{.V0}}.", err.Error())
		}
	}

	v.out.Print(&outputFormat{targetDir})

	return nil
}
