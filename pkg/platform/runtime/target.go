package runtime

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/hash"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/project"
)

type Trigger string

func (t Trigger) String() string {
	return string(t)
}

const (
	TriggerActivate Trigger = "activate"
	TriggerScript   Trigger = "script"
	TriggerDeploy   Trigger = "deploy"
	TriggerExec     Trigger = "exec"
	TriggerDefault  Trigger = "default-setup"
	TriggerBranch   Trigger = "branch"
	TriggerImport   Trigger = "import"
	TriggerPackage  Trigger = "package"
	TriggerPull     Trigger = "pull"
	TriggerReset    Trigger = "reset"
	TriggerRevert   Trigger = "revert"
	triggerUnknown  Trigger = "unknown"
)

func NewExecTrigger(cmd string) Trigger {
	return Trigger(fmt.Sprintf("%s: %s", TriggerExec, cmd))
}

type ProjectTarget struct {
	*project.Project
	cacheDir     string
	customCommit *strfmt.UUID
	trigger      Trigger
}

func NewProjectTarget(pj *project.Project, runtimeCacheDir string, customCommit *strfmt.UUID, trigger Trigger) *ProjectTarget {
	return &ProjectTarget{pj, runtimeCacheDir, customCommit, trigger}
}

func (p *ProjectTarget) Dir() string {
	return ProjectDirToTargetDir(filepath.Dir(p.Project.Source().Path()), p.cacheDir)
}

func (p *ProjectTarget) CommitUUID() strfmt.UUID {
	if p.customCommit != nil {
		return *p.customCommit
	}
	return p.Project.CommitUUID()
}

func (p *ProjectTarget) OnlyUseCache() bool {
	return false
}

func (p *ProjectTarget) Trigger() string {
	if p.trigger == "" {
		return triggerUnknown.String()
	}
	return p.trigger.String()
}

func (p *ProjectTarget) Headless() string {
	return strconv.FormatBool(p.Project.IsHeadless())
}

type CustomTarget struct {
	owner      string
	name       string
	commitUUID strfmt.UUID
	dir        string
	trigger    Trigger
	headless   string
}

func NewCustomTarget(owner string, name string, commitUUID strfmt.UUID, dir string, trigger Trigger, headless string) *CustomTarget {
	cleanDir, err := fileutils.ResolveUniquePath(dir)
	if err != nil {
		logging.Error("Could not resolve unique path for dir: %s, error: %s", dir, err.Error())
	} else {
		dir = cleanDir
	}
	return &CustomTarget{owner, name, commitUUID, dir, trigger, headless}
}

func (c *CustomTarget) Owner() string {
	return c.owner
}

func (c *CustomTarget) Name() string {
	return c.name
}

func (c *CustomTarget) CommitUUID() strfmt.UUID {
	return c.commitUUID
}

func (c *CustomTarget) Dir() string {
	return c.dir
}

func (c *CustomTarget) OnlyUseCache() bool {
	return c.commitUUID == ""
}

func (c *CustomTarget) Trigger() string {
	if c.trigger == "" {
		return triggerUnknown.String()
	}
	return c.trigger.String()
}

func (c *CustomTarget) Headless() string {
	return c.headless
}

func ProjectDirToTargetDir(projectDir, cacheDir string) string {
	resolvedDir, err := fileutils.ResolveUniquePath(projectDir)
	if err != nil {
		logging.Error("Could not resolve unique path for projectDir: %s, error: %s", projectDir, err.Error())
		resolvedDir = projectDir
	}
	logging.Debug("In newStore: resolved project dir is: %s", resolvedDir)

	return filepath.Join(cacheDir, hash.ShortHash(resolvedDir))
}
