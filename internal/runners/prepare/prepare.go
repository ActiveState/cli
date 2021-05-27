package prepare

import (
	"fmt"
	"runtime"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/globaldefault"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils/autostart"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/subshell"
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
			r.reportError(locale.Tr("err_prepare_completions", "Could not generate completions script, error received: {{.V0}}.", err.Error()), err)
		}
	}

	if err := prepareCompletions(cmd, r.subshell); err != nil {
		if !errs.Matches(err, &ErrorNotSupported{}) {
			r.reportError(locale.Tr("err_prepare_completions", "Could not generate completions script, error received: {{.V0}}.", err.Error()), err)
		}
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