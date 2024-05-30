package target

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/hash"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type Targeter interface {
	CommitUUID() strfmt.UUID
	Name() string
	Owner() string
	Trigger() Trigger
	Dir() string
}

type Target struct {
	owner       string
	name        string
	dirOverride *string
	commit      strfmt.UUID
	trigger     Trigger
}

func NewProjectTarget(owner, name string, commit strfmt.UUID, trigger Trigger, dirOverride *string) *Target {
	return &Target{owner, name, dirOverride, commit, trigger}
}

func NewProjectTargetCache(pj *project.Project, cacheDir string, customCommit *strfmt.UUID, trigger Trigger) *Target {
	return &Target{pj, cacheDir, customCommit, trigger}
}

func (t *Target) Owner() string {
	return t.owner
}

func (t *Target) Name() string {
	return t.name
}

func (t *Target) CommitUUID() strfmt.UUID {
	return t.commit
}

func (t *Target) Trigger() Trigger {
	if t.trigger == "" {
		return triggerUnknown
	}
	return t.trigger
}

func (t *Target) Dir() string {
	if t.dirOverride != nil {
		return *t.dirOverride
	}
	return filepath.Join(storage.CachePath(), hash.ShortHash())
}

func ProjectDirToTargetDir(cacheDir, projectDir string) string {
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
}

func NewCustomTarget(owner string, name string, commitUUID strfmt.UUID, dir string, trigger Trigger) *CustomTarget {
	cleanDir, err := fileutils.ResolveUniquePath(dir)
	if err != nil {
		multilog.Error("Could not resolve unique path for dir: %s, error: %s", dir, err.Error())
	} else {
		dir = cleanDir
	}
	return &CustomTarget{owner, name, commitUUID, dir, trigger}
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

func (c *CustomTarget) InstallDir() string {
	return c.dir
}

func (c *CustomTarget) Trigger() Trigger {
	if c.trigger == "" {
		return triggerUnknown
	}
	return c.trigger
}

func (c *CustomTarget) ReadOnly() bool {
	return c.commitUUID == ""
}

func (c *CustomTarget) ProjectDir() string {
	return ""
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

func (i *OfflineTarget) InstallDir() string {
	return i.dir
}

func (i *OfflineTarget) SetTrigger(t Trigger) {
	i.trigger = t
}

func (i *OfflineTarget) Trigger() Trigger {
	return i.trigger
}

func (i *OfflineTarget) ReadOnly() bool {
	return false
}

func (i *OfflineTarget) InstallFromDir() *string {
	return &i.artifactsDir
}

func (i *OfflineTarget) ProjectDir() string {
	return ""
}
