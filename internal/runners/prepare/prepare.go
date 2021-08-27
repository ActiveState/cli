package prepare

import (
	"fmt"
	"os"
	"runtime"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/globaldefault"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/updater"
)

type primeable interface {
	primer.Outputer
	primer.Subsheller
	primer.Configurer
}

type Configurable interface {
	GetString(string) string
	globaldefault.DefaultConfigurer
}

// Prepare manages the prepare execution context.
type Prepare struct {
	out      output.Outputer
	subshell subshell.SubShell
	cfg      Configurable
}

type Params struct {
	SessionToken string
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

	sessionToken := os.Getenv(constants.SessionTokenEnvVarName)
	if sessionToken != "" && r.cfg.GetString(analytics.CfgSessionToken) == "" {
		if err := r.cfg.Set(analytics.CfgSessionToken, sessionToken); err != nil {
			logging.Error("Failed to set session token: %s", errs.Join(err, ": ").Error())
		}
	}

	updateTagValue := os.Getenv(constants.UpdateTagEnvVarName)
	if err := r.cfg.Set(updater.CfgTag, updateTagValue); err != nil {
		logging.Error("Failed to set update tag value: %s", errs.Join(err, ": "))
	}

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

	// OS specific preparations
	return r.prepareOS()
}

func (r *Prepare) reportError(message string, err error) {
	logging.Error("prepare error, message: %s, error: %v", message, errs.Join(err, ": "))
	r.out.Notice(output.Heading(locale.Tl("warning", "Warning")))
	r.out.Notice(message)
}
