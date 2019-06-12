package project

import (
	"testing"

	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/stretchr/testify/suite"
)

type ProjectInternalTestSuite struct {
	suite.Suite
	projectFile *projectfile.Project
}

func (suite *ProjectInternalTestSuite) BeforeTest(suiteName, testName string) {
	failures.ResetHandled()
}

func (suite *ProjectInternalTestSuite) TestParseURL() {
	{
		// url pass including commitID
		meta, fail := parseURL("https://platform.activestate.com/Project/TestParseURL?commitID=00010001-0001-0001-0001-000100010001")
		suite.NoError(fail.ToError(), "Should load project without issue.")
		suite.Equal("Project", meta.owner, "They should match")
		suite.Equal("TestParseURL", meta.name, "They should match")
		suite.Equal("00010001-0001-0001-0001-000100010001", meta.commitID, "They should match")
	}
	{
		// url pass without commitID
		meta, fail := parseURL("https://platform.activestate.com/Project/TestParseURL")
		suite.NoError(fail.ToError(), "Should load project without issue.")
		suite.Equal("Project", meta.owner, "They should match")
		suite.Equal("TestParseURL", meta.name, "They should match")
		suite.Equal("", meta.commitID, "They should match")
	}
	{
		// url fail
		_, fail := parseURL("Thisisnotavalidaprojecturl")
		suite.Error(fail.ToError(), "This should fail")
	}

}

func Test_ProjectInternalTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectInternalTestSuite))
}
