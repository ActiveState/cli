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
	key   string
}

var lookup = [...]envData{
	{
		constants.RCAppendDeployStartLine,
		constants.RCAppendDeployStopLine,
		"user_deploy_env",
	},
	{
		constants.RCAppendDefaultStartLine,
		constants.RCAppendDefaultStopLine,
		"user_default_env",
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

func (e EnvType) ConfigKey() string {
	return e.data().key
}
