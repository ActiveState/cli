package platforms

import (
	"errors"
	"sort"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// Platform represents the output data of a platform.
type Platform struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	BitWidth string `json:"bitWidth"`
}

func (l *Listing) MarshalOutput(format output.Format) interface{} {
	if format == output.PlainFormatName {
		if len(l.Platforms) == 0 {
			return locale.Tl("platforms_list_no_platforms", "There are no platforms for this project.")
		}
		return l.Platforms
	}

	return l.Platforms
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
			p.BitWidth = platform.CPUArchitecture.BitWidth
		}

		ps = append(ps, &p)
	}

	sort.Slice(ps, func(i, j int) bool {
		tmpI := strings.ToLower(ps[i].Name) + ps[i].BitWidth + ps[i].Version
		tmpJ := strings.ToLower(ps[j].Name) + ps[j].BitWidth + ps[j].Version
		return tmpI < tmpJ
	})

	return ps
}

// Params represents the minimal defining details of a platform.
type Params struct {
	Name     string
	BitWidth int
	Version  string
}

func prepareParams(ps Params) (Params, error) {
	if ps.Name == "" {
		ps.Name = model.HostPlatform
	}

	if ps.BitWidth == 0 {
		ps.BitWidth = 32 << (^uint(0) >> 63) // gets host word size
	}

	if ps.Version == "" {
		return ps, errors.New(locale.T("err_bad_platform_version"))
	}

	return ps, nil
}
