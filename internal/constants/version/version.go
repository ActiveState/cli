package version

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"

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
	// linux-amd64.json is our single source of truth for the latest version number
	stateURL := "https://s3.ca-central-1.amazonaws.com/cli-update/update/state/%s/linux-amd64.json"
	resp, err := http.Get(fmt.Sprintf(stateURL, branch))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	type versionJSON struct {
		Version string
	}
	var v versionJSON
	err = json.NewDecoder(resp.Body).Decode(&v)
	if err != nil {
		return "", err
	}

	return v.Version, err
}

func masterVersion(branchName string, buildEnv Env) (*semver.Version, error) {
	var err error
	if !needsIncrement(buildEnv, branchName) {
		return semver.New("0.0.0")
	}
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
		return semver.New("0.0.0")
	case RemoteEnv:
		return v.increment()
	default:
		return nil, errors.New("encountered unknown build environment")
	}
}

func (v *Incrementation) increment() (*semver.Version, error) {
	baseVersion, err := masterVersion(v.branch, v.env)
	if err != nil {
		return nil, err
	}
	increment, err := v.Type()
	fmt.Printf("Base version for increment is: %s\n", baseVersion)

	if err != nil {
		return nil, err
	}

	copy := *baseVersion
	switch increment {
	case Zeroed:
		copy.Major = 0
		copy.Minor = 0
		copy.Patch = 0
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

// NumberIsProduction returns whether or not the provided version number
// indicates a production build.
func NumberIsProduction(number string) bool {
	version, err := semver.Parse(number)
	if err != nil {
		return false
	}

	return version.Major > 0 || version.Minor > 0 || version.Patch > 0
}
