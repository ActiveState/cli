package preprocess

import (
	"errors"
	"fmt"
	"log"
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
	IncrementType(branch string) (string, error)
}

// NewVersionIncrementer returns a version service initialized with provider and environment information
func NewVersionIncrementer(provider IncrementProvider, branchName string, buildEnvironment int) *VersionIncrementer {
	return &VersionIncrementer{
		branch:      branchName,
		environment: buildEnvironment,
		provider:    provider,
	}
}

// IncrementVersion bumps the master version based on the current build
// environment and the increment provided
func (s *VersionIncrementer) IncrementVersion() (string, error) {
	var err error
	s.master, err = s.masterVersion()
	if err != nil {
		return "", err
	}

	return s.incrementFromEnvironment()
}

// MustIncrementVersion calls IncrementVersion, any subsequent failures
// are logged and the application will exit
func (s *VersionIncrementer) MustIncrementVersion() string {
	version, err := s.IncrementVersion()
	if err != nil {
		log.Fatalf("Failed to increment version: %s", err)
	}

	return version
}

// IncrementVersionPreRelease bumps the master version based on the current build
// environment, the increment and revision string provided
func (s *VersionIncrementer) IncrementVersionPreRelease(revision string) (string, error) {
	var err error
	s.master, err = s.masterVersionPreRelease(revision)
	if err != nil {
		return "", err
	}

	return s.incrementFromEnvironment()
}

// MustIncrementVersionPreRelease calls IncrementVersionPreRelease, any subsequent
// failures are logged and the application will exit
func (s *VersionIncrementer) MustIncrementVersionPreRelease(revision string) string {
	version, err := s.IncrementVersionPreRelease(revision)
	if err != nil {
		log.Fatalf("Failed to increment version: %s", err)
	}

	return version
}

func (s *VersionIncrementer) masterVersion() (*semver.Version, error) {
	cmd := exec.Command(constants.CommandName, "--version")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	regex := regexp.MustCompile("\\d+\\.\\d+\\.\\d+-[a-f0-9]+")
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

func (s *VersionIncrementer) masterVersionPreRelease(revision string) (*semver.Version, error) {
	version, err := s.masterVersion()
	if err != nil {
		return nil, err
	}

	prVersion, err := semver.NewPRVersion(revision)
	if err != nil {
		return nil, fmt.Errorf("failed to create pre-release version number: %v", err)
	}
	version.Pre = []semver.PRVersion{prVersion}

	return version, nil
}

func (s *VersionIncrementer) incrementFromEnvironment() (string, error) {
	switch s.environment {
	case localEnv:
		return s.master.String(), nil
	case remoteEnv:
		return s.incrementVersion()
	default:
		return "", errors.New("encountered unknown build environment")
	}
}

func (s *VersionIncrementer) incrementVersion() (string, error) {
	increment, err := s.provider.IncrementType(s.branch)
	if err != nil {
		return "", err
	}

	switch increment {
	case patch:
		s.master.Patch++
	case minor:
		s.master.Minor++
		s.master.Patch = 0
	case major:
		s.master.Major++
		s.master.Minor = 0
		s.master.Patch = 0
	default:
		return "", fmt.Errorf("encountered unexpected increment value: %s", increment)
	}

	return s.master.String(), nil
}
