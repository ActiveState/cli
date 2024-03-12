package autostart

import (
	"github.com/ActiveState/cli/internal/constants"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
)

func init() {
	configMediator.RegisterOption(configMediator.Option{
		Name:    constants.AutostartSvcConfigKey,
		Type:    configMediator.Bool,
		Default: true,
	})
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

// IsEnabled is provided for testing only.
func IsEnabled(exec string, opts Options) (bool, error) {
	return isEnabled(exec, opts)
}

func AutostartPath(exec string, opts Options) (string, error) {
	return autostartPath(exec, opts)
}

func Upgrade(exec string, opts Options) error {
	return upgrade(exec, opts)
}
