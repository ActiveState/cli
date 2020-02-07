package platforms

import (
	"errors"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/model"
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
		if platform.Kernel != nil && platform.Kernel.Name != nil {
			p.Name = *platform.Kernel.Name
		}
		if platform.KernelVersion != nil && platform.KernelVersion.Version != nil {
			p.Version = *platform.KernelVersion.Version
		}
		if platform.CPUArchitecture != nil {
			p.WordSize = platform.CPUArchitecture.BitWidth
		}

		ps = append(ps, &p)
	}

	return ps
}

// Params represents the minimal defining details of a platform.
type Params struct {
	Name     string
	WordSize int
	Version  string
}

func prepareParams(ps Params) (Params, error) {
	if ps.Name == "" {
		ps.Name = model.HostPlatform
	}

	if ps.WordSize == 0 {
		ps.WordSize = 32 << (^uint(0) >> 63) // gets host word size
	}

	if ps.Version == "" {
		return ps, errors.New(locale.T("bad_platform_version"))
	}

	return ps, nil
}
