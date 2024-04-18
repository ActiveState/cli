package projectfile

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/suite"
)

type ProjectInternalTestSuite struct {
	suite.Suite
}

func (suite *ProjectInternalTestSuite) TestPassParseURL() {
	// url pass without commitID
	meta, err := parseURL("https://platform.activestate.com/Org/TestParseURL")
	suite.Equal(err, ErrInvalidCommitID, "Should return an invalid commit ID error")
	suite.False(IsFatalError(err), "Error should be non-fatal")
	suite.Equal("Org", meta.Owner, "They should match")
	suite.Equal("TestParseURL", meta.Name, "They should match")
	suite.Equal("", meta.LegacyCommitID, "They should match")
}

func (suite *ProjectInternalTestSuite) TestPassParseLegacyURL() {
	// url pass including commitID
	meta, err := parseURL("https://platform.activestate.com/Org/TestParseURL?commitID=00010001-0001-0001-0001-000100010001")
	suite.NoError(err, "Should load project without issue.")
	suite.Equal("Org", meta.Owner, "They should match")
	suite.Equal("TestParseURL", meta.Name, "They should match")
	suite.Equal("00010001-0001-0001-0001-000100010001", meta.LegacyCommitID, "They should match")
}

func (suite *ProjectInternalTestSuite) TestPassParseURLWithDots() {
	meta, err := parseURL("https://platform.activestate.com/Org.Name/Project.Name")
	suite.False(IsFatalError(err), "Should load project without fatal error")
	suite.Equal("Org.Name", meta.Owner, "They should match")
	suite.Equal("Project.Name", meta.Name, "They should match")
	suite.Equal("", meta.LegacyCommitID, "They should match")
}

func (suite *ProjectInternalTestSuite) TestPassParseURLWithCommitPath() {
	// url pass including commitID
	meta, err := parseURL("https://platform.activestate.com/commit/00010001-0001-0001-0001-000100010001")
	suite.NoError(err, "Should load project without issue.")
	suite.Equal("", meta.Owner, "They should match")
	suite.Equal("", meta.Name, "They should match")
	suite.Equal("00010001-0001-0001-0001-000100010001", meta.LegacyCommitID, "They should match")
}

func (suite *ProjectInternalTestSuite) TestFailParseURL() {
	// url fail
	_, err := parseURL("Thisisnotavalidaprojecturl")
	suite.Error(err, "This should fail.")
	suite.True(IsFatalError(err), "It should be a fatal error")
}

func Test_ProjectInternalTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectInternalTestSuite))
}
