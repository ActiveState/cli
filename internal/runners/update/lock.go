package update

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/project"
)

var _ captain.FlagMarshaler = &StateToolChannelVersion{}

type StateToolChannelVersion struct {
	captain.NameVersion
}

func (stv *StateToolChannelVersion) Set(arg string) error {
	err := stv.NameVersion.Set(arg)
	if err != nil {
		return locale.WrapInputError(err, "err_channel_format", "The State Tool channel and version provided is not formatting correctly, must be in the form of <channel>@<version>")
	}
	return nil
}

func (stv *StateToolChannelVersion) Type() string {
	return "channel"
}

type LockParams struct {
	Channel StateToolChannelVersion
	Force   bool
}

type Lock struct {
	project *project.Project
	out     output.Outputer
	prompt  prompt.Prompter
}

func NewLock(prime primeable) *Lock {
	return &Lock{
		prime.Project(),
		prime.Output(),
		prime.Prompt(),
	}
}

func (l *Lock) Run(params *LockParams) error {
	return locale.NewInputError("locking_unsupported", "This version of the State Tool does not support version locking anymore.")
}
