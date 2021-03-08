package runtime

import (
	"path/filepath"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/hash"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/project"
)

type ProjectTarget struct {
	*project.Project
	cacheDir string
}

func NewProjectTarget(pj *project.Project, runtimeCacheDir string) *ProjectTarget {
	return &ProjectTarget{pj, runtimeCacheDir}
}

func (p *ProjectTarget) Dir() string {
	projectDirDirty := filepath.Dir(p.Project.Source().Path())
	projectDir, err := fileutils.ResolveUniquePath(projectDirDirty)
	if err != nil {
		logging.Error("Could not resolve unique path for projectDir: %s, error: %s", projectDir, err.Error())
		projectDir = projectDirDirty
	}
	logging.Debug("In newStore: resolved project dir is: %s", projectDir)

	return filepath.Join(p.cacheDir, hash.ShortHash(projectDir))
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
