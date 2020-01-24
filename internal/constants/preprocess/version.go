package preprocess

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/blang/semver"
)

const (
	patch = "patch"
	minor = "minor"
	major = "major"
)

// TODO: Move this and the github type to another package?
type versionService struct {
	github      *githubClient
	branch      string
	environment int
	master      *semver.Version
}

func newVersionService(github *githubClient, branchName string) (*versionService, error) {
	environment, err := buildEnvironment(branchName)
	if err != nil {
		return nil, err
	}

	return &versionService{
		github:      github,
		branch:      branchName,
		environment: environment,
	}, nil
}

func (s *versionService) version() (string, error) {
	var err error
	s.master, err = s.masterVersion()
	if err != nil {
		return "", err
	}

	// TODO: Use common method
	switch s.environment {
	case localEnv, branchEnv:
		return s.master.String(), nil
	case masterEnv, pullRequestEnv:
		return s.incrementVersion()
	default:
		return "", errors.New("Build state is not local, remote branch, remote master, or pull request")
	}
}

func (s *versionService) versionPreRelease() (string, error) {
	var err error
	s.master, err = s.masterVersionPreRelease()
	if err != nil {
		return "", err
	}

	switch s.environment {
	case localEnv, branchEnv:
		return s.master.String(), nil
	case masterEnv, pullRequestEnv:
		return s.incrementVersion()
	default:
		return "", errors.New("build state is not local, remote branch, remote master, or pull request")
	}
}

func (s *versionService) masterVersion() (*semver.Version, error) {
	output := getCmdOutput(fmt.Sprintf("%s --version", constants.CommandName))
	regex := regexp.MustCompile("\\d+\\.\\d+\\.\\d+-[a-f0-9]+")
	match := regex.FindString(output)
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

func (s *versionService) masterVersionPreRelease() (*semver.Version, error) {
	version, err := s.masterVersion()
	if err != nil {
		return nil, err
	}

	prVersion, err := semver.NewPRVersion((Constants["RevisionHashShort"]()))
	if err != nil {
		return nil, fmt.Errorf("failed to create pre-release version number: %v", err)
	}
	version.Pre = []semver.PRVersion{prVersion}

	return version, nil
}

func (s *versionService) incrementVersion() (string, error) {
	increment, err := s.github.incrementValue(s.branch)
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

	prNum, err := pullRequestNumber()
	if err != nil {
		return unknownEnv, err
	}
	if prNum == 0 {
		return branchEnv, nil
	}

	return pullRequestEnv, nil
}

func onCI() bool {
	if os.Getenv("CI") != "" {
		return true
	}
	return false
}
