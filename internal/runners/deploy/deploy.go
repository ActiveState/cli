package deploy

import (
	"os"
	"path/filepath"
	rt "runtime"
	"strings"

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
	step   Step

	DefaultBranchForProjectName defaultBranchForProjectNameFunc
	NewRuntimeInstaller         newInstallerFunc
}

func NewDeploy(step Step, out output.Outputer) *Deploy {
	return &Deploy{
		out,
		step,
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

	return runSteps(targetPath, params.Force, d.step, installer, d.output)
}

func (d *Deploy) createInstaller(namespace project.Namespaced, path string) (installable, string, error) {
	branch, fail := d.DefaultBranchForProjectName(namespace.Owner, namespace.Project)
	if fail != nil {
		return nil, "", errs.Wrap(fail, "Could not create installer")
	}

	if branch.CommitID == nil {
		return nil, "", locale.NewInputError(
			"err_deploy_no_commits",
			"The project '{{.V0}}' does not have any packages configured, please add add some packages first.", namespace.String())
	}

	installable, cacheDir, fail := d.NewRuntimeInstaller(*branch.CommitID, namespace.Owner, namespace.Project, path)
	return installable, cacheDir, fail.ToError()
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

	installed, fail := installer.IsInstalled()
	if fail != nil {
		return fail
	}

	if !installed && step != UnsetStep && step != InstallStep {
		return locale.NewInputError("err_deploy_run_install", "Please run the install step at least once")
	}

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
	if step == UnsetStep || step == SymlinkStep {
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
	if !installed {
		out.Notice(locale.T("using_cached_env"))
	}
	out.Print(locale.Tl("deploy_install_done", "Installation completed"))
	return envGetter, nil
}

type configureFunc func(envGetter runtime.EnvGetter, out output.Outputer) error

func configure(envGetter runtime.EnvGetter, out output.Outputer) error {
	venv := virtualenvironment.New(envGetter.GetEnv)
	env, err := venv.GetEnv(false, "")
	if err != nil {
		return err
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
	env, err := venv.GetEnv(false, "")
	if err != nil {
		return err
	}

	// Retrieve path to write symlinks to
	path, err := usablePath()
	if err != nil {
		return locale.WrapError(err, "err_usablepath", "Could not retrieve a usable PATH")
	}

	// Retrieve artifact binary directory
	var bins []string
	if p, ok := env["PATH"]; ok {
		bins = strings.Split(p, string(os.PathListSeparator))
	}

	var pathExt []string
	if rt.GOOS == "windows" {
		var pes string
		var ok bool
		if pes, ok = env["PATHEXT"]; !ok {
			pes = os.Getenv("PATHEXT")
		}
		pathExt = strings.Split(pes, ";")
	}

	if rt.GOOS == "linux" {
		// Symlink to PATH (eg. /usr/local/bin)
		if err := symlinkWithTarget(overwrite, path, bins, pathExt, out); err != nil {
			return locale.WrapError(err, "err_symlink", "Could not create symlinks to {{.V0}}.", path)
		}
	}

	// Symlink to targetDir/bin
	if err := symlinkWithTarget(overwrite, filepath.Join(installPath, "bin"), bins, pathExt, out); err != nil {
		return locale.WrapError(err, "Could not create symlinks to {{.V0}}.", path)
	}

	return nil
}

// maySymlink enforces that only executables in the most prioritized PATH directory are symlinked.
// If a symlink to a file of the same name (possibly with a different file extension on
// Windows) in a different directory exists has been created already, it returns false.
func maySymlink(path string, symlinkedFiles map[string]string) bool {
	oldPath, exists := symlinkedFiles[fileNameBase(path)]
	// if not file of that name has been written yet, then symlinking is okay
	if !exists {
		return true
	}

	// if the the new path is in a different directory, we should not symlink, as the previously symlinked file was in a higher priority PATH
	return filepath.Dir(oldPath) == filepath.Dir(path)
}

// shouldOverwriteSymlink enforces that only executables with the most prioritized PATHEXT extension (on Windows) are symlinked.
// It returns two booleans, the first one indicating whether the symlink should be overwritten, the second one indicating whether it is allowed.
// On Linux and MacOS it always returns true
// On Windows only, if the new path has a higher priority extension than previously symlinked exectuables.
func shouldOverwriteSymlink(overwrite bool, path string, symlinkedFiles map[string]string, pathExt []string) (bool, bool) {
	// on non-windows systems, overwrite says it all
	if rt.GOOS != "windows" {
		return true, overwrite
	}

	oldPath, exists := symlinkedFiles[fileNameBase(path)]
	// if it is a new file, then the overwrite flag decide whether we are allowed to overwrite the file
	if !exists {
		return true, overwrite
	}

	// Only overwrite if this path has a higher pathext priority
	oldExt := filepath.Ext(oldPath)
	ext := filepath.Ext(path)
	for _, pe := range pathExt {
		if strings.ToLower(oldExt) == strings.ToLower(pe) {
			return false, false
		}
		if strings.ToLower(ext) == strings.ToLower(pe) {
			return true, true
		}
	}

	// this should not happen: none of the pathes has pathext extension
	return false, false
}

func fileNameBase(path string) string {
	base := filepath.Base(path)
	if rt.GOOS == "windows" {
		ext := filepath.Ext(path)
		base = base[0 : len(base)-len(ext)]
	}
	return base
}

func linkTarget(targetDir string, path string) string {
	target := filepath.Clean(filepath.Join(targetDir, filepath.Base(path)))
	if rt.GOOS != "windows" {
		return target
	}

	oldExt := filepath.Ext(target)
	return target[0:len(target)-len(oldExt)] + ".lnk"
}

func symlinkWritten(path string, symlinkedFiles map[string]string) map[string]string {
	key := fileNameBase(path)

	symlinkedFiles[key] = path
	return symlinkedFiles
}

// symlinkWithTarget creates symlinks in the target path of all executables found in the bins dir
// It overwrites existing files, if the overwrite flag is set.
// On Windows the same executable name can have several file extensions,
// therefore executables are only symlinked if it has not been symlinked to a
// target (with the same or a different extension) from a different directory.
// Also: Only the executable with the highest priority according to pathExt is symlinked.
func symlinkWithTarget(overwrite bool, path string, bins []string, pathExt []string, out output.Outputer) error {
	out.Notice(locale.Tr("deploy_symlink", path))

	if fail := fileutils.MkdirUnlessExists(path); fail != nil {
		return locale.WrapInputError(
			fail, "err_deploy_mkdir",
			"Could not create directory at {{.V0}}, make sure you have permissions to write to %s.", path, filepath.Dir(path))
	}

	symlinkedFiles := make(map[string]string)
	for _, bin := range bins {
		err := filepath.Walk(bin, func(fpath string, info os.FileInfo, err error) error {
			// Filter out files that are not executable
			if info == nil || info.IsDir() || !fileutils.IsExecutable(fpath) { // check if executable by anyone
				return nil // not executable
			}

			// Ensure target is valid
			target := linkTarget(path, fpath)
			if !maySymlink(fpath, symlinkedFiles) {
				return nil
			}
			if fileutils.TargetExists(target) {
				doOverwrite, allowed := shouldOverwriteSymlink(overwrite, fpath, symlinkedFiles, pathExt)
				if doOverwrite {
					if !allowed {
						return locale.NewInputError(
							"err_deploy_symlink_target_exists",
							"Cannot create symlink as the target already exists: {{.V0}}. Use '--force' to overwrite any existing files.", target)
					}
					out.Notice(locale.Tr("deploy_overwrite_target", target))
					if err := os.Remove(target); err != nil {
						return locale.WrapInputError(
							err, "err_deploy_overwrite",
							"Could not overwrite {{.V0}}, make sure you have permissions to write to this file.", target)
					}
				}
			}

			symlinkedFiles[fileNameBase(fpath)] = fpath
			return link(fpath, target)
		})
		if err != nil {
			return errs.Wrap(err, "Error while walking path")
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
	env, err := venv.GetEnv(false, "")
	if err != nil {
		return err
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

	if rt.GOOS == "windows" {
		out.Notice(locale.T("deploy_restart_cmd"))
	} else {
		out.Notice(locale.T("deploy_restart_shell"))
	}

	return nil
}

// usablePath will find a writable directory under PATH
func usablePath() (string, error) {
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	if len(paths) == 0 {
		return "", locale.NewInputError("err_deploy_path_empty", "Your system does not have any PATH entries configured, so symlinks can not be created.")
	}

	preferredPaths := []string{
		"/usr/local/bin",
		"/usr/bin",
	}
	var result string
	for _, path := range paths {
		if path == "" || (!fileutils.IsDir(path) && !fileutils.FileExists(path)) || !fileutils.IsWritable(path) {
			continue
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

	return "", locale.NewInputError("err_deploy_path_noperm", "No permission to create symlinks on any of the PATH entries: {{.V0}}.", os.Getenv("PATH"))
}
