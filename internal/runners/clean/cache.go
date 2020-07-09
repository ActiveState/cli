package clean

import (
	"errors"
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/runtime"
)

type Cache struct {
	output  output.Outputer
	confirm confirmAble
	path    string
}

type CacheParams struct {
	Force   bool
	Project string
}

func NewCache(out output.Outputer, confirmer confirmAble) *Cache {
	return &Cache{
		output:  out,
		confirm: confirmer,
		path:    config.CachePath(),
	}
}

func (c *Cache) Run(params *CacheParams) error {
	if os.Getenv(constants.ActivatedStateEnvVarName) != "" {
		return errors.New(locale.T("err_clean_cache_activated"))
	}

	if params.Project != "" {
		return c.removeProject(params.Project, params.Force)
	}

	return c.removeCache(c.path, params.Force)
}

func (c *Cache) removeCache(path string, force bool) error {
	if !force {
		ok, fail := c.confirm.Confirm(locale.T("clean_cache_confirm"), false)
		if fail != nil {
			return fail.ToError()
		}
		if !ok {
			return nil
		}
	}

	logging.Debug("Removing cache path: %s", path)
	return removeCache(c.path)
}

func (c *Cache) removeProject(namespace string, force bool) error {
	if !force {
		ok, fail := c.confirm.Confirm(locale.Tr("clean_cache_artifact_confirm", namespace), false)
		if fail != nil {
			return fail.ToError()
		}
		if !ok {
			return nil
		}
	}

	split := strings.Split(namespace, "/")
	if len(split) != 2 {
		return locale.NewInputError("err_clean_cache_invalid_namespace", "Namespace argument is not in the correct format of <Owner>/<ProjectName>")
	}
	projectInstallPath := runtime.InstallPath(split[0], split[1])

	logging.Debug("Remove project path: %s", projectInstallPath)
	return os.RemoveAll(projectInstallPath)
}
