package version

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/blang/semver"
)

// Semver increment type values.
const (
	Zeroed = "zeroed"
	Patch  = "patch"
	Minor  = "minor"
	Major  = "major"
)

// IncrementTyper represents a client/service that returns strings related to
// semver increment type values.
type IncrementTyper interface {
	IncrementType() (string, error)
}

// Env helps define and limit the available environments.
type Env int

// Env constants are the available environment tokens.
const (
	UnknownEnv Env = iota
	LocalEnv
	RemoteEnv
)

// Incrementation provides methods for incrementing version numbers
type Incrementation struct {
	branch string
	env    Env
	typer  IncrementTyper
}

// NewIncrementation returns a version service initialized with provider and environment information
func NewIncrementation(typer IncrementTyper, branchName string, buildEnv Env) (*Incrementation, error) {
	return &Incrementation{
		branch: branchName,
		env:    buildEnv,
		typer:  typer,
	}, nil
}

// Increment bumps the master version based on the current build environment
// and the increment type provided.
func (v *Incrementation) Increment() (*semver.Version, error) {
	return v.incrementFromEnvironment()
}

// IncrementWithRevision bumps the master version based on the current build
// environment, the increment type and revision string provided.
func (v *Incrementation) IncrementWithRevision(revision string) (*semver.Version, error) {
	version, err := v.incrementFromEnvironment()
	if err != nil {
		return nil, err
	}

	prVersion, err := semver.NewPRVersion("SHA" + revision)
	if err != nil {
		return nil, fmt.Errorf("failed to create pre-release version number: %v", err)
	}
	version.Pre = []semver.PRVersion{prVersion}

	return version, nil
}

// needsIncrement whether we need to an increment for the environment
func needsIncrement(env Env, branch string) bool {
	return true
	return env != LocalEnv && (branch == "master" || branch == "unstable")
}

// Type returns the string representation of the version bump
// ie. patch, minor, or major
func (v *Incrementation) Type() (string, error) {
	if needsIncrement(v.env, v.branch) {
		return v.typer.IncrementType()
	}

	return Zeroed, nil
}

func fetchLatestVersionString(branch string) (string, error) {
	type versionJSON struct {
		Version string
	}
	var v versionJSON

	rootPath, err := environment.GetRootPath()
	if err != nil {
		return "", err
	}

	versionFilePath := filepath.Join(rootPath, "build", "version.json")
	if !fileutils.FileExists(versionFilePath) {
		return "", errors.New("Version file does not exist")
	}

	data, fail := fileutils.ReadFile(versionFilePath)
	if fail != nil {
		return "", err
	}

	err = json.Unmarshal(data, &v)
	if err != nil {
		return "", err
	}

	return v.Version, err
}

func masterVersion(branchName string) (*semver.Version, error) {
	versionString, err := fetchLatestVersionString(branchName)
	if err != nil {
		return nil, err
	}

	regex := regexp.MustCompile(`\d+\.\d+\.\d+-(SHA)?[a-f0-9]+`)
	match := regex.FindString(versionString)
	if match == "" {
		return nil, errors.New("could not determine master version")
	}

	masterVersion, err := semver.New(match)
	if err != nil {
		return nil, fmt.Errorf("failed to create semver from version string: %s", err)
	}
	masterVersion.Pre = nil

	return masterVersion, nil
}

func (v *Incrementation) incrementFromEnvironment() (*semver.Version, error) {
	switch v.env {
	case LocalEnv:
		// return v.increment()
		return semver.New("0.0.0")
	case RemoteEnv:
		return v.increment()
	default:
		return nil, errors.New("encountered unknown build environment")
	}
}

func incrementFrom(baseVersion *semver.Version, increment string) (*semver.Version, error) {
	copy := *baseVersion
	switch increment {
	case Patch:
		copy.Patch++
	case Minor:
		copy.Minor++
		copy.Patch = 0
	case Major:
		copy.Major++
		copy.Minor = 0
		copy.Patch = 0
	default:
		return nil, fmt.Errorf("encountered unexpected increment value: %s", increment)
	}

	return &copy, nil
}

func (v *Incrementation) increment() (*semver.Version, error) {
	inc, err := v.Type()
	if err != nil {
		return nil, err
	}

	if inc == Zeroed {
		return semver.New("0.0.0")
	}

	baseVersion, err := masterVersion(v.branch)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Base version for increment is: %s\n", baseVersion)

	return incrementFrom(baseVersion, inc)
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
