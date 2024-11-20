package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/pkg/project"
)

type configurable interface {
	project.ConfigAble
	ConfigPath() string
	GetInt(string) int
	Set(string, interface{}) error
	GetStringMap(string) map[string]interface{}
	GetBool(string) bool
	GetString(string) string
}

type Config struct {
	output  output.Outputer
	confirm prompt.Prompter
	cfg     configurable
	ipComm  svcctl.IPCommunicator
}

type ConfigParams struct {
	Force bool
}

func NewConfig(prime primeable) *Config {
	return newConfig(prime.Output(), prime.Prompt(), prime.Config(), prime.IPComm())
}

func newConfig(out output.Outputer, confirm prompt.Prompter, cfg configurable, ipComm svcctl.IPCommunicator) *Config {
	return &Config{
		output:  out,
		confirm: confirm,
		cfg:     cfg,
		ipComm:  ipComm,
	}
}

func (c *Config) Run(params *ConfigParams) error {
	if os.Getenv(constants.ActivatedStateEnvVarName) != "" {
		return locale.NewError("err_clean_cache_activated")
	}

	defaultChoice := !c.confirm.IsInteractive()
	ok, err := c.confirm.Confirm(locale.T("confirm"), locale.T("clean_config_confirm"), &defaultChoice, ptr.To(true))
	if err != nil {
		return errs.Wrap(err, "Not confirmed")
	}
	if !ok {
		return locale.NewInputError("err_clean_config_aborted", "Cleaning of config aborted by user")
	}

	if err := stopServices(c.cfg, c.output, c.ipComm, params.Force); err != nil {
		return errs.Wrap(err, "Failed to stop services.")
	}

	dir := c.cfg.ConfigPath()
	c.cfg.Close()

	logging.Debug("Removing config directory: %s", dir)
	return removeConfig(dir, c.output)
}
