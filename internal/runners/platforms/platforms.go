package platforms

import (
	"sort"
	"strconv"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type PlatformVersion struct {
	captain.NameVersion
}

func (pv *PlatformVersion) Set(arg string) error {
	err := pv.NameVersion.Set(arg)
	if err != nil {
		return locale.WrapInputError(err, "err_platform_format", "The platform and version provided is not formatting correctly, must be in the form of <platform>@<version>")
	}
	return nil
}

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
		if platform.CPUArchitecture != nil && platform.CPUArchitecture.BitWidth != nil {
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
	Platform PlatformVersion
	BitWidth int
	name     string
	version  string
}

func prepareParams(ps Params) (Params, error) {
	ps.name = ps.Platform.Name()
	if ps.name == "" {
		ps.name = model.HostPlatform
	}
	ps.version = ps.Platform.Version()
	if ps.version == "" {
		return prepareLatestVersion(ps)
	}

	if ps.BitWidth == 0 {
		ps.BitWidth = 32 << (^uint(0) >> 63) // gets host word size
	}

	return ps, nil
}

func prepareLatestVersion(params Params) (Params, error) {
	platformUUID, err := model.PlatformNameToPlatformID(params.Platform.Name())
	if err != nil {
		return params, locale.WrapInputError(err, "err_resolve_platform_id", "Could not resolve platform ID from name: {{.V0}}", params.Platform.Name())
	}

	platform, err := model.FetchPlatformByUID(strfmt.UUID(platformUUID))
	if err != nil {
		return params, locale.WrapError(err, "err_fetch_platform", "Could not get platform details")
	}
	params.name = *platform.Kernel.Name
	params.version = *platform.KernelVersion.Version

	bitWidth, err := strconv.Atoi(*platform.CPUArchitecture.BitWidth)
	if err != nil {
		return params, locale.WrapError(err, "err_platform_bitwidth", "Unable to determine platform bit width")
	}
	params.BitWidth = bitWidth

	return params, nil
}
