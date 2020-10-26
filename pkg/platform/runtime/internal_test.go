package runtime

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/pkg/platform/api"
	graphMock "github.com/ActiveState/cli/pkg/platform/api/graphql/request/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type InternalTestSuite struct {
	suite.Suite

	cacheDir    string
	downloadDir string
	installer   *Installer
	authMock    *authMock.Mock
	httpmock    *httpmock.HTTPMock
	graphMock   *graphMock.Mock
}

func (suite *InternalTestSuite) BeforeTest(suiteName, testName string) {
	suite.authMock = authMock.Init()
	suite.authMock.MockLoggedin()
	suite.httpmock = httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())

	pjfile := projectfile.Project{
		Project: model.ProjectURL("string", "string", "00010001-0001-0001-0001-000100010001"),
	}
	pjfile.Persist()

	var err error
	suite.cacheDir, err = ioutil.TempDir("", "")
	suite.Require().NoError(err)

	suite.downloadDir, err = ioutil.TempDir("", "cli-installer-test-download")
	suite.Require().NoError(err)

	msgHandler := runbits.NewRuntimeMessageHandler(&outputhelper.TestOutputer{})
	r, err := NewRuntime("", "00010001-0001-0001-0001-000100010001", "string", "string", msgHandler)
	suite.Require().NoError(err)
	r.SetInstallPath(suite.cacheDir)
	suite.installer = NewInstaller(r)
	suite.Require().NotNil(suite.installer)

	suite.graphMock = graphMock.Init()
}

func (suite *InternalTestSuite) AfterTest(suiteName, testName string) {
	projectfile.Reset()
	suite.authMock.Close()
	httpmock.DeActivate()
	suite.graphMock.Close()
}

func (suite *InternalTestSuite) TestValidateCheckpointNoCommit() {
	msgHandler := runbits.NewRuntimeMessageHandler(&outputhelper.TestOutputer{})
	var fail *failures.Failure
	r, err := NewRuntime("", "", "string", "string", msgHandler)
	suite.Require().NoError(err)
	r.SetInstallPath(suite.cacheDir)
	suite.installer = NewInstaller(r)
	suite.Require().NotNil(suite.installer)

	fail = suite.installer.validateCheckpoint()
	suite.Equal(FailNoCommitID.Name, fail.Type.Name)
}

func (suite *InternalTestSuite) TestValidateCheckpointPrePlatform() {
	suite.graphMock.CheckpointWithPrePlatform(graphMock.NoOptions)
	fail := suite.installer.validateCheckpoint()
	suite.Equal(FailPrePlatformNotSupported.Name, fail.Type.Name)
}

func (suite *InternalTestSuite) TestPPMShim() {
	dir := fileutils.TempDirUnsafe()
	err := installPPMShim(dir)
	suite.Require().NoError(err)

	exe, err := os.Executable()
	suite.Require().NoError(err)

	suite.FileExists(filepath.Join(dir, "ppm"))
	p := string(fileutils.ReadFileUnsafe(filepath.Join(dir, "ppm")))
	suite.True(strings.Index(p, exe) != -1, fmt.Sprintf("%s should contain %s", p, exe))
	if runtime.GOOS == "windows" {
		suite.FileExists(filepath.Join(dir, "ppm.bat"))
		p = string(fileutils.ReadFileUnsafe(filepath.Join(dir, "ppm.bat")))
		suite.True(strings.Index(p, exe) != -1, fmt.Sprintf("%s should contain %s", p, exe))
	}

}

func TestInternalTestSuite(t *testing.T) {
	suite.Run(t, new(InternalTestSuite))
}
