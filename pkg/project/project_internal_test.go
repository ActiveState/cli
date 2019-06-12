package project

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/pkg/projectfile"
	yaml "gopkg.in/yaml.v2"

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
		prj, fail := suite.loadProject(`project: "https://platform.activestate.com/Project/TestParseURL?commitID=00010001-0001-0001-0001-000100010001"`)
		suite.NoError(fail.ToError(), "Should load project without issue.")
		meta, fail := prj.parseURL()
		suite.NoError(fail.ToError(), "Should load project without issue.")
		suite.Equal("Project", meta.owner, "They should match")
		suite.Equal("TestParseURL", meta.name, "They should match")
		suite.Equal("00010001-0001-0001-0001-000100010001", meta.commitID, "They should match")
	}
	{
		// url pass without commitID
		prj, fail := suite.loadProject(`project: "https://platform.activestate.com/Project/TestParseURL"`)
		suite.NoError(fail.ToError(), "Should load project without issue.")
		meta, fail := prj.parseURL()
		suite.NoError(fail.ToError(), "Should load project without issue.")
		suite.Equal("Project", meta.owner, "They should match")
		suite.Equal("TestParseURL", meta.name, "They should match")
		suite.Equal("", meta.commitID, "They should match")
	}
	{
		// url fail
		_, fail := suite.loadProject(`project: "Thisisnotavalidaprojecturl"`)
		suite.Error(fail.ToError(), "This should fail")
	}

}

func (suite *ProjectInternalTestSuite) loadProject(projectStr string) (*Project, *failures.Failure) {
	projectfile.Reset()

	pjFile := &projectfile.Project{}
	contents := strings.TrimSpace(projectStr)
	err := yaml.Unmarshal([]byte(contents), pjFile)
	suite.NoError(err, "Unmarshalled YAML")

	pjFile.Persist()
	prj, fail := New(pjFile)
	return prj, fail
}

func Test_ProjectInternalTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectInternalTestSuite))
}
