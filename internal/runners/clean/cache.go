package clean

import (
	"context"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/cli/pkg/runtime_helpers"
)

type Cache struct {
	prime   primeable
	output  output.Outputer
	config  configurable
	confirm prompt.Prompter
	path    string
	ipComm  svcctl.IPCommunicator
}

type CacheParams struct {
	Project string
}

func NewCache(prime primeable) *Cache {
	return newCache(prime, prime.Output(), prime.Config(), prime.Prompt(), prime.IPComm())
}

func newCache(prime primeable, output output.Outputer, cfg configurable, confirm prompt.Prompter, ipComm svcctl.IPCommunicator) *Cache {
	return &Cache{
		prime:   prime,
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
			err := c.removeProjectCache(projectPath, params.Project)
			if err != nil {
				return err
			}
		}
		return nil
	}

	return c.removeCache(c.path)
}

func (c *Cache) removeCache(path string) error {
	defaultValue := !c.prime.Prompt().IsInteractive()
	ok, err := c.prime.Prompt().Confirm(locale.T("confirm"), locale.T("clean_cache_confirm"), &defaultValue, nil)
	if err != nil {
		return errs.Wrap(err, "Could not confirm")
	}
	if !ok {
		return locale.NewInputError("err_clean_cache_not_confirmed", "Cleaning of cache aborted by user")
	}
	if !c.prime.Prompt().IsInteractive() {
		c.prime.Output().Notice(locale.T("prompt_continue_non_interactive"))
	}

	inUse, err := c.checkPathInUse(path)
	if err != nil {
		return errs.Wrap(err, "Failed to check if path is in use")
	}
	if inUse {
		return locale.NewInputError("err_clean_in_use")
	}

	logging.Debug("Removing cache path: %s", path)
	if err := removeCache(c.path); err != nil {
		return errs.Wrap(err, "Failed to remove cache")
	}

	c.output.Notice(locale.Tl("clean_cache_success_message", "Successfully cleaned cache."))
	return nil
}

func (c *Cache) removeProjectCache(projectDir, namespace string) error {
	defaultValue := !c.prime.Prompt().IsInteractive()
	ok, err := c.prime.Prompt().Confirm(locale.T("confirm"), locale.Tr("clean_cache_artifact_confirm", namespace), &defaultValue, nil)
	if err != nil {
		return errs.Wrap(err, "Could not confirm")
	}
	if !ok {
		return locale.NewInputError("err_clean_cache_artifact_not_confirmed", "Cleaning of cached runtime aborted by user")
	}
	if !c.prime.Prompt().IsInteractive() {
		c.prime.Output().Notice(locale.T("prompt_continue_non_interactive"))
	}

	inUse, err := c.checkPathInUse(projectDir)
	if err != nil {
		return errs.Wrap(err, "Failed to check if path is in use")
	}
	if inUse {
		return locale.NewInputError("err_clean_in_use")
	}

	projectInstallPath, err := runtime_helpers.TargetDirFromProjectDir(projectDir)
	if err != nil {
		return errs.Wrap(err, "Failed to determine project install path")
	}

	logging.Debug("Remove project path: %s", projectInstallPath)
	if err := os.RemoveAll(projectInstallPath); err != nil {
		return locale.WrapError(err, "err_clean_remove_artifact", "Could not remove cached runtime environment for project: {{.V0}}", namespace)
	}

	return nil
}

func (c *Cache) checkPathInUse(path string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	procs, err := c.prime.SvcModel().GetProcessesInUse(ctx, path)
	if err != nil {
		return false, errs.Wrap(err, "Failed to get processes in use")
	}

	if len(procs) > 0 {
		return true, nil
	}

	return false, nil
}
