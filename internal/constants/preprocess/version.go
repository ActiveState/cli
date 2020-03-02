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
	patch = "patch"
	minor = "minor"
	major = "major"
)

// VersionIncrementer provides methods for incrementing version numbers
type VersionIncrementer struct {
	branch      string
	environment int
	master      *semver.Version
	provider    IncrementProvider
}

// IncrementProvider represents a client/service that returns
// strings related to semver increment values (ie. major, minor, patch)
type IncrementProvider interface {
	IncrementBranch() (string, error)
	IncrementMaster() (string, error)
}

// NewVersionIncrementer returns a version service initialized with provider and environment information
func NewVersionIncrementer(provider IncrementProvider, branchName string, buildEnvironment int) (*VersionIncrementer, error) {
	master, err := masterVersion()
	if err != nil {
		return nil, err
	}

	return &VersionIncrementer{
		branch:      branchName,
		environment: buildEnvironment,
		provider:    provider,
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

// IncrementString returns the string representation of the version bump
// ie. patch, minor, or major
func (v *VersionIncrementer) IncrementString() (string, error) {
	if v.environment == localEnv {
		return v.master.String(), nil
	}

	if v.branch == "master" {
		return v.provider.IncrementMaster()
	}
	return v.provider.IncrementBranch()
}

func masterVersion() (*semver.Version, error) {
	cmd := exec.Command(constants.CommandName, "--version")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	regex := regexp.MustCompile(`\d+\.\d+\.\d+-[SHA]?[a-f0-9]+`)
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
		return v.master, nil
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
		increment, err = v.provider.IncrementMaster()
	default:
		increment, err = v.provider.IncrementBranch()
	}
	if err != nil {
		return nil, err
	}

	copy := *v.master
	switch increment {
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
