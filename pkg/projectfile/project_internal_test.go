package projectfile

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ProjectInternalTestSuite struct {
	suite.Suite
}

func (suite *ProjectInternalTestSuite) TestPassParseURL() {
	// url pass without commitID
	meta, err := parseURL("https://platform.activestate.com/Org/TestParseURL")
	suite.NoError(err, "Should load project without issue.")
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
	// url pass including commitID
	meta, err := parseURL("https://platform.activestate.com/Org.Name/Project.Name")
	suite.NoError(err, "Should load project without issue.")
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
}

func Test_ProjectInternalTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectInternalTestSuite))
}
