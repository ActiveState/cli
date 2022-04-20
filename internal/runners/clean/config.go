package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/pkg/project"
)

type configurable interface {
	project.ConfigAble
	ConfigPath() string
	GetInt(string) int
	Set(string, interface{}) error
	IsSet(string) bool
	GetStringMap(string) map[string]interface{}
	GetBool(string) bool
	GetString(string) string
}

type Config struct {
	output  output.Outputer
	confirm confirmAble
	cfg     configurable
	ipComm  svcctl.IPCommunicator
}

type ConfigParams struct {
	Force bool
}

func NewConfig(prime primeable) *Config {
	return newConfig(prime.Output(), prime.Prompt(), prime.Config(), prime.IPComm())
}

func newConfig(out output.Outputer, confirm confirmAble, cfg configurable, ipComm svcctl.IPCommunicator) *Config {
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

	if !params.Force {
		ok, err := c.confirm.Confirm(locale.T("confirm"), locale.T("clean_config_confirm"), new(bool))
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	if err := stopServices(c.cfg, c.output, c.ipComm, params.Force); err != nil {
		return errs.Wrap(err, "Failed to stop services.")
	}

	dir := c.cfg.ConfigPath()
	c.cfg.Close()

	logging.Debug("Removing config directory: %s", dir)
	return removeConfig(dir, c.output)
}
