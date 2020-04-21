package deploy

import (
	"os"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/google/uuid"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
)

type Params struct {
	Namespace project.Namespaced
	Path      string
	Step      Step
	Force     bool
}

type Deploy struct {
	output output.Outputer

	DefaultBranchForProjectName defaultBranchForProjectNameFunc
	NewRuntimeInstaller         newInstallerFunc
}

func NewDeploy(out output.Outputer) *Deploy {
	return &Deploy{
		out,
		model.DefaultBranchForProjectName,
		newInstaller,
	}
}

func (d *Deploy) Run(params *Params) error {
	installer, targetPath, err := d.createInstaller(params.Namespace, params.Path)
	if err != nil {
		return locale.WrapError(
			err, "err_deploy_create_install",
			"Could not initialize an installer for {{.V0}}.", params.Namespace.String())
	}

	return runSteps(targetPath, params.Force, params.Step, installer, d.output)
}

func (d *Deploy) createInstaller(namespace project.Namespaced, path string) (installable, string, error) {
	branch, fail := d.DefaultBranchForProjectName(namespace.Owner, namespace.Project)
	if fail != nil {
		return nil, "", errs.Wrap(fail, "Could not create installer")
	}

	if branch.CommitID == nil {
		return nil, "", locale.InputError().New(
			"err_deploy_no_commits",
			"The project '{{.V0}}' does not have any packages configured, please add add some packages first.", namespace.String())
	}

	return d.NewRuntimeInstaller(*branch.CommitID, namespace.Owner, namespace.Project, path)
}

func runSteps(targetPath string, force bool, step Step, installer installable, out output.Outputer) error {
	return runStepsWithFuncs(
		targetPath, force, step, installer, out,
		install, configure, symlink, report)
}

func runStepsWithFuncs(targetPath string, force bool, step Step, installer installable, out output.Outputer, installf installFunc, configuref configureFunc, symlinkf symlinkFunc, reportf reportFunc) error {
	logging.Debug("runSteps: %s", step.String())

	var envGetter runtime.EnvGetter
	var fail *failures.Failure

	if step == UnsetStep || step == InstallStep {
		logging.Debug("Running install step")
		var err error
		if envGetter, err = installf(installer, out); err != nil {
			return err
		}

		if step == UnsetStep {
			out.Notice("") // Some space between steps
		}
	}
	if step == UnsetStep || step == ConfigureStep {
		logging.Debug("Running configure step")
		if envGetter == nil {
			if envGetter, fail = installer.Env(); fail != nil {
				return errs.Wrap(fail, "Could not retrieve env for Configure step")
			}
		}
		if err := configuref(envGetter, out); err != nil {
			return err
		}
		if step == UnsetStep {
			out.Notice("") // Some space between steps
		}
	}
	if rt.GOOS == "linux" && (step == UnsetStep || step == SymlinkStep) {
		logging.Debug("Running symlink step")
		if envGetter == nil {
			if envGetter, fail = installer.Env(); fail != nil {
				return errs.Wrap(fail, "Could not retrieve env for Symlink step")
			}
		}
		if err := symlinkf(targetPath, force, envGetter, out); err != nil {
			return err
		}
		if step == UnsetStep {
			out.Notice("") // Some space between steps
		}
	}
	if step == UnsetStep || step == ReportStep {
		logging.Debug("Running report step")
		if envGetter == nil {
			if envGetter, fail = installer.Env(); fail != nil {
				return errs.Wrap(fail, "Could not retrieve env for Report step")
			}
		}
		if err := reportf(envGetter, out); err != nil {
			return err
		}
	}

	return nil
}

type installFunc func(installer installable, out output.Outputer) (runtime.EnvGetter, error)

func install(installer installable, out output.Outputer) (runtime.EnvGetter, error) {
	out.Notice(locale.T("deploy_install"))
	envGetter, installed, fail := installer.Install()
	if fail != nil {
		return envGetter, errs.Wrap(fail, "Install failed")
	}
	if ! installed {
		out.Notice(locale.T("using_cached_env"))
	}
	return envGetter, nil
}

type configureFunc func(envGetter runtime.EnvGetter, out output.Outputer) error

func configure(envGetter runtime.EnvGetter, out output.Outputer) error {
	venv := virtualenvironment.New(envGetter.GetEnv)
	env := venv.GetEnv(false, "")

	if len(env) == 0 {
		return locale.NewError("err_deploy_run_install", "Please run the install step at least once")
	}

	// Configure Shell
	sshell, fail := subshell.Get()
	if fail != nil {
		return locale.WrapError(fail, "err_deploy_subshell_get", "Could not retrieve information about your shell environment.")
	}
	out.Notice(locale.Tr("deploy_configure_shell", sshell.Shell()))

	fail = sshell.WriteUserEnv(env)
	if fail != nil {
		return locale.WrapError(fail, "err_deploy_subshell_write", "Could not write environment information to your shell configuration.")
	}

	return nil
}

