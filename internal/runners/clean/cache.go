package clean

import (
	"errors"
	"os"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/spf13/viper"
)

type Cache struct {
	output  output.Outputer
	config  runbits.ConfigAble
	confirm confirmAble
	path    string
}

type CacheParams struct {
	Force   bool
	Project string
}

func NewCache(prime primeable) *Cache {
	return newCache(prime.Output(), viper.GetViper(), prime.Prompt())
}

func newCache(output output.Outputer, cfg runbits.ConfigAble, confirm confirmAble) *Cache {
	return &Cache{
		output:  output,
		config:  cfg,
		confirm: confirm,
		path:    config.CachePath(),
	}
}

func (c *Cache) Run(params *CacheParams) error {
	if os.Getenv(constants.ActivatedStateEnvVarName) != "" {
		return errors.New(locale.T("err_clean_cache_activated"))
	}

	if params.Project != "" {
		paths := runbits.AvailableProjectPaths(c.config, params.Project)

		// TODO: Info-box or question on whether we should remove
		for _, projectPath := range paths {
			err := c.removeProjectCache(projectPath, params.Project, params.Force)
			if err != nil {
				return err
			}
		}
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

func (c *Cache) removeProjectCache(projectDir, namespace string, force bool) error {
	if !force {
		ok, fail := c.confirm.Confirm(locale.Tr("clean_cache_artifact_confirm", namespace), false)
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

	runtime, err := runtime.NewRuntime(projectDir, "", parsed.Owner, parsed.Project, nil)
	if err != nil {
		return locale.WrapError(err, "err_clean_cache_runtime_init", "Could not determine cache directory for project used in {{.V0}}", projectDir)
	}
	projectInstallPath := runtime.InstallPath()

	logging.Debug("Remove project path: %s", projectInstallPath)
	err = os.RemoveAll(projectInstallPath)
	if err != nil {
		return locale.WrapError(err, "err_clean_remove_artifact", "Could not remove cached runtime environment for project: {{.V0}}", namespace)
	}

	return nil
}
