package runtime_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/failures"

	"github.com/ActiveState/cli/internal/download"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"

	projMock "github.com/ActiveState/cli/internal/projects/mock"
	hcMock "github.com/ActiveState/cli/pkg/platform/api/headchef/mock"
	invMock "github.com/ActiveState/cli/pkg/platform/api/inventory/mock"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/sysinfo"
	"github.com/stretchr/testify/suite"
)

type RuntimeDLTestSuite struct {
	suite.Suite

	project *project.Project
	dir     string

	httpMock *httpmock.HTTPMock
	hcMock   *hcMock.Mock
	invMock  *invMock.Mock
	apiMock  *apiMock.Mock
	authMock *authMock.Mock
	projMock *projMock.Mock
}

func (suite *RuntimeDLTestSuite) BeforeTest(suiteName, testName string) {
	pj := &projectfile.Project{Name: "string", Owner: "string"}
	suite.project = project.New(pj)

	var err error
	suite.dir, err = ioutil.TempDir("", "runtime-test")
	suite.Require().NoError(err)

	suite.hcMock = hcMock.Init()
	suite.invMock = invMock.Init()
	suite.apiMock = apiMock.Init()
	suite.authMock = authMock.Init()
	suite.projMock = projMock.Init()
	suite.httpMock = httpmock.Activate("http://test.tld/")

	suite.authMock.MockLoggedin()
	suite.apiMock.MockVcsGetCheckpoint()
	suite.apiMock.MockSignS3URI()
	suite.invMock.MockOrderRecipes()
	suite.invMock.MockPlatforms()
	suite.projMock.MockGetProject()

	suite.httpMock.RegisterWithResponse("GET", "archive.tar.gz", 200, "archive.tar.gz")

	// Disable the mocking this lib does natively, it's a bad mechanic that has to change, but out of scope for right now
	download.SetMocking(false)

	model.OS = sysinfo.Linux // for now we only support linux, so force it
}

func (suite *RuntimeDLTestSuite) AfterTest(suiteName, testName string) {
	suite.hcMock.Close()
	suite.invMock.Close()
	suite.apiMock.Close()
	suite.authMock.Close()
	suite.projMock.Close()
	httpmock.DeActivate()

	err := os.RemoveAll(suite.dir)
	suite.Require().NoError(err)
}

func (suite *RuntimeDLTestSuite) TestGetRuntimeDL() {
	r := runtime.NewRuntimeDownload(suite.project, suite.dir, suite.hcMock.Requester(hcMock.NoOptions))
	filename, fail := r.Download()

	suite.Require().NoError(fail.ToError())
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
