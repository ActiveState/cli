package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/gobuffalo/packr"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
)

type Params struct {
	Namespace project.Namespaced
	Path      string
	Force     bool
	UserScope bool
}

// RequiresAdministratorRights checks if the requested deploy command requires administrator privileges.
func RequiresAdministratorRights(step Step, userScope bool) bool {
	if rt.GOOS != "windows" {
		return false
	}
	return (step == UnsetStep || step == ConfigureStep) && !userScope
}

type Deploy struct {
	output   output.Outputer
	subshell subshell.SubShell
	step     Step

	DefaultBranchForProjectName defaultBranchForProjectNameFunc
	NewRuntimeInstaller         newInstallerFunc
}

type primeable interface {
	primer.Outputer
	primer.Subsheller
}

func NewDeploy(step Step, prime primeable) *Deploy {
	return &Deploy{
		prime.Output(),
		prime.Subshell(),
		step,
		model.DefaultBranchForProjectName,
		newInstaller,
	}
}

func (d *Deploy) Run(params *Params) error {
	if RequiresAdministratorRights(d.step, params.UserScope) {
		isAdmin, err := osutils.IsWindowsAdmin()
		if err != nil {
			logging.Error("Could not check for windows administrator privileges: %v", err)
		}
		if !isAdmin {
			return locale.NewError("err_deploy_admin_privileges_required", "Administrator rights are required for this command to modify the system PATH.  If you want to deploy to the user environment, please adjust the command line flags.")
		}
	}

	targetPath := params.Path
	runtime, installer, err := d.createRuntimeInstaller(params.Namespace, targetPath)
	if err != nil {
		return locale.WrapError(
			err, "err_deploy_create_install",
			"Could not initialize an installer for {{.V0}}.", params.Namespace.String())
	}

	return runSteps(targetPath, params.Force, params.UserScope, params.Namespace, d.step, runtime, installer, d.output, d.subshell)
}

func (d *Deploy) createRuntimeInstaller(namespace project.Namespaced, targetPath string) (*runtime.Runtime, installable, error) {
	commitID := namespace.CommitID
	if commitID == nil {
		branch, fail := d.DefaultBranchForProjectName(namespace.Owner, namespace.Project)
		if fail != nil {
			return nil, nil, errs.Wrap(fail, "Could not create installer")
		}

		if branch.CommitID == nil {
			return nil, nil, locale.NewInputError(
				"err_deploy_no_commits",
				"The project '{{.V0}}' does not have any packages configured, please add add some packages first.", namespace.String())
		}

		commitID = branch.CommitID
	}

	runtime := runtime.NewRuntime(*commitID, namespace.Owner, namespace.Project, runbits.NewRuntimeMessageHandler(d.output))
	runtime.SetInstallPath(targetPath)
	return runtime, d.NewRuntimeInstaller(runtime), nil
}

func runSteps(targetPath string, force bool, userScope bool, namespace project.Namespaced, step Step, runtime *runtime.Runtime, installer installable, out output.Outputer, subshell subshell.SubShell) error {
	return runStepsWithFuncs(
		targetPath, force, userScope, namespace, step, runtime, installer, out, subshell,
		install, configure, symlink, report)
}

func runStepsWithFuncs(targetPath string, force, userScope bool, namespace project.Namespaced, step Step, rt *runtime.Runtime, installer installable, out output.Outputer, subshell subshell.SubShell, installf installFunc, configuref configureFunc, symlinkf symlinkFunc, reportf reportFunc) error {
	logging.Debug("runSteps: %s", step.String())

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
		if err := installf(targetPath, installer, out); err != nil {
			return err
		}
	}
	if step == UnsetStep || step == ConfigureStep {
		logging.Debug("Running configure step")
		if err := configuref(targetPath, rt, out, subshell, namespace, userScope); err != nil {
			return err
		}
	}
	if step == UnsetStep || step == SymlinkStep {
		logging.Debug("Running symlink step")
		if err := symlinkf(targetPath, force, rt, out); err != nil {
			return err
		}
	}
	if step == UnsetStep || step == ReportStep {
		logging.Debug("Running report step")
		if err := reportf(targetPath, rt, out); err != nil {
			return err
		}
	}

	return nil
}

