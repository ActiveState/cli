package autostart

import (
	"github.com/ActiveState/cli/internal/constants"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
)

func init() {
	configMediator.RegisterOption(constants.AutostartSvcConfigKey, configMediator.Bool, configMediator.EmptyEvent, configMediator.EmptyEvent)
}

type Configurable interface {
	Set(string, interface{}) error
	IsSet(string) bool
}

type Options struct {
	Name           string
	Args           []string
	LaunchFileName string
	IconFileName   string
	IconFileSource string
	GenericName    string
	Comment        string
	Keywords       string
	MacLabel       string // macOS plist Label
	MacInteractive bool   // macOS plist Interactive ProcessType
}

func Enable(exec string, opts Options) error {
	return enable(exec, opts)
}

func Disable(exec string, opts Options) error {
	return disable(exec, opts)
}

func IsEnabled(exec string, opts Options) (bool, error) {
	return isEnabled(exec, opts)
}
