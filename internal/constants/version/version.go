package version

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/blang/semver"
)

func Detect() (semver.Version, error) {
	root, err := environment.GetRootPath()
	if err != nil {
		return semver.Version{}, err
	}
	versionFile := filepath.Join(root, "version.txt")
	_, err = os.Stat(versionFile)
	if err != nil {
		return semver.Version{}, fmt.Errorf("Could not access version.txt file at %s: %w", versionFile, err)
	}
	data, err := ioutil.ReadFile(versionFile)
	if err != nil {
		return semver.Version{}, fmt.Errorf("Could not read from file %s: %w", versionFile, err)
	}
	v, err := semver.Parse(strings.TrimSpace(string(data)))
	if err != nil {
		return semver.Version{}, fmt.Errorf("Failed to parse version from file %s: %w", versionFile, err)
	}
	return v, nil
}

func VersionWithRevision(v semver.Version, revision string) (semver.Version, error) {
	prVersion, err := semver.NewPRVersion("SHA" + revision)
	if err != nil {
		return semver.Version{}, fmt.Errorf("failed to create pre-release version number: %v", err)
	}
	v.Pre = []semver.PRVersion{prVersion}

	return v, nil
}

// NumberIsProduction returns whether or not the provided version number
// indicates a production build.
func NumberIsProduction(number string) bool {
	version, err := semver.Parse(number)
	if err != nil {
		return false
	}

	return version.Major > 0 || version.Minor > 0 || version.Patch > 0
}
