package runtime

import (
	"path/filepath"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/hash"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/project"
)

type ProjectTarget struct {
	*project.Project
	cacheDir     string
	customCommit *strfmt.UUID
}

func NewProjectTarget(pj *project.Project, runtimeCacheDir string, customCommit *strfmt.UUID) *ProjectTarget {
	return &ProjectTarget{pj, runtimeCacheDir, customCommit}
}

func (p *ProjectTarget) Dir() string {
	projectDir := filepath.Dir(p.Project.Source().Path())
	targetDir, err := ProjectDirToTargetDir(projectDir, p.cacheDir)
	if err != nil {
		logging.Error("Could not translate project dir to target dir, falling back to project dir, error: %v", err)
		targetDir = projectDir
	}
	return targetDir
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

type CustomTarget struct {
	owner      string
	name       string
	commitUUID strfmt.UUID
	dir        string
}

func NewCustomTarget(owner string, name string, commitUUID strfmt.UUID, dir string) *CustomTarget {
	cleanDir, err := fileutils.ResolveUniquePath(dir)
	if err != nil {
		logging.Error("Could not resolve unique path for dir: %s, error: %s", dir, err.Error())
	} else {
		dir = cleanDir
	}
	return &CustomTarget{owner, name, commitUUID, dir}
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

func ProjectDirToTargetDir(projectDir, cacheDir string) (string, error) {
	resolvedDir, err := fileutils.ResolveUniquePath(projectDir)
	if err != nil {
		return "", locale.WrapError(err, "err_target_resolve_dir", "Could not resolve project directory")
	}
	logging.Debug("In newStore: resolved project dir is: %s", projectDir)

	return filepath.Join(cacheDir, hash.ShortHash(resolvedDir)), nil
}
