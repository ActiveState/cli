// +build linux

package runtime_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/failures"
	hcMock "github.com/ActiveState/cli/pkg/platform/api/headchef/mock"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	rtMock "github.com/ActiveState/cli/pkg/platform/runtime/mock"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/suite"
)

type RuntimeDLTestSuite struct {
	suite.Suite

	project *project.Project
	dir     string

	hcMock *hcMock.Mock
	rtMock *rtMock.Mock
}

func (suite *RuntimeDLTestSuite) BeforeTest(suiteName, testName string) {
	pj := &projectfile.Project{Name: "string", Owner: "string"}
	suite.project = project.New(pj)

	var err error
	suite.dir, err = ioutil.TempDir("", "runtime-test")
	suite.Require().NoError(err)

	suite.hcMock = hcMock.Init()
	suite.rtMock = rtMock.Init()

	suite.rtMock.MockFullRuntime()
}

func (suite *RuntimeDLTestSuite) AfterTest(suiteName, testName string) {
	suite.rtMock.Close()
	suite.hcMock.Close()

	err := os.RemoveAll(suite.dir)
	suite.Require().NoError(err)
}

func (suite *RuntimeDLTestSuite) TestGetRuntimeDL() {
	r := runtime.NewRuntimeDownload(suite.project, suite.dir, suite.hcMock.Requester(hcMock.NoOptions))
	filename, fail := r.Download()

	suite.Require().NoError(fail.ToError())
	suite.Implements((*runtime.Downloader)(nil), r)
	suite.Equal("archive.tar.gz", filename)
	suite.FileExists(filepath.Join(suite.dir, filename))
}

func (suite *RuntimeDLTestSuite) TestGetRuntimeDLNoArtifacts() {
	r := runtime.NewRuntimeDownload(suite.project, suite.dir, suite.hcMock.Requester(hcMock.NoArtifacts))
	_, fail := r.Download()

	suite.Equal(runtime.FailNoArtifacts.Name, fail.Type.Name)
}

func (suite *RuntimeDLTestSuite) TestGetRuntimeDLInvalidArtifact() {
	r := runtime.NewRuntimeDownload(suite.project, suite.dir, suite.hcMock.Requester(hcMock.InvalidArtifact))
	_, fail := r.Download()

	suite.Equal(runtime.FailNoValidArtifact.Name, fail.Type.Name)
}

func (suite *RuntimeDLTestSuite) TestGetRuntimeDLInvalidURL() {
	r := runtime.NewRuntimeDownload(suite.project, suite.dir, suite.hcMock.Requester(hcMock.InvalidURL))
	_, fail := r.Download()

	suite.Equal(model.FailSignS3URL.Name, fail.Type.Name)
}

func (suite *RuntimeDLTestSuite) TestGetRuntimeDLBuildFailure() {
	r := runtime.NewRuntimeDownload(suite.project, suite.dir, suite.hcMock.Requester(hcMock.BuildFailure))
	_, fail := r.Download()

	suite.Equal(runtime.FailBuild.Name, fail.Type.Name)
}

func (suite *RuntimeDLTestSuite) TestGetRuntimeDLFailure() {
	r := runtime.NewRuntimeDownload(suite.project, suite.dir, suite.hcMock.Requester(hcMock.RegularFailure))
	_, fail := r.Download()

	suite.Equal(failures.FailDeveloper.Name, fail.Type.Name)
}

func TestRuntimeDLSuite(t *testing.T) {
	suite.Run(t, new(RuntimeDLTestSuite))
}
