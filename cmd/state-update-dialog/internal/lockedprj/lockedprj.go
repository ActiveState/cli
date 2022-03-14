package lockedprj

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/version"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type LockedCheckout struct {
	Name    string
	Path    string // The path at which the project is checked out
	Channel string // The channel at which the State Tool version is locked at
	Version string // The version at which the State Tool is locked at
}

func LockedProjectMapping(cfg projectfile.ConfigGetter) []LockedCheckout {
	localProjects := projectfile.GetProjectFileMapping(cfg)
	var lockedProjects []LockedCheckout
	for name, prjs := range localProjects {
		for _, prj := range prjs {
			if prj.Version() == "" {
				continue
			}
			ver, err := version.ParseStateToolVersion(prj.Version())
			if err != nil {
				multilog.Error("Failed to parse State Tool version %s: %v", prj.Version, err)
			}
			// We can ignore projects that are locked to a multi-file update version
			if version.IsMultiFileUpdate(ver) {
				continue
			}
			lockedProjects = append(lockedProjects, LockedCheckout{
				name,
				filepath.Dir(prj.Path()),
				prj.VersionBranch(),
				prj.Version(),
			})
		}
	}
	return lockedProjects
}
