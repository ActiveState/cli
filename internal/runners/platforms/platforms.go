package platforms

import (
	"sort"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/go-openapi/strfmt"
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
	Name     string
	BitWidth int
	Version  string
}

func prepareParams(ps Params) (Params, error) {
	ps.Name, ps.Version = splitNameAndVersion(ps.Name)
	if ps.Name == "" {
		ps.Name = model.HostPlatform
	}

	if ps.BitWidth == 0 {
		ps.BitWidth = 32 << (^uint(0) >> 63) // gets host word size
	}

	if ps.Version == "" {
		return prepareLatestVersion(ps)
	}

	return ps, nil
}

func splitNameAndVersion(input string) (string, string) {
	nameArg := strings.Split(input, "@")
	name := nameArg[0]
	version := ""
	if len(nameArg) == 2 {
		version = nameArg[1]
	}

	return name, version
}

func prepareLatestVersion(params Params) (Params, error) {
	platformUUID, fail := model.HostPlatformToPlatformID(params.Name)
	if fail != nil {
		return params, locale.WrapInputError(fail.ToError(), "err_resolve_platform_id", "Could not resolve platform ID from name: {{.V0}}", params.Name)
	}

	platform, fail := model.FetchPlatformByUID(strfmt.UUID(platformUUID))
	if fail != nil {
		return params, locale.WrapError(fail.ToError(), "err_fetch_platform", "Could not get platform details")
	}
	params.Name = *platform.Kernel.Name
	params.Version = *platform.KernelVersion.Version

	bitWidth, err := strconv.Atoi(*platform.CPUArchitecture.BitWidth)
	if err != nil {
		return params, locale.WrapError(err, "err_platform_bitwidth", "Unable to determine platform bit width")
	}
	params.BitWidth = bitWidth

	return params, nil
}
