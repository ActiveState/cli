package runtime_helpers

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/hash"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/runtime"
)

/*
This package contains helpers for interacting with the runtime. Because while the runtime package itself may not deal
with certain concepts, like projects, we still want convenience layers for interacting with the runtime from the perspective
of projects.
*/

func FromProject(proj *project.Project) (_ *runtime.Runtime, rerr error) {
	targetDir := TargetDirFromProject(proj)
	rt, err := runtime.New(targetDir)
	if err != nil {
		return nil, errs.Wrap(err, "Could not initialize runtime")
	}
	return rt, nil
}

func ExecutorPathFromProject(proj *project.Project) string {
	return runtime.ExecutorsPath(TargetDirFromProject(proj))
}

func TargetDirFromProject(proj *project.Project) string {
	if cache := proj.Cache(); cache != "" {
		return cache
	}

	resolvedDir, err := fileutils.ResolveUniquePath(proj.Dir())
	if err != nil {
		multilog.Error("Could not resolve unique path for projectDir: %s, error: %s", proj.Dir(), err.Error())
		resolvedDir = proj.Dir()
	}

	return filepath.Join(storage.CachePath(), hash.ShortHash(resolvedDir))
}
