package sscommon

import "github.com/ActiveState/cli/internal/constants"

type EnvType int

const (
	Deploy EnvType = iota
	Default
)

type envData struct {
	start string
	stop  string
}

var lookup = [...]envData{
	{
		constants.RCAppendDeployStartLine,
		constants.RCAppendDeployStopLine,
	},
	{
		constants.RCAppendDefaultStartLine,
		constants.RCAppendDefaultStopLine,
	},
}

func (e EnvType) data() envData {
	i := int(e)
	if i < 0 || i > len(lookup)-1 {
		i = 1
	}
	return lookup[i]
}

func (e EnvType) start() string {
	return e.data().start
}

func (e EnvType) stop() string {
	return e.data().stop
}
