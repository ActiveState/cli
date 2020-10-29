package platforms

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/blang/semver"
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

	var err error
	if ps.Version == "" {
		ps.Version, err = latestPlatformVersion(ps.Name)
		if err != nil {
			return ps, locale.WrapError(err, "err_prepare_params_latest", "Could not determine latest version for the platform: {{.V0}}", ps.Name)
		}
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

type platformSemver struct {
	semver semver.Version
	patch  int
}

func latestPlatformVersion(name string) (string, error) {
	result, err := newSearchResult()
	if err != nil {
		return "", locale.WrapError(err, "err_latest_search_platforms", "Could not get platforms search result")
	}

	var latest platformSemver
	for _, platform := range result.Platforms {
		if strings.ToLower(platform.Name) == strings.ToLower(name) {
			version := platform.Version
			parts := strings.Split(version, ".")
			var patch string
			if len(parts) > 3 {
				version = strings.Join(parts[:len(parts)-1], ".")
				patch = parts[3]
			}

			parsed, err := semver.Parse(version)
			if err != nil {
				logging.Debug("Could not parse semver: %s for platform: %s", version, platform.Name)
				continue
			}

			parsedPatch, err := strconv.Atoi(patch)
			if err != nil {
				logging.Debug("Invalid patch value: %s for platform: %s", patch, platform.Name)
				parsedPatch = 0
			}

			if parsed.GE(latest.semver) && parsedPatch >= latest.patch {
				latest.semver = parsed
				latest.patch = parsedPatch
			}
		}
	}

	if latest.semver.Equals(semver.Version{}) {
		return "", locale.NewError("err_platform_not_found", "Could not find latest version for platform: {{.V0}}", name)
	}

	if latest.patch != 0 {
		return fmt.Sprintf("%s.%s", latest.semver.String(), strconv.Itoa(latest.patch)), nil
	}
	return latest.semver.String(), nil
}
