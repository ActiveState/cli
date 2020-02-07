package platforms

import (
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// Platform represents the output data of a platform.
type Platform struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	WordSize string `json:"wordSize"`
}

func makePlatformsFromModelPlatforms(platforms []*model.Platform) []*Platform {
	var ps []*Platform

	for _, platform := range platforms {
		var p Platform
		if platform.Kernel != nil {
			p.Name = normString(platform.Kernel.Name)
		}
		if platform.KernelVersion != nil {
			p.Version = normString(platform.KernelVersion.Version)
		}
		if platform.CPUArchitecture != nil {
			p.WordSize = platform.CPUArchitecture.BitWidth
		}

		ps = append(ps, &p)
	}

	return ps
}

func normString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

type committer interface {
	CommitPlatform(op model.Operation, name, version string) error
}

type commitOp struct{}

func (c *commitOp) CommitPlatform(op model.Operation, name, version string) error {
	proj := project.Get()

	return model.CommitPlatform(proj.Owner(), proj.Name(), op, name, version)
}
