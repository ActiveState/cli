package project

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type ProjectInternalTestSuite struct {
	suite.Suite
	projectFile *projectfile.Project
}

func (suite *ProjectInternalTestSuite) BeforeTest(suiteName, testName string) {
	failures.ResetHandled()
}

func (suite *ProjectInternalTestSuite) TestPassParseURL() {
	// url pass including commitID
	meta, fail := parseURL("https://platform.activestate.com/Org/TestParseURL?commitID=00010001-0001-0001-0001-000100010001")
	suite.NoError(fail.ToError(), "Should load project without issue.")
	suite.Equal("Org", meta.owner, "They should match")
	suite.Equal("TestParseURL", meta.name, "They should match")
	suite.Equal("00010001-0001-0001-0001-000100010001", meta.commitID, "They should match")
}

func (suite *ProjectInternalTestSuite) TestPassParseURLWithDots() {
	// url pass including commitID
	meta, fail := parseURL("https://platform.activestate.com/Org.Name/Project.Name?commitID=00010001-0001-0001-0001-000100010001")
	suite.NoError(fail.ToError(), "Should load project without issue.")
	suite.Equal("Org.Name", meta.owner, "They should match")
	suite.Equal("Project.Name", meta.name, "They should match")
	suite.Equal("00010001-0001-0001-0001-000100010001", meta.commitID, "They should match")
}

func (suite *ProjectInternalTestSuite) TestPassParseURLNoCommit() {
	// url pass without commitID
	meta, fail := parseURL("https://platform.activestate.com/Org/TestParseURL")
	suite.NoError(fail.ToError(), "Should load project without issue.")
	suite.Equal("Org", meta.owner, "They should match")
	suite.Equal("TestParseURL", meta.name, "They should match")
	suite.Equal("", meta.commitID, "They should match")
}

func (suite *ProjectInternalTestSuite) TestPassParseURLWithCommitPath() {
	// url pass including commitID
	meta, fail := parseURL("https://platform.activestate.com/commit/00010001-0001-0001-0001-000100010001")
	suite.NoError(fail.ToError(), "Should load project without issue.")
	suite.Equal("", meta.owner, "They should match")
	suite.Equal("", meta.name, "They should match")
	suite.Equal("00010001-0001-0001-0001-000100010001", meta.commitID, "They should match")
}

func (suite *ProjectInternalTestSuite) TestFailParseURL() {
	// url fail
	_, fail := parseURL("Thisisnotavalidaprojecturl")
	suite.Error(fail.ToError(), "This should fail")
}

func Test_ProjectInternalTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectInternalTestSuite))
}
