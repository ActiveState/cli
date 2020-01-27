package version

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/blang/semver"
)

const (
	unknownEnv = iota
	localEnv
	masterEnv
	remoteEnv
)

const (
	patch = "patch"
	minor = "minor"
	major = "major"
)

// Service provides methods for incrementing version numbers
type Service struct {
	branch      string
	environment int
	master      *semver.Version
	provider    IncrementProvider
}

// IncrementProvider represents a client/service that returns
// strings related to semver values (ie. major, minor, patch)
type IncrementProvider interface {
	Increment(branch string) (string, error)
}

// New returns a version service initialized with provider and environment information
func New(provider IncrementProvider, branchName string) (*Service, error) {
	environment, err := buildEnvironment(branchName)
	if err != nil {
		return nil, err
	}

	return &Service{
		branch:      branchName,
		environment: environment,
		provider:    provider,
	}, nil
}

// IncrementVersion bumps the master version based on the current build
// environment and the increment provided
func (s *Service) IncrementVersion() (string, error) {
	var err error
	s.master, err = s.masterVersion()
	if err != nil {
		return "", err
	}

	// TODO: Use common method
	switch s.environment {
	case localEnv:
		return s.master.String(), nil
	case masterEnv, remoteEnv:
		return s.incrementVersion()
	default:
		return "", errors.New("build state is not local, remote branch, remote master, or pull request")
	}
}

// MustIncrementVersion calls IncrementVersion, any subsequent failures
// are logged and the application will exit
func (s *Service) MustIncrementVersion() string {
	version, err := s.IncrementVersion()
	if err != nil {
		log.Fatalf("Failed to increment version: %s", err)
	}

	return version
}

// IncrementVersionPreRelease bumps the master version based on the current build
// environment, the increment and revision string provided
func (s *Service) IncrementVersionPreRelease(revision string) (string, error) {
	var err error
	s.master, err = s.masterVersionPreRelease(revision)
	if err != nil {
		return "", err
	}

	switch s.environment {
	case localEnv:
		return s.master.String(), nil
	case masterEnv, remoteEnv:
		return s.incrementVersion()
	default:
		return "", errors.New("build state is not local, remote branch, remote master, or pull request")
	}
}

// MustIncrementVersionPreRelease calls IncrementVersionPreRelease, any subsequent
// failures are logged and the application will exit
func (s *Service) MustIncrementVersionPreRelease(revision string) string {
	version, err := s.IncrementVersionPreRelease(revision)
	if err != nil {
		log.Fatalf("Failed to increment version: %s", err)
	}

	return version
}

func (s *Service) masterVersion() (*semver.Version, error) {
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

func (s *Service) masterVersionPreRelease(revision string) (*semver.Version, error) {
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

func (s *Service) incrementVersion() (string, error) {
	increment, err := s.provider.Increment(s.branch)
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

func buildEnvironment(branchName string) (int, error) {
	if !onCI() {
		return localEnv, nil
	}

	if branchName == "master" {
		return masterEnv, nil
	}

	return remoteEnv, nil
}

func onCI() bool {
	if os.Getenv("CI") != "" {
		return true
	}
	return false
}
