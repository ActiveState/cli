package version

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/blang/semver"
)

func ParseStateToolVersion(version string) (semver.Version, error) {
	ver, err := semver.Parse(version)
	if err != nil {
		return ver, errs.Wrap(err, "Failed to parse State Tool version %s", version)
	}

	return ver, nil
}

func IsMultiFileUpdate(version semver.Version) bool {
	// We ignore pre-release tags for this version test, as our attaching of `-SHA123455` is technically interpreted as a pre-release
	testVer := version
	testVer.Pre = nil
	return testVer.GTE(semver.MustParse(constants.FirstMultiFileStateToolVersion)) || (version.Major == 0 && version.Minor == 0 && version.Patch == 0)
}
