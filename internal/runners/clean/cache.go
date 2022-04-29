package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type Cache struct {
	output  output.Outputer
	config  configurable
	confirm confirmAble
	path    string
	ipComm  svcctl.IPCommunicator
}

type CacheParams struct {
	Force   bool
	Project string
}

func NewCache(prime primeable) *Cache {
	return newCache(prime.Output(), prime.Config(), prime.Prompt(), prime.IPComm())
}

func newCache(output output.Outputer, cfg configurable, confirm confirmAble, ipComm svcctl.IPCommunicator) *Cache {
	return &Cache{
		output:  output,
		config:  cfg,
		confirm: confirm,
		path:    storage.CachePath(),
		ipComm:  ipComm,
	}
}

func (c *Cache) Run(params *CacheParams) error {
	if os.Getenv(constants.ActivatedStateEnvVarName) != "" {
		return locale.NewError("err_clean_cache_activated")
	}

	if params.Project != "" {
		paths := projectfile.GetProjectPaths(c.config, params.Project)

		if len(paths) == 0 {
			return locale.NewInputError("err_cache_no_project", "Could not determine path to project {{.V0}}", params.Project)
		}

		for _, projectPath := range paths {
			err := c.removeProjectCache(projectPath, params.Project, params.Force)
			if err != nil {
				return err
			}
		}
		return nil
	}

	return c.removeCache(c.path, params.Force)
}

func (c *Cache) removeCache(path string, force bool) error {
	if !force {
		ok, err := c.confirm.Confirm(locale.T("confirm"), locale.T("clean_cache_confirm"), new(bool))
		if err != nil {
			return err
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
		ok, err := c.confirm.Confirm(locale.T("confirm"), locale.Tr("clean_cache_artifact_confirm", namespace), new(bool))
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	projectInstallPath := target.ProjectDirToTargetDir(projectDir, storage.CachePath())
	logging.Debug("Remove project path: %s", projectInstallPath)
	err := os.RemoveAll(projectInstallPath)
	if err != nil {
		return locale.WrapError(err, "err_clean_remove_artifact", "Could not remove cached runtime environment for project: {{.V0}}", namespace)
	}

	return nil
}
