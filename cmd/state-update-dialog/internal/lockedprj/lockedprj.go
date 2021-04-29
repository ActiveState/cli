package lockedprj

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/version"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type LockedCheckout struct {
	Path    string // The path at which the project is checked out
	Channel string // The channel at which the State Tool version is locked at
	Version string // The version at which the State Tool is locked at
}

func LockedProjectMapping(cfg projectfile.ConfigGetter) map[string][]LockedCheckout {
	localProjects := projectfile.GetProjectFileMapping(cfg)
	lockedProjects := make(map[string][]LockedCheckout)
	for name, prjs := range localProjects {

		var locks []LockedCheckout
		for _, prj := range prjs {
			if prj.VersionBranch() != "" && prj.Version() != "" {
				ver, err := version.ParseStateToolVersion(prj.Version())
				if err != nil {
					logging.Error("Failed to parse State Tool version %s: %v", prj.Version, err)
				}
				// We can ignore projects that are locked to a multi-file update version
				if version.IsMultiFileUpdate(ver) {
					continue
				}
				locks = append(locks, LockedCheckout{filepath.Dir(prj.Path()), prj.VersionBranch(), prj.Version()})
			}
		}
		if len(locks) > 0 {
			lockedProjects[name] = locks
		}
	}
	return lockedProjects
}
