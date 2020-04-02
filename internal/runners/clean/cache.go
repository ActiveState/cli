package clean

import (
	"errors"
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
)

type Cache struct {
	output  output.Outputer
	confirm confirmAble
}

type CacheParams struct {
	Path  string
	Force bool
}

func NewCache(out output.Outputer, confirmer confirmAble) *Cache {
	return &Cache{
		output:  out,
		confirm: confirmer,
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

	logging.Debug("Removing cache path: %s", params.Path)
	return removeCache(params.Path)
}
