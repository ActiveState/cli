package prepare

import (
	"fmt"
	"runtime"

	analytics2 "github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/globaldefault"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/subshell"
	rt "github.com/ActiveState/cli/pkg/platform/runtime"
)

type primeable interface {
	primer.Outputer
	primer.Subsheller
	primer.Configurer
	primer.Analyticer
}

// Prepare manages the prepare execution context.
type Prepare struct {
	out       output.Outputer
	subshell  subshell.SubShell
	cfg       *config.Instance
	analytics analytics2.Dispatcher
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
	defaultTargetDir := rt.ProjectDirToTargetDir(defaultProjectDir, storage.CachePath())
	run, err := rt.New(rt.NewCustomTarget("", "", "", defaultTargetDir), r.analytics)
	if err != nil {
		return errs.Wrap(err, "Could not initialize runtime for global default project.")
	}

	if err := globaldefault.SetupDefaultActivation(r.subshell, r.cfg, run, defaultProjectDir); err != nil {
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
	r.prepareOS()

	return nil
}

func (r *Prepare) reportError(message string, err error) {
	logging.Error("prepare error, message: %s, error: %v", message, errs.Join(err, ": "))
	r.out.Notice(output.Heading(locale.Tl("warning", "Warning")))
	r.out.Notice(message)
}