type installFunc func(path string, installer installable, out output.Outputer) error

func install(path string, installer installable, out output.Outputer) error {
	out.Notice(output.Heading(locale.T("deploy_install")))
	_, installed, fail := installer.Install()
	if fail != nil {
		return locale.WrapError(fail, "deploy_install_failed", "Installation failed.")
	}
	if !installed {
		out.Notice(locale.T("using_cached_env"))
	}

	if rt.GOOS == "windows" {
		box := packr.NewBox("../../../assets/scripts")
		contents := box.Bytes("setenv.bat")
		fail = fileutils.WriteFile(filepath.Join(path, "setenv.bat"), contents)
		if fail != nil {
			return locale.WrapError(fail.ToError(), "err_deploy_write_setenv", "Could not create setenv batch scriptfile at path: %s", path)
		}
	}

	out.Print(locale.Tl("deploy_install_done", "Installation completed"))
	return nil
}

type configureFunc func(installpath string, runtime *runtime.Runtime, out output.Outputer, sshell subshell.SubShell, namespace project.Namespaced, userScope bool) error

func configure(installpath string, runtime *runtime.Runtime, out output.Outputer, sshell subshell.SubShell, namespace project.Namespaced, userScope bool) error {
	venv := virtualenvironment.New(runtime)
	env, err := venv.GetEnv(false, "")
	if err != nil {
		return err
	}

	out.Notice(output.Heading(locale.Tr("deploy_configure_shell", sshell.Shell())))

	fail := sshell.WriteUserEnv(env, sscommon.Deploy, userScope)
	if fail != nil {
		return locale.WrapError(fail, "err_deploy_subshell_write", "Could not write environment information to your shell configuration.")
	}

	binPath := filepath.Join(installpath, "bin")
	if fail := fileutils.MkdirUnlessExists(binPath); fail != nil {
		return locale.WrapError(fail.ToError(), "err_deploy_binpath", "Could not create bin directory.")
	}

	// Write global env file
	out.Notice(fmt.Sprintf("Writing shell env file to %s\n", filepath.Join(installpath, "bin")))
	err = sshell.SetupShellRcFile(binPath, env, namespace)
	if err != nil {
		return locale.WrapError(err, "err_deploy_subshell_rc_file", "Could not create environment script.")
	}

	return nil
}

type symlinkFunc func(installPath string, overwrite bool, runtime *runtime.Runtime, out output.Outputer) error

func symlink(installPath string, overwrite bool, runtime *runtime.Runtime, out output.Outputer) error {
	venv := virtualenvironment.New(runtime)
	env, err := venv.GetEnv(false, "")
	if err != nil {
		return err
	}

	var path string
	if rt.GOOS != "windows" {
		// Retrieve path to write symlinks to
		path, err = usablePath()
		if err != nil {
			return locale.WrapError(err, "err_usablepath", "Could not retrieve a usable PATH")
		}
	}

	// Retrieve artifact binary directory
	var bins []string
	if p, ok := env["PATH"]; ok {
		bins = strings.Split(p, string(os.PathListSeparator))
	}

	exes, err := exeutils.Executables(bins)
	if err != nil {
		return locale.WrapError(err, "err_symlink_exes", "Could not detect executables")
	}

	// Remove duplicate executables as per PATH and PATHEXT
	exes, err = exeutils.UniqueExes(exes, os.Getenv("PATHEXT"))
	if err != nil {
		return locale.WrapError(err, "err_unique_exes", "Could not detect unique executables, make sure your PATH and PATHEXT environment variables are properly configured.")
	}

	if rt.GOOS != "windows" {
		// Symlink to PATH (eg. /usr/local/bin)
		if err := symlinkWithTarget(overwrite, path, exes, out); err != nil {
			return locale.WrapError(err, "err_symlink", "Could not create symlinks to {{.V0}}.", path)
		}
	}

	return nil
}

