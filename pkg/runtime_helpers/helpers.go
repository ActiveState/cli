package runtime_helpers

import (
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/hash"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/multilog"
	buildscript_runbit "github.com/ActiveState/cli/internal/runbits/buildscript"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/runtime"
	"github.com/go-openapi/strfmt"
)

/*
This package contains helpers for interacting with the runtime. Because while the runtime package itself may not deal
with certain concepts, like projects, we still want convenience layers for interacting with the runtime from the perspective
of projects.
*/

func FromProject(proj *project.Project) (*runtime.Runtime, error) {
	targetDir := TargetDirFromProject(proj)
	rt, err := runtime.New(targetDir)
	if err != nil {
		return nil, errs.Wrap(err, "Could not initialize runtime")
	}
	return rt, nil
}

func NeedsUpdate(proj *project.Project, overrideCommitID *strfmt.UUID, cfg *config.Instance) (bool, error) {
	rt, err := FromProject(proj)
	if err != nil {
		return false, errs.Wrap(err, "Could not obtain runtime")
	}

	hash, err := Hash(proj, overrideCommitID, cfg)
	if err != nil {
		return false, errs.Wrap(err, "Could not get hash")
	}

	return hash != rt.Hash(), nil
}

func Hash(proj *project.Project, overrideCommitID *strfmt.UUID, cfg *config.Instance) (string, error) {
	var err error
	var commitID strfmt.UUID
	if overrideCommitID == nil {
		commitID, err = buildscript_runbit.CommitID(proj.Dir(), cfg)
		if err != nil {
			return "", errs.Wrap(err, "Failed to get commit ID")
		}
	} else {
		commitID = *overrideCommitID
	}

	path, err := fileutils.ResolveUniquePath(proj.Dir())
	if err != nil {
		return "", errs.Wrap(err, "Could not resolve unique path for projectDir")
	}

	return hash.ShortHash(strings.Join([]string{proj.NamespaceString(), path, commitID.String(), constants.RevisionHashShort}, "")), nil
}

func ExecutorPathFromProject(proj *project.Project) string {
	return runtime.ExecutorsPath(TargetDirFromProject(proj))
}

func TargetDirFromProject(proj *project.Project) string {
	if cache := proj.Cache(); cache != "" {
		return cache
	}

	return filepath.Join(storage.CachePath(), DirNameFromProjectDir(proj.Dir()))
}

func DirNameFromProjectDir(dir string) string {
	resolvedDir, err := fileutils.ResolveUniquePath(dir)
	if err != nil {
		multilog.Error("Could not resolve unique path for projectDir: %s, error: %s", dir, err.Error())
		resolvedDir = dir
	}

	return hash.ShortHash(resolvedDir)
}

func TargetDirFromProjectDir(path string) (string, error) {
	// Attempt to route via project file if it exists, since this considers the configured cache dir
	if fileutils.TargetExists(filepath.Join(path, constants.ConfigFileName)) {
		proj, err := project.FromPath(path)
		if err != nil {
			return "", errs.Wrap(err, "Could not load project from path")
		}
		return TargetDirFromProject(proj), nil
	}

	// Fall back on the provided path, because we can't assume the project file exists and is valid
	resolvedDir, err := fileutils.ResolveUniquePath(path)
	if err != nil {
		multilog.Error("Could not resolve unique path for projectDir: %s, error: %s", path, err.Error())
		resolvedDir = path
	}

	return filepath.Join(storage.CachePath(), hash.ShortHash(resolvedDir)), nil
}
