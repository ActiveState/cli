package prepare

import (
	"fmt"
	"os"
	"runtime"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/globaldefault"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils/autostart"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/subshell"
	rt "github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type primeable interface {
	primer.Outputer
	primer.Subsheller
	primer.Configurer
}

// Prepare manages the prepare execution context.
type Prepare struct {
	out      output.Outputer
	subshell subshell.SubShell
	cfg      *config.Instance
}

// New prepares a prepare execution context for use.
func New(prime primeable) *Prepare {
	return &Prepare{
		out:      prime.Output(),
		subshell: prime.Subshell(),
		cfg:      prime.Config(),
	}
}

// resetExecutors removes the executor directories for all project installations, and rewrites the global default executors
// This ensures that the installation is compatible with an updated State Tool installation
func (r *Prepare) resetExecutors() error {
	projects := projectfile.GetProjectMapping(r.cfg)
	for _, projectDirs := range projects {
		for _, projectDir := range projectDirs {
			installDir := rt.ProjectDirToTargetDir(projectDir, r.cfg.CachePath())
			logging.Debug("Reset executor for %s", projectDir)
			err := os.RemoveAll(setup.ExecDir(installDir))
			if err != nil {
				logging.Error("Failed to re-set executor directory %s", setup.ExecDir(installDir))
			}
		}
	}

	defaultProjectDir := r.cfg.GetString(constants.GlobalDefaultPrefname)
	if defaultProjectDir == "" {
		return nil
	}

	logging.Debug("Reset default project at %s", defaultProjectDir)
	defaultTargetDir := rt.ProjectDirToTargetDir(defaultProjectDir, r.cfg.CachePath())
	run, err := rt.New(rt.NewCustomTarget("", "", "", defaultTargetDir))
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
			return locale.WrapError(err, msgLocale, globaldefault.BinDir(r.cfg))
		}
		r.reportError(locale.Tr(msgLocale, globaldefault.BinDir(r.cfg)), err)
	}

	if err := prepareCompletions(cmd, r.subshell); err != nil {
		if !errs.Matches(err, &ErrorNotSupported{}) {
			r.reportError(locale.Tl("err_prepare_completions", "Could not generate completions script, error received: {{.V0}}.", err.Error()), err)
		}
	}

	logging.Debug("Reset executors")
	if err := r.resetExecutors(); err != nil {
		r.reportError(locale.Tl("err_reset_executor", "Could not reset project executors, error received: {{.V0}}", errs.JoinMessage(err)), err)
	}

	r.prepareSystray()

	// OS specific preparations
	r.prepareOS()

	return nil
}

func (r *Prepare) reportError(message string, err error) {
	logging.Error("prepare error, message: %s, error: %v", message, errs.Join(err, ": "))
	r.out.Notice(output.Heading(locale.Tl("warning", "Warning")))
	r.out.Notice(message)
}

func (r *Prepare) prepareSystray() {
	trayInfo := appinfo.TrayApp()
	name, exec := trayInfo.Name(), trayInfo.Exec()

	if err := autostart.New(name, exec, r.cfg).EnableFirstTime(); err != nil {
		r.reportError(locale.Tr("err_prepare_autostart", "Could not enable auto-start, error received: {{.V0}}.", err.Error()), err)
	}

	return
}
