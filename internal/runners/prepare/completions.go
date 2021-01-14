package prepare

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/globaldefault"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/bash"
	"github.com/ActiveState/cli/internal/subshell/fish"
	"github.com/ActiveState/cli/internal/subshell/zsh"
)

type ErrorNotSupported struct{ *locale.LocalizedError }

// Prepare manages the prepare execution context.
type Completions struct {
	out      output.Outputer
	subshell subshell.SubShell
	cfg      globaldefault.DefaultConfigurer
}

// New prepares a prepare execution context for use.
func NewCompletions(prime primeable) *Completions {
	return &Completions{
		out:      prime.Output(),
		subshell: prime.Subshell(),
		cfg:      prime.Config(),
	}
}

// Run executes the prepare behavior.
func (c *Completions) Run(cmd *captain.Command) error {
	if err := prepareCompletions(cmd, c.subshell); err != nil {
		return locale.WrapError(err, "err_prepare_completions", "Could not prepare completions")
	}

	c.out.Notice(locale.Tl("completions_success", "Completions have been written, please reload your shell."))

	return nil
}

func prepareCompletions(cmd *captain.Command, sub subshell.SubShell) error {
	var err error
	var completions string
	shell := sub.Shell()
	switch shell {
	case zsh.Name:
		completions, err = cmd.GenZshCompletion()
	case bash.Name:
		completions, err = cmd.GenBashCompletions()
	case fish.Name:
		completions, err = cmd.GenFishCompletions()
	default:
		return &ErrorNotSupported{
			locale.NewInputError("err_shell_not_supported", "Completions are currently not supported for {{.V0}}.", shell),
		}
	}

	if err != nil {
		return locale.WrapError(err, "err_completions_generate", "Could not generate completions due to error: {{.V0}}.", err.Error())
	}

	if err := sub.WriteCompletionScript(completions); err != nil {
		return locale.WrapError(err, "err_completions_write", "Writing completion data failed")
	}

	return nil
}
