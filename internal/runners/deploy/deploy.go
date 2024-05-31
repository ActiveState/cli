package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/runbits/checkout"
	runtime_runbit "github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/runbits/runtime/progress"
	"github.com/ActiveState/cli/pkg/runtime/helpers"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
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
	prime primeable
	// The remainder is redundant with the above. Refactoring this will follow in a later story so as not to blow
	// up the one that necessitates adding the primer at this level.
	// https://activestatef.atlassian.net/browse/DX-2869
	auth      *authentication.Auth
	output    output.Outputer
	subshell  subshell.SubShell
	step      Step
	cfg       *config.Instance
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
}

type primeable interface {
	primer.Auther
	primer.Outputer
	primer.Subsheller
	primer.Configurer
	primer.Analyticer
	primer.SvcModeler
	primer.Projecter
}

func NewDeploy(step Step, prime primeable) *Deploy {
	return &Deploy{
		prime,
		prime.Auth(),
		prime.Output(),
		prime.Subshell(),
		step,
		prime.Config(),
		prime.Analytics(),
		prime.SvcModel(),
	}
}

func (d *Deploy) Run(params *Params) error {
	if RequiresAdministratorRights(d.step, params.UserScope) {
		isAdmin, err := osutils.IsAdmin()
		if err != nil {
			multilog.Error("Could not check for windows administrator privileges: %v", err)
		}
		if !isAdmin {
			return locale.NewError("err_deploy_admin_privileges_required", "Administrator rights are required for this command to modify the system PATH.  If you want to deploy to the user environment, please adjust the command line flags.")
		}
	}

	commitID, err := d.commitID(params.Namespace)
	if err != nil {
		return locale.WrapError(err, "err_deploy_commitid", "Could not grab commit ID for project: {{.V0}}.", params.Namespace.String())
	}

	logging.Debug("runSteps: %s", d.step.String())

	if d.step == UnsetStep || d.step == InstallStep {
		logging.Debug("Running install step")
		if err := d.install(params, commitID); err != nil {
			return err
		}
	}
	if d.step == UnsetStep || d.step == ConfigureStep {
		logging.Debug("Running configure step")
		if err := d.configure(params); err != nil {
			return err
		}
	}
	if d.step == UnsetStep || d.step == SymlinkStep {
		logging.Debug("Running symlink step")
		if err := d.symlink(params); err != nil {
			return err
		}
	}
	if d.step == UnsetStep || d.step == ReportStep {
		logging.Debug("Running report step")
		if err := d.report(params); err != nil {
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

func (d *Deploy) install(params *Params, commitID strfmt.UUID) (rerr error) {
	d.output.Notice(output.Title(locale.T("deploy_install")))

	if err := checkout.CreateProjectFiles(
		params.Path, params.Path, params.Namespace.Owner, params.Namespace.Project,
		constants.DefaultBranchName, commitID.String(), "",
	); err != nil {
		return errs.Wrap(err, "Could not create project files")
	}

	proj, err := project.FromPath(params.Path)
	if err != nil {
		return locale.WrapError(err, "err_project_frompath")
	}
	d.prime.SetProject(proj)

	pg := progress.NewRuntimeProgressIndicator(d.output)
	defer rtutils.Closer(pg.Close, &rerr)

	if _, err := runtime_runbit.Update(d.prime, runtime_runbit.TriggerDeploy, runtime_runbit.WithTargetDir(params.Path)); err != nil {
		return locale.WrapError(err, "err_deploy_runtime_err", "Could not initialize runtime")
	}

	if rt.GOOS == "windows" {
		contents, err := assets.ReadFileBytes("scripts/setenv.bat")
		if err != nil {
			return err
		}
		err = fileutils.WriteFile(filepath.Join(params.Path, "setenv.bat"), contents)
		if err != nil {
			return locale.WrapError(err, "err_deploy_write_setenv", "Could not create setenv batch scriptfile at path: %s", params.Path)
		}
	}

	d.output.Print(locale.Tl("deploy_install_done", "Installation completed"))
	return nil
}

func (d *Deploy) configure(params *Params) error {
	proj, err := project.FromPath(params.Path)
	if err != nil {
		return locale.WrapInputError(err, "err_deploy_run_install")
	}

	rti, err := runtime_helpers.FromProject(proj)
	if err != nil {
		return locale.WrapError(err, "deploy_runtime_err", "Could not initialize runtime")
	}

	if !rti.HasCache() {
		return locale.NewInputError("err_deploy_run_install")
	}

	d.output.Notice(output.Title(locale.Tr("deploy_configure_shell", d.subshell.Shell())))

	env := rti.Env().Variables

	// Configure available shells
	err = subshell.ConfigureAvailableShells(d.subshell, d.cfg, env, sscommon.DeployID, params.UserScope)
	if err != nil {
		return locale.WrapError(err, "err_deploy_subshell_write", "Could not write environment information to your shell configuration.")
	}

	binPath := filepath.Join(rti.Path(), "bin")
	if err := fileutils.MkdirUnlessExists(binPath); err != nil {
		return locale.WrapError(err, "err_deploy_binpath", "Could not create bin directory.")
	}

	// Write global env file
	d.output.Notice(fmt.Sprintf("Writing shell env file to %s\n", filepath.Join(rti.Path(), "bin")))
	err = d.subshell.SetupShellRcFile(binPath, env, &params.Namespace, d.cfg)
	if err != nil {
		return locale.WrapError(err, "err_deploy_subshell_rc_file", "Could not create environment script.")
	}

	return nil
}

func (d *Deploy) symlink(params *Params) error {
	proj, err := project.FromPath(params.Path)
	if err != nil {
		return locale.WrapInputError(err, "err_deploy_run_install")
	}

	rti, err := runtime_helpers.FromProject(proj)
	if err != nil {
		return locale.WrapError(err, "deploy_runtime_err", "Could not initialize runtime")
	}

	if !rti.HasCache() {
		return locale.NewInputError("err_deploy_run_install")
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
	bins, err := osutils.ExecutablePaths(rti.Env().Variables)
	if err != nil {
		return locale.WrapError(err, "err_symlink_exes", "Could not detect executable paths")
	}

	exes, err := osutils.Executables(bins)
	if err != nil {
		return locale.WrapError(err, "err_symlink_exes", "Could not detect executables")
	}

	// Remove duplicate executables as per PATH and PATHEXT
	exes, err = osutils.UniqueExes(exes, os.Getenv("PATHEXT"))
	if err != nil {
		return locale.WrapError(err, "err_unique_exes", "Could not detect unique executables, make sure your PATH and PATHEXT environment variables are properly configured.")
	}

	if rt.GOOS != "windows" {
		// Symlink to PATH (eg. /usr/local/bin)
		if err := symlinkWithTarget(params.Force, path, exes, d.output); err != nil {
			return locale.WrapError(err, "err_symlink", "Could not create symlinks to {{.V0}}.", path)
		}
	} else {
		d.output.Notice(locale.Tl("deploy_symlink_skip", "Skipped on Windows"))
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
	out.Notice(output.Title(locale.Tr("deploy_symlink", symlinkPath)))

	if err := fileutils.MkdirUnlessExists(symlinkPath); err != nil {
		return locale.WrapExternalError(
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
				return locale.WrapExternalError(
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

func (d *Deploy) report(params *Params) error {
	proj, err := project.FromPath(params.Path)
	if err != nil {
		return locale.WrapInputError(err, "err_deploy_run_install")
	}

	rti, err := runtime_helpers.FromProject(proj)
	if err != nil {
		return locale.WrapError(err, "deploy_runtime_err", "Could not initialize runtime")
	}

	if !rti.HasCache() {
		return locale.NewInputError("err_deploy_run_install")
	}

	env := rti.Env().Variables

	var bins []string
	if path, ok := env["PATH"]; ok {
		delete(env, "PATH")
		bins = strings.Split(path, string(os.PathListSeparator))
	}

	d.output.Notice(output.Title(locale.T("deploy_info")))

	d.output.Print(Report{
		BinaryDirectories: bins,
		Environment:       env,
	})

	d.output.Notice(output.Title(locale.T("deploy_restart")))

	if rt.GOOS == "windows" {
		d.output.Notice(locale.Tr("deploy_restart_cmd", filepath.Join(params.Path, "setenv.bat")))
	} else {
		d.output.Notice(locale.T("deploy_restart_shell"))
	}

	return nil
}
