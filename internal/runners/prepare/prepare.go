package prepare

import (
	"fmt"
	"runtime"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/globaldefault"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/pkg/platform/model"
	rt "github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
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
	if defaultProjectDir == "" {
		return nil
	}

	logging.Debug("Reset default project at %s", defaultProjectDir)
	defaultTargetDir := target.ProjectDirToTargetDir(defaultProjectDir, storage.CachePath())

	proj, err := project.FromPath(defaultProjectDir)
	if err != nil {
		return errs.Wrap(err, "Could not get project from default project directory")
	}

	run, err := rt.New(target.NewCustomTarget(proj.Owner(), proj.Name(), proj.CommitUUID(), defaultTargetDir, target.TriggerResetExec, proj.IsHeadless()), r.analytics, r.svcModel)
	if err != nil {
		return errs.Wrap(err, "Could not initialize runtime for global default project.")
	}

	if err := globaldefault.SetupDefaultActivation(r.subshell, r.cfg, run, proj); err != nil {
		return errs.Wrap(err, "Failed to rewrite the default executors.")
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
		if !errs.Matches(err, &ErrorNotSupported{}) {
			r.reportError(locale.Tl("err_prepare_completions", "Could not generate completions script, error received: {{.V0}}.", err.Error()), err)
		}
	}

	logging.Debug("Reset global executors")
	if err := r.resetExecutors(); err != nil {
		r.reportError(locale.Tl("err_reset_executor", "Could not reset global executors, error received: {{.V0}}", errs.JoinMessage(err)), err)
	}

	// OS specific preparations
	err := r.prepareOS()
	if err != nil {
		return errs.Wrap(err, "Could not prepare OS")
	}

	if err := updateConfigKey(r.cfg, oldGlobalDefaultPrefname, constants.GlobalDefaultPrefname); err != nil {
		r.reportError(locale.Tl("err_prepare_config", "Could not update stale config keys, error recieved: {{.V0}}", errs.JoinMessage(err)), err)
	}

	return nil
}

func (r *Prepare) reportError(message string, err error) {
	multilog.Error("prepare error, message: %s, error: %v", message, errs.Join(err, ": "))
	r.out.Notice(output.Heading(locale.Tl("warning", "Warning")))
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
