package version

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"

	"github.com/ActiveState/cli/internal/constants"
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
	master *semver.Version
	typer  IncrementTyper
}

// NewIncrementation returns a version service initialized with provider and environment information
func NewIncrementation(typer IncrementTyper, branchName string, buildEnv Env) (*Incrementation, error) {
	master, err := masterVersion()
	if err != nil {
		return nil, err
	}

	return &Incrementation{
		branch: branchName,
		env:    buildEnv,
		typer:  typer,
		master: master,
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

// Type returns the string representation of the version bump
// ie. patch, minor, or major
func (v *Incrementation) Type() (string, error) {
	if v.env != LocalEnv && (v.branch == "master" || v.branch == "unstable") {
		return v.typer.IncrementType()
	}

	return Zeroed, nil
}

func masterVersion() (*semver.Version, error) {
	cmd := exec.Command(constants.CommandName, "--version")
	output, err := cmd.Output()
	if err != nil {
		errMsg := err.Error()
		if ee, ok := err.(*exec.ExitError); ok {
			errMsg = fmt.Sprintf("Stderr: %s, code: %s", ee.Stderr, errMsg)
		}
		return nil, errors.New(errMsg)
	}

	regex := regexp.MustCompile(`\d+\.\d+\.\d+-(SHA)?[a-f0-9]+`)
	match := regex.FindString(string(output))
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
		copy := *v.master
		copy.Major = 0
		copy.Minor = 0
		copy.Patch = 0
		return &copy, nil
	case RemoteEnv:
		return v.increment()
	default:
		return nil, errors.New("encountered unknown build environment")
	}
}

func (v *Incrementation) increment() (*semver.Version, error) {
	var increment string
	var err error

	switch v.branch {
	case "master", "unstable":
		increment, err = v.typer.IncrementType()
	default:
		increment = Zeroed
	}
	if err != nil {
		return nil, err
	}

	copy := *v.master
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
