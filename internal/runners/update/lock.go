package update

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
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
}

func NewLock(prime primeable) *Lock {
	return &Lock{}
}

func (l *Lock) Run(params *LockParams) error {
	return locale.NewInputError("locking_unsupported", "This version of the State Tool does not support version locking anymore.")
}
