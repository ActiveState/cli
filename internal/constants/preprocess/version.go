package preprocess

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/blang/semver"
)

const (
	unknownEnv = iota
	localEnv
	remoteEnv
)

const (
	zeroed = "zeroed"
	patch  = "patch"
	minor  = "minor"
	major  = "major"
)

// VersionIncrementer provides methods for incrementing version numbers
type VersionIncrementer struct {
	branch      string
	environment int
	master      *semver.Version
	typer       IncrementTyper
}

// IncrementTyper represents a client/service that returns
// strings related to semver increment values (ie. major, minor, patch)
type IncrementTyper interface {
	IncrementType() (string, error)
}

// NewVersionIncrementer returns a version service initialized with provider and environment information
func NewVersionIncrementer(typer IncrementTyper, branchName string, buildEnvironment int) (*VersionIncrementer, error) {
	master, err := masterVersion()
	if err != nil {
		return nil, err
	}

	return &VersionIncrementer{
		branch:      branchName,
		environment: buildEnvironment,
		typer:       typer,
		master:      master,
	}, nil
}

// IncrementVersion bumps the master version based on the current build
// environment and the increment provided
func (v *VersionIncrementer) IncrementVersion() (*semver.Version, error) {
	return v.incrementFromEnvironment()
}

// IncrementVersionRevision bumps the master version based on the current build
// environment, the increment and revision string provided
func (v *VersionIncrementer) IncrementVersionRevision(revision string) (*semver.Version, error) {
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

// IncrementType returns the string representation of the version bump
// ie. patch, minor, or major
func (v *VersionIncrementer) IncrementType() (string, error) {
	if v.environment != localEnv && v.branch == "master" {
		return v.typer.IncrementType()
	}

	return zeroed, nil
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

func (v *VersionIncrementer) incrementFromEnvironment() (*semver.Version, error) {
	switch v.environment {
	case localEnv:
		copy := *v.master
		copy.Major = 0
		copy.Minor = 0
		copy.Patch = 0
		return &copy, nil
	case remoteEnv:
		return v.incrementVersion()
	default:
		return nil, errors.New("encountered unknown build environment")
	}
}

func (v *VersionIncrementer) incrementVersion() (*semver.Version, error) {
	var increment string
	var err error

	switch v.branch {
	case "master":
		increment, err = v.typer.IncrementType()
	default:
		increment = zeroed
	}
	if err != nil {
		return nil, err
	}

	copy := *v.master
	switch increment {
	case zeroed:
		copy.Major = 0
		copy.Minor = 0
		copy.Patch = 0
	case patch:
		copy.Patch++
	case minor:
		copy.Minor++
		copy.Patch = 0
	case major:
		copy.Major++
		copy.Minor = 0
		copy.Patch = 0
	default:
		return nil, fmt.Errorf("encountered unexpected increment value: %s", increment)
	}

	return &copy, nil
}

// VersionNumberIsProduction returns whether or not the provided version number
// indicates a production build. The accuracy of this likely relies on constant
// generation being run first.
func VersionNumberIsProduction(versionNumber string) bool {
	version, err := semver.Parse(versionNumber)
	if err != nil {
		return false
	}

	return version.Major > 0 || version.Minor > 0 || version.Patch > 0
}