type symlinkFunc func(installPath string, overwrite bool, envGetter runtime.EnvGetter, out output.Outputer) error

func symlink(installPath string, overwrite bool, envGetter runtime.EnvGetter, out output.Outputer) error {
	venv := virtualenvironment.New(envGetter.GetEnv)
	env := venv.GetEnv(false, "")

	if len(env) == 0 {
		return locale.InputError().New("err_deploy_run_install", "Please run the install step at least once")
	}

	// Retrieve path to write symlinks to
	path, err := usablePath()
	if err != nil {
		return errs.Wrap(err, "Could not retrieve a usable PATH")
	}

	// Retrieve artifact binary directory
	var bins []string
	if p, ok := env["PATH"]; ok {
		bins = strings.Split(p, string(os.PathListSeparator))
	}

	// Symlink to PATH (eg. /usr/local/bin)
	if err := symlinkWithTarget(overwrite, path, bins, out); err != nil {
		return errs.Wrap(err, "Could not create symlinks to %s, overwrite: %v.", path, overwrite)
	}

	// Symlink to targetDir/bin
	if err := symlinkWithTarget(overwrite, filepath.Join(installPath, "bin"), bins, out); err != nil {
		return errs.Wrap(err, "Could not create symlinks to %s, overwrite: %v.", path, overwrite)
	}

	return nil
}

func symlinkWithTarget(overwrite bool, path string, bins []string, out output.Outputer) error {
	out.Notice(locale.Tr("deploy_symlink", path))

	if fail := fileutils.MkdirUnlessExists(path); fail != nil {
		return locale.InputError().Wrap(
			fail, "err_deploy_mkdir",
			"Could not create directory at {{.V0}}, make sure you have permissions to write to %s.", path, filepath.Dir(path))
	}

	for _, bin := range bins {
		err := filepath.Walk(bin, func(fpath string, info os.FileInfo, err error) error {
			// Filter out files that are executable
			if info == nil || info.IsDir() || info.Mode()&0111 == 0 { // check if executable by anyone
				return nil // not executable
			}

			// Ensure target is valid
			target := filepath.Join(path, filepath.Base(fpath))
			if fileutils.FileExists(target) {
				if overwrite {
					out.Notice(locale.Tr("deploy_overwrite_target", target))
					if err := os.Remove(target); err != nil {
						return locale.InputError().Wrap(
							err, "err_deploy_overwrite",
							"Could not overwrite {{.V0}}, make sure you have permissions to write to this file.", target)
					}
				} else {
					return locale.InputError().New(
						"err_deploy_symlink_target_exists",
						"Cannot create symlink as the target already exists: {{.V0}}. Use '--force' to overwrite any existing files.", target)
				}
			}

			// Create symlink
			err = os.Symlink(fpath, target)
			if err != nil {
				return locale.InputError().Wrap(
					err, "err_deploy_symlink",
					"Cannot create symlink at {{.V0}}, ensure you have permission to write to {{.V1}}.", target, filepath.Dir(target))
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

type Report struct {
	BinaryDirectories []string
	Environment       map[string]string
}

type reportFunc func(envGetter runtime.EnvGetter, out output.Outputer) error

func report(envGetter runtime.EnvGetter, out output.Outputer) error {
	venv := virtualenvironment.New(envGetter.GetEnv)
	env := venv.GetEnv(false, "")

	if len(env) == 0 {
		return locale.NewError("err_deploy_run_install", "Please run the install step at least once")
	}

	var bins []string

	if path, ok := env["PATH"]; ok {
		delete(env, "PATH")
		bins = strings.Split(path, string(os.PathListSeparator))
	}

	out.Notice(locale.T("deploy_info"))

	out.Print(Report{
		BinaryDirectories: bins,
		Environment:       env,
	})

	out.Notice(locale.T("deploy_restart_shell"))

	return nil
}

// usablePath will find a writable directory under PATH
func usablePath() (string, error) {
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	if len(paths) == 0 {
		return "", locale.InputError().New("err_deploy_path_empty", "Your system does not have any PATH entries configured, so symlinks can not be created.")
	}

	preferredPaths := []string{
		"/usr/local/bin",
		"/usr/bin",
	}
	var result string
	for _, path := range paths {
		// Check if we can write to this path
		fpath := filepath.Join(path, uuid.New().String())
		if err := fileutils.Touch(fpath); err != nil {
			continue
		}
		if errr := os.Remove(fpath); errr != nil {
			logging.Error("Could not clean up test file: %v", errr)
		}

		// Record result
		if funk.Contains(preferredPaths, path) {
			return path, nil
		}
		result = path
	}

	if result != "" {
		return result, nil
	}

	return "", locale.InputError().New("err_deploy_path_noperm", "No permission to create symlinks on any of the PATH entries: {{.V0}}.", os.Getenv("PATH"))
}
