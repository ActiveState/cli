package clean

import (
	"errors"
	"os"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
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

	if params.Project != "" {
		return c.removeProject(params.Project, params.Force)
	}

	return c.removeCache(c.path, params.Force)
}

func (c *Cache) removeCache(path string, force bool) error {
	if !force {
		ok, fail := c.confirm.Confirm(locale.T("confirm"), locale.T("clean_cache_confirm"), false)
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
		ok, fail := c.confirm.Confirm(locale.T("confirm"), locale.Tr("clean_cache_artifact_confirm", namespace), false)
		if fail != nil {
			return fail.ToError()
		}
		if !ok {
			return nil
		}
	}

	parsed, fail := project.ParseNamespace(namespace)
	if fail != nil {
		return locale.WrapError(fail.ToError(), "err_clean_cache_invalid_namespace", "Namespace argument is not of the correct format")
	}

	runtime := runtime.NewRuntime("", parsed.Owner, parsed.Project, nil)
	projectInstallPath := runtime.InstallPath()

	logging.Debug("Remove project path: %s", projectInstallPath)
	err := os.RemoveAll(projectInstallPath)
	if err != nil {
		return locale.WrapError(err, "err_clean_remove_artifact", "Could not remove cached runtime environment for project: {{.V0}}", namespace)
	}

	return nil
}
