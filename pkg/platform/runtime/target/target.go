package target

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/hash"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"
)

type Trigger string

func (t Trigger) String() string {
	return string(t)
}

const (
	TriggerActivate           Trigger = "activate"
	TriggerScript             Trigger = "script"
	TriggerDeploy             Trigger = "deploy"
	TriggerExec               Trigger = "exec-cmd"
	TriggerExecutor           Trigger = "exec"
	TriggerResetExec          Trigger = "reset-exec"
	TriggerSwitch             Trigger = "switch"
	TriggerImport             Trigger = "import"
	TriggerPackage            Trigger = "package"
	TriggerLanguage           Trigger = "language"
	TriggerPlatform           Trigger = "platform"
	TriggerPull               Trigger = "pull"
	TriggerReset              Trigger = "reset"
	TriggerRevert             Trigger = "revert"
	TriggerOffline            Trigger = "offline"
	TriggerShell              Trigger = "shell"
	TriggerCheckout           Trigger = "checkout"
	TriggerUse                Trigger = "use"
	TriggerOfflineInstaller   Trigger = "offline-installer"
	TriggerOfflineUninstaller Trigger = "offline-uninstaller"
	triggerUnknown            Trigger = "unknown"
)

// usageTriggers are triggers that indicate actual usage of the runtime (as oppose to simply making changes to the runtime)
var usageTriggers = []Trigger{
	TriggerActivate,
	TriggerScript,
	TriggerDeploy,
	TriggerExec,
	TriggerExecutor,
	TriggerSwitch,
	TriggerImport,
	TriggerPackage,
	TriggerPull,
	TriggerReset,
	TriggerRevert,
	TriggerShell,
	TriggerCheckout,
	TriggerUse,
	TriggerOfflineInstaller,
	TriggerOfflineUninstaller,
}

func NewExecTrigger(cmd string) Trigger {
	return Trigger(fmt.Sprintf("%s: %s", TriggerExec, cmd))
}

func (t Trigger) IndicatesUsage() bool {
	if funk.Contains(usageTriggers, t) {
		return true
	}
	return t.IsExecTrigger() && funk.Contains(usageTriggers, TriggerExec)
}

func (t Trigger) IsExecTrigger() bool {
	return strings.HasPrefix(string(t), string(TriggerExec)+": ")
}

type ProjectTarget struct {
	*project.Project
	cacheDir     string
	customCommit *strfmt.UUID
	trigger      Trigger
}

func NewProjectTarget(pj *project.Project, customCommit *strfmt.UUID, trigger Trigger) *ProjectTarget {
	runtimeCacheDir := storage.CachePath()
	if pj.Cache() != "" {
		runtimeCacheDir = pj.Cache()
	}
	return &ProjectTarget{pj, runtimeCacheDir, customCommit, trigger}
}

func NewProjectTargetCache(pj *project.Project, cacheDir string, customCommit *strfmt.UUID, trigger Trigger) *ProjectTarget {
	return &ProjectTarget{pj, cacheDir, customCommit, trigger}
}

func (p *ProjectTarget) Dir() string {
	if p.Project.Cache() != "" {
		return p.Project.Cache()
	}
	return ProjectDirToTargetDir(filepath.Dir(p.Project.Source().Path()), p.cacheDir)
}

func (p *ProjectTarget) CommitUUID() strfmt.UUID {
	if p.customCommit != nil {
		return *p.customCommit
	}
	return p.Project.CommitUUID()
}

func (p *ProjectTarget) Trigger() Trigger {
	if p.trigger == "" {
		return triggerUnknown
	}
	return p.trigger
}

func (p *ProjectTarget) Headless() bool {
	return p.Project.IsHeadless()
}

func (p *ProjectTarget) ReadOnly() bool {
	return false
}

func (p *ProjectTarget) InstallFromDir() *string {
	return nil
}

func ProjectDirToTargetDir(projectDir, cacheDir string) string {
	resolvedDir, err := fileutils.ResolveUniquePath(projectDir)
	if err != nil {
		multilog.Error("Could not resolve unique path for projectDir: %s, error: %s", projectDir, err.Error())
		resolvedDir = projectDir
	}
	logging.Debug("In newStore: resolved project dir is: %s", resolvedDir)

	return filepath.Join(cacheDir, hash.ShortHash(resolvedDir))
}

type CustomTarget struct {
	owner      string
	name       string
	commitUUID strfmt.UUID
	dir        string
	trigger    Trigger
	headless   bool
}

func NewCustomTarget(owner string, name string, commitUUID strfmt.UUID, dir string, trigger Trigger, headless bool) *CustomTarget {
	cleanDir, err := fileutils.ResolveUniquePath(dir)
	if err != nil {
		multilog.Error("Could not resolve unique path for dir: %s, error: %s", dir, err.Error())
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

func (c *CustomTarget) Trigger() Trigger {
	if c.trigger == "" {
		return triggerUnknown
	}
	return c.trigger
}

func (c *CustomTarget) Headless() bool {
	return c.headless
}

func (c *CustomTarget) ReadOnly() bool {
	return c.commitUUID == ""
}

func (c *CustomTarget) InstallFromDir() *string {
	return nil
}

type OfflineTarget struct {
	ns           *project.Namespaced
	dir          string
	artifactsDir string
	trigger      Trigger
}

func NewOfflineTarget(namespace *project.Namespaced, dir string, artifactsDir string) *OfflineTarget {
	cleanDir, err := fileutils.ResolveUniquePath(dir)
	if err != nil {
		multilog.Error("Could not resolve unique path for dir: %s, error: %s", dir, err.Error())
	} else {
		dir = cleanDir
	}
	return &OfflineTarget{namespace, dir, artifactsDir, TriggerOffline}
}

func (i *OfflineTarget) Owner() string {
	if i.ns == nil {
		return ""
	}
	return i.ns.Owner
}

func (i *OfflineTarget) Name() string {
	if i.ns == nil {
		return ""
	}
	return i.ns.Project
}

func (i *OfflineTarget) CommitUUID() strfmt.UUID {
	if i.ns == nil || i.ns.CommitID == nil {
		return ""
	}
	return *i.ns.CommitID
}

func (i *OfflineTarget) Dir() string {
	return i.dir
}

func (i *OfflineTarget) SetTrigger(t Trigger) {
	i.trigger = t
}

func (i *OfflineTarget) Trigger() Trigger {
	return i.trigger
}

func (i *OfflineTarget) Headless() bool {
	return false
}

func (i *OfflineTarget) ReadOnly() bool {
	return false
}

func (i *OfflineTarget) InstallFromDir() *string {
	return &i.artifactsDir
}
