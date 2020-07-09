package clean

import (
	"errors"
	"os"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
)

type Cache struct {
	output  output.Outputer
	confirm confirmAble
	path    string
}

type CacheParams struct {
	Force bool
}

func NewCache(prime primeable) *Cache {
	return newCache(prime.Output(), prime.Prompt())
}

func newCache(output output.Outputer, confirm confirmAble) *Cache {
	return &Cache{
		output:  output,
		confirm: confirm,
		path:    config.CachePath(),
	}
}

func (c *Cache) Run(params *CacheParams) error {
	if os.Getenv(constants.ActivatedStateEnvVarName) != "" {
		return errors.New(locale.T("err_clean_cache_activated"))
	}

	if !params.Force {
		ok, fail := c.confirm.Confirm(locale.T("clean_cache_confirm"), false)
		if fail != nil {
			return fail.ToError()
		}
		if !ok {
			return nil
		}
	}

	logging.Debug("Removing cache path: %s", c.path)
	return removeCache(c.path)
}
