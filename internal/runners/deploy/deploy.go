package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/gobuffalo/packr"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
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
	cfg      *config.Instance
}

type primeable interface {
	primer.Outputer
	primer.Subsheller
	primer.Configurer
}

func NewDeploy(step Step, prime primeable) *Deploy {
	return &Deploy{
		prime.Output(),
		prime.Subshell(),
		step,
		prime.Config(),
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

	commitID, err := d.commitID(params.Namespace)
	if err != nil {
		return locale.WrapError(err, "err_deploy_commitid", "Could not grab commit ID for project: {{.V0}}.", params.Namespace.String())
	}

	rtTarget := runtime.NewCustomTarget(params.Namespace.Owner, params.Namespace.Project, commitID, params.Path) /* TODO: handle empty path */

	logging.Debug("runSteps: %s", d.step.String())

	if d.step == UnsetStep || d.step == InstallStep {
		logging.Debug("Running install step")
		if err := d.install(rtTarget); err != nil {
			return err
		}
	}
	if d.step == UnsetStep || d.step == ConfigureStep {
		logging.Debug("Running configure step")
		if err := d.configure(params.Namespace, rtTarget, params.UserScope); err != nil {
			return err
		}
	}
	if d.step == UnsetStep || d.step == SymlinkStep {
		logging.Debug("Running symlink step")
		if err := d.symlink(rtTarget, params.Force); err != nil {
			return err
		}
	}
	if d.step == UnsetStep || d.step == ReportStep {
		logging.Debug("Running report step")
		if err := d.report(rtTarget); err != nil {
			return err
		}
	}

	return nil
}

func (d *Deploy) commitID(namespace project.Namespaced) (strfmt.UUID, error) {
	commitID := namespace.CommitID
	if commitID == nil {
		branch, err := model.DefaultBranchForProjectName(namespace.Owner, namespace.Project)
		if err != nil {
			return "", errs.Wrap(err, "Could not detect default branch")
		}

		if branch.CommitID == nil {
			return "", locale.NewInputError(
				"err_deploy_no_commits",
				"The project '{{.V0}}' does not have any packages configured, please add add some packages first.", namespace.String())
		}

		commitID = branch.CommitID
	}

	if commitID == nil {
		return "", errs.New("commitID is nil")
	}

	return *commitID, nil
}

func (d *Deploy) install(rtTarget setup.Targeter) error {
	d.output.Notice(output.Heading(locale.T("deploy_install")))

	rti, err := runtime.New(rtTarget)
	if err == nil {
		d.output.Notice(locale.Tl("deploy_already_installed", "Already installed"))
		return nil
	}
	if !runtime.IsNeedsUpdateError(err) {
		return locale.WrapError(err, "deploy_runtime_err", "Could not initialize runtime")
	}

	if err := rti.Update(runbits.DefaultRuntimeEventHandler(d.output)); err != nil {
		return locale.WrapError(err, "deploy_install_failed", "Installation failed.")
	}

	if rt.GOOS == "windows" {
		box := packr.NewBox("../../../assets/scripts")
		contents := box.Bytes("setenv.bat")
		err := fileutils.WriteFile(filepath.Join(rtTarget.Dir(), "setenv.bat"), contents)
		if err != nil {
			return locale.WrapError(err, "err_deploy_write_setenv", "Could not create setenv batch scriptfile at path: %s", rtTarget.Dir())
		}
	}

	d.output.Print(locale.Tl("deploy_install_done", "Installation completed"))
	return nil
}

func (d *Deploy) configure(namespace project.Namespaced, rtTarget setup.Targeter, userScope bool) error {
	rti, err := runtime.New(rtTarget)
	if err != nil {
		if runtime.IsNeedsUpdateError(err) {
			return locale.NewInputError("err_deploy_run_install")
		}
		return locale.WrapError(err, "deploy_runtime_err", "Could not initialize runtime")
	}

	env, err := rti.Env(false, false)
	if err != nil {
		return err
	}

	d.output.Notice(output.Heading(locale.Tr("deploy_configure_shell", d.subshell.Shell())))

	err = d.subshell.WriteUserEnv(d.cfg, env, sscommon.DeployID, userScope)
	if err != nil {
		return locale.WrapError(err, "err_deploy_subshell_write", "Could not write environment information to your shell configuration.")
	}

	binPath := filepath.Join(rtTarget.Dir(), "bin")
	if err := fileutils.MkdirUnlessExists(binPath); err != nil {
		return locale.WrapError(err, "err_deploy_binpath", "Could not create bin directory.")
	}

	// Write global env file
	d.output.Notice(fmt.Sprintf("Writing shell env file to %s\n", filepath.Join(rtTarget.Dir(), "bin")))
	err = d.subshell.SetupShellRcFile(binPath, env, namespace)
	if err != nil {
		return locale.WrapError(err, "err_deploy_subshell_rc_file", "Could not create environment script.")
	}

	return nil
}

func (d *Deploy) symlink(rtTarget setup.Targeter, overwrite bool) error {
	rti, err := runtime.New(rtTarget)
	if err != nil {
		if runtime.IsNeedsUpdateError(err) {
			return locale.NewInputError("err_deploy_run_install")
		}
		return locale.WrapError(err, "deploy_runtime_err", "Could not initialize runtime")
	}

	var path string
	if rt.GOOS != "windows" {
		// Retrieve path to write symlinks to
		path, err = usablePath()
		if err != nil {
			return locale.WrapError(err, "err_usablepath", "Could not retrieve a usable PATH")
		}
	}

	// Retrieve artifact binary directories
	bins, err := rti.ExecutablePaths()
	if err != nil {
		return locale.WrapError(err, "err_symlink_exes", "Could not detect executable paths")
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
		if err := symlinkWithTarget(overwrite, path, exes, d.output); err != nil {
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

	if err := fileutils.MkdirUnlessExists(symlinkPath); err != nil {
		return locale.WrapInputError(
			err, "err_deploy_mkdir",
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

type Report struct {
	BinaryDirectories []string
	Environment       map[string]string
}

func (d *Deploy) report(rtTarget setup.Targeter) error {
	rti, err := runtime.New(rtTarget)
	if err != nil {
		if runtime.IsNeedsUpdateError(err) {
			return locale.NewInputError("err_deploy_run_install")
		}
		return locale.WrapError(err, "deploy_runtime_err", "Could not initialize runtime")
	}

	env, err := rti.Env(false, false)
	if err != nil {
		return err
	}

	var bins []string
	if path, ok := env["PATH"]; ok {
		delete(env, "PATH")
		bins = strings.Split(path, string(os.PathListSeparator))
	}

	d.output.Notice(output.Heading(locale.T("deploy_info")))

	d.output.Print(Report{
		BinaryDirectories: bins,
		Environment:       env,
	})

	d.output.Notice(output.Heading(locale.T("deploy_restart")))

	if rt.GOOS == "windows" {
		d.output.Notice(locale.Tr("deploy_restart_cmd", filepath.Join(rtTarget.Dir(), "setenv.bat")))
	} else {
		d.output.Notice(locale.T("deploy_restart_shell"))
	}

	return nil
}
