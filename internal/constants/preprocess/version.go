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
	IncrementMaster() (*semver.Version, error)
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
func (s *VersionIncrementer) IncrementVersion() (string, error) {
	version, err := s.incrementFromEnvironment()
	if err != nil {
		return "", err
	}

	return version.String(), nil
}

// IncrementVersionRevision bumps the master version based on the current build
// environment, the increment and revision string provided
func (s *VersionIncrementer) IncrementVersionRevision(revision string) (string, error) {
	version, err := s.incrementFromEnvironment()
	if err != nil {
		return "", err
	}

	prVersion, err := semver.NewPRVersion(revision)
	if err != nil {
		return "", fmt.Errorf("failed to create pre-release version number: %v", err)
	}
	version.Pre = []semver.PRVersion{prVersion}

	return version.String(), nil
}

func masterVersion() (*semver.Version, error) {
	cmd := exec.Command(constants.CommandName, "--version")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	regex := regexp.MustCompile(`\d+\.\d+\.\d+-[a-f0-9]+`)
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

func (s *VersionIncrementer) incrementFromEnvironment() (*semver.Version, error) {
	switch s.environment {
	case localEnv:
		return s.master, nil
	case remoteEnv:
		return s.incrementVersion()
	default:
		return nil, errors.New("encountered unknown build environment")
	}
}

func (s *VersionIncrementer) incrementVersion() (*semver.Version, error) {
	if s.branch == "master" {
		return s.provider.IncrementMaster()
	}

	increment, err := s.provider.IncrementBranch()
	if err != nil {
		return nil, err
	}

	copy := *s.master
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
