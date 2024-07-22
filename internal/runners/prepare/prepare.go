package prepare

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	svcApp "github.com/ActiveState/cli/cmd/state-svc/app"
	svcAutostart "github.com/ActiveState/cli/cmd/state-svc/autostart"
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/globaldefault"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils/autostart"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/runtime_helpers"
	"github.com/thoas/go-funk"
)

const oldGlobalDefaultPrefname = "default_project_path"

type primeable interface {
	primer.Outputer
	primer.Subsheller
	primer.Configurer
	primer.Analyticer
	primer.SvcModeler
}

// Prepare manages the prepare execution context.
type Prepare struct {
	out       output.Outputer
	subshell  subshell.SubShell
	cfg       *config.Instance
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
}

// New prepares a prepare execution context for use.
func New(prime primeable) *Prepare {
	return &Prepare{
		out:       prime.Output(),
		subshell:  prime.Subshell(),
		cfg:       prime.Config(),
		analytics: prime.Analytics(),
	}
}

// resetExecutors removes the executor directories for all project installations, and rewrites the global default executors
// This ensures that the installation is compatible with an updated State Tool installation
func (r *Prepare) resetExecutors() error {
	defaultProjectDir := r.cfg.GetString(constants.GlobalDefaultPrefname)
	if defaultProjectDir == "" || !fileutils.TargetExists(defaultProjectDir) {
		return nil
	}

	logging.Debug("Reset default project at %s", defaultProjectDir)
	proj, err := project.FromPath(defaultProjectDir)
	if err != nil {
		return errs.Wrap(err, "Could not get project from its directory")
	}

	rt, err := runtime_helpers.FromProject(proj)
	if err != nil {
		return errs.Wrap(err, "Could not initialize runtime for project.")
	}

	rtHash, err := runtime_helpers.Hash(proj, nil)
	if err != nil {
		return errs.Wrap(err, "Could not get runtime hash")
	}

	if rtHash == rt.Hash() || !rt.HasCache() {
		return nil
	}

	if err := globaldefault.SetupDefaultActivation(r.subshell, r.cfg, rt, proj); err != nil {
		return errs.Wrap(err, "Failed to rewrite the executors.")
	}

	return nil
}

// Run executes the prepare behavior.
func (r *Prepare) Run(cmd *captain.Command) error {
	logging.Debug("ExecutePrepare")

	if err := globaldefault.Prepare(r.cfg, r.subshell); err != nil {
		msgLocale := fmt.Sprintf("prepare_instructions_%s", runtime.GOOS)
		if runtime.GOOS != "linux" {
			return locale.WrapError(err, msgLocale, globaldefault.BinDir())
		}
		r.reportError(locale.Tr(msgLocale, globaldefault.BinDir()), err)
	}

	if err := prepareCompletions(cmd, r.subshell); err != nil {
		var errNotSupported *ErrorNotSupported
		if !errors.As(err, &errNotSupported) && !os.IsPermission(err) {
			r.reportError(locale.Tl("err_prepare_generate_completions", "Could not generate completions script, error received: {{.V0}}.", err.Error()), err)
		}
	}

	logging.Debug("Reset global executors")
	if err := r.resetExecutors(); err != nil {
		r.reportError(locale.Tl("err_reset_executor", "Could not reset global executors, error received: {{.V0}}", errs.JoinMessage(err)), err)
	}

	// OS specific preparations
	err := r.prepareOS()
	if err != nil {
		if installation.IsStateExeDoesNotExistError(err) && runtime.GOOS == "windows" {
			return locale.WrapExternalError(err, "err_install_state_exe_does_not_exist", "", constants.ForumsURL)
		}
		return errs.Wrap(err, "Could not prepare OS")
	}

	if err := updateConfigKey(r.cfg, oldGlobalDefaultPrefname, constants.GlobalDefaultPrefname); err != nil {
		r.reportError(locale.Tl("err_prepare_config", "Could not update stale config keys, error recieved: {{.V0}}", errs.JoinMessage(err)), err)
	}

	return nil
}

func (r *Prepare) reportError(message string, err error) {
	multilog.Error("prepare error, message: %s, error: %v", message, errs.JoinMessage(err))
	r.out.Notice(output.Title(locale.Tl("warning", "Warning")))
	r.out.Notice(message)
}

func updateConfigKey(cfg *config.Instance, oldKey, newKey string) error {
	if !funk.Contains(cfg.AllKeys(), oldKey) {
		return nil
	}

	value := cfg.Get(oldKey)
	err := cfg.Set(oldKey, "")
	if err != nil {
		return errs.Wrap(err, "Could not clear old global default prefname")
	}

	if cfg.Get(newKey) != nil {
		return nil
	}

	err = cfg.Set(newKey, value)
	if err != nil {
		return errs.Wrap(err, "Could not set new config key")
	}

	return nil
}

// InstalledPreparedFiles returns the files installed by state _prepare
func InstalledPreparedFiles() ([]string, error) {
	var files []string
	svcApp, err := svcApp.New()
	if err != nil {
		return nil, locale.WrapError(err, "err_autostart_app")
	}

	path, err := autostart.AutostartPath(svcApp.Path(), svcAutostart.Options)
	if err != nil {
		multilog.Error("Failed to determine shortcut path for removal: %v", err)
	} else if path != "" {
		files = append(files, path)
	}

	return files, nil
}

// CleanOS performs any OS-specific cleanup that is needed other than deleting installed files.
func CleanOS() error {
	return cleanOS()
}
