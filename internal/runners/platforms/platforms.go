package platforms

import (
	"sort"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type PlatformVersion struct {
	captain.NameVersionValue
}

func (pv *PlatformVersion) Set(arg string) error {
	err := pv.NameVersionValue.Set(arg)
	if err != nil {
		return locale.WrapInputError(err, "err_platform_format", "The platform and version provided is not formatting correctly. It must be in the form of <platform>@<version>")
	}
	return nil
}

// Platform represents the output data of a platform.
type Platform struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Arch     string `json:"arch"`
	BitWidth string `json:"bitWidth"`
}

func makePlatformsFromModelPlatforms(platforms []*model.Platform) []*Platform {
	var ps []*Platform

	for _, platform := range platforms {
		if platform.EndOfSupportDate != nil && time.Since(time.Time(*platform.EndOfSupportDate)) > 0 {
			continue // ignore EOL platforms; the Platform will fail to resolve dependencies on them
		}

		var p Platform
		if platform.Kernel != nil && platform.Kernel.Name != nil {
			p.Name = *platform.Kernel.Name
		}
		if platform.KernelVersion != nil && platform.KernelVersion.Version != nil {
			p.Version = *platform.KernelVersion.Version
		}
		if platform.CPUArchitecture != nil && platform.CPUArchitecture.BitWidth != nil {
			p.Arch = *platform.CPUArchitecture.Name
			p.BitWidth = *platform.CPUArchitecture.BitWidth
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
	Platform        PlatformVersion
	BitWidth        int
	resolvedName    string // Holds the provided platforn name, or defaults to curernt platform name if not provided
	resolvedVersion string // Holds the provided platform version, or defaults to latest version if not provided
}