// SymlinkTargetPath adds the .lnk file ending on windows
func symlinkTargetPath(targetDir string, path string) string {
	target := filepath.Clean(filepath.Join(targetDir, filepath.Base(path)))
	if rt.GOOS != "windows" {
		return target
	}

	oldExt := filepath.Ext(target)
	return target[0:len(target)-len(oldExt)] + ".lnk"
}

// symlinkWithTarget creates symlinks in the target path of all executables found in the bins dir
// It overwrites existing files, if the overwrite flag is set.
// On Windows the same executable name can have several file extensions,
// therefore executables are only symlinked if it has not been symlinked to a
// target (with the same or a different extension) from a different directory.
// Also: Only the executable with the highest priority according to pathExt is symlinked.
func symlinkWithTarget(overwrite bool, symlinkPath string, exePaths []string, out output.Outputer) error {
	out.Notice(output.Heading(locale.Tr("deploy_symlink", symlinkPath)))

	if fail := fileutils.MkdirUnlessExists(symlinkPath); fail != nil {
		return locale.WrapInputError(
			fail, "err_deploy_mkdir",
			"Could not create directory at {{.V0}}, make sure you have permissions to write to {{.V1}}.", symlinkPath, filepath.Dir(symlinkPath))
	}

	for _, exePath := range exePaths {
		symlink := symlinkTargetPath(symlinkPath, exePath)

		// If the link already exists we may have to overwrite it, skip it, or fail..
		if fileutils.TargetExists(symlink) {
			// If the existing symlink already matches the one we want to create, skip it
			skip, err := shouldSkipSymlink(symlink, exePath)
			if err != nil {
				return locale.WrapError(err, "err_deploy_shouldskip", "Could not determine if link already exists.")
			}
			if skip {
				continue
			}

			// If we're trying to overwrite a link not owned by us but overwrite=false then we should fail
			if !overwrite {
				return locale.NewInputError(
					"err_deploy_symlink_target_exists",
					"Cannot create symlink as the target already exists: {{.V0}}. Use '--force' to overwrite any existing files.", symlink)
			}

			// We're about to overwrite, so if this link isn't owned by us we should let the user know
			out.Notice(locale.Tr("deploy_overwrite_target", symlink))

			// to overwrite the existing file, we have to remove it first, or the link command will fail
			if err := os.Remove(symlink); err != nil {
				return locale.WrapInputError(
					err, "err_deploy_overwrite",
					"Could not overwrite {{.V0}}, make sure you have permissions to write to this file.", symlink)
			}
		}

		if err := link(exePath, symlink); err != nil {
			return err
		}
	}

	return nil
}

type exeFile struct {
	fpath string
	name  string
	ext   string
}

type Report struct {
	BinaryDirectories []string
	Environment       map[string]string
}

type reportFunc func(path string, runtime *runtime.Runtime, out output.Outputer) error

func report(path string, runtime *runtime.Runtime, out output.Outputer) error {
	venv := virtualenvironment.New(runtime)
	env, err := venv.GetEnv(false, "")
	if err != nil {
		return err
	}

	var bins []string
	if path, ok := env["PATH"]; ok {
		delete(env, "PATH")
		bins = strings.Split(path, string(os.PathListSeparator))
	}

	out.Notice(output.Heading(locale.T("deploy_info")))

	out.Print(Report{
		BinaryDirectories: bins,
		Environment:       env,
	})

	if rt.GOOS == "windows" {
		out.Notice(locale.Tr("deploy_restart_cmd", filepath.Join(path, "setenv.bat")))
	} else {
		out.Notice(locale.T("deploy_restart_shell"))
	}

	return nil
}
