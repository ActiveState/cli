// +build !darwin

package runtime_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	rt "runtime"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/progress"
	hcMock "github.com/ActiveState/cli/pkg/platform/api/headchef/mock"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	rtMock "github.com/ActiveState/cli/pkg/platform/runtime/mock"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/sysinfo"
)

type RuntimeDLTestSuite struct {
	suite.Suite

	project *project.Project
	dir     string

	prg *progress.Progress

	hcMock *hcMock.Mock
	rtMock *rtMock.Mock
}

func (suite *RuntimeDLTestSuite) BeforeTest(suiteName, testName string) {
	projectURL := fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
	pj := &projectfile.Project{Project: projectURL}
	var fail *failures.Failure
	suite.project, fail = project.New(pj)
	suite.NoError(fail.ToError(), "No failure should occur when loading project")

	var err error
	suite.dir, err = ioutil.TempDir("", "runtime-test")
	suite.Require().NoError(err)

	suite.hcMock = hcMock.Init()
	suite.rtMock = rtMock.Init()

	suite.rtMock.MockFullRuntime()

	pjfile := projectfile.Project{
		Project: projectURL,
	}
	pjfile.Persist()

	cachePath := config.CachePath()
	if fileutils.DirExists(cachePath) {
		err := os.RemoveAll(config.CachePath())
		suite.Require().NoError(err)
	}

	// Only linux is supported for now, so force it so we can run this test on mac
	// If we want to skip this on mac it should be skipped through build tags, in
	// which case this tweak is meaningless and only a convenience for when testing manually
	if rt.GOOS == "darwin" {
		model.HostPlatform = sysinfo.Linux.String()
	}

	suite.prg = progress.New(progress.WithMutedOutput())
}

func (suite *RuntimeDLTestSuite) AfterTest(suiteName, testName string) {
	suite.rtMock.Close()
	suite.hcMock.Close()

	err := os.RemoveAll(suite.dir)
	suite.Require().NoError(err)
	suite.prg.Close()
}

func (suite *RuntimeDLTestSuite) TestGetRuntimeDL() {
	r := runtime.NewDownload(suite.project, suite.dir, suite.hcMock.Requester(hcMock.NoOptions))
	artfs, fail := r.FetchArtifacts()
	suite.Require().NoError(fail.ToError())
	files, fail := r.Download(artfs, suite.prg)
	suite.Require().NoError(fail.ToError())

	suite.Implements((*runtime.Downloader)(nil), r)
	suite.Contains(files, filepath.Join(suite.dir, "python"+runtime.InstallerExtension))
	suite.Contains(files, filepath.Join(suite.dir, "legacy-python"+runtime.InstallerExtension))

	for file := range files {
		suite.FileExists(file)
	}
}

func (suite *RuntimeDLTestSuite) TestGetRuntimeDLNoArtifacts() {
	r := runtime.NewDownload(suite.project, suite.dir, suite.hcMock.Requester(hcMock.NoArtifacts))
	_, fail := r.FetchArtifacts()
	suite.Require().Error(fail.ToError())

	suite.Equal(runtime.FailNoArtifacts.Name, fail.Type.Name)
}

func (suite *RuntimeDLTestSuite) TestGetRuntimeDLInvalidArtifact() {
	r := runtime.NewDownload(suite.project, suite.dir, suite.hcMock.Requester(hcMock.InvalidArtifact))
	_, fail := r.FetchArtifacts()
	suite.Require().Error(fail.ToError())

	suite.Equal(runtime.FailNoValidArtifact.Name, fail.Type.Name)
}

func (suite *RuntimeDLTestSuite) TestGetRuntimeDLInvalidURL() {
	r := runtime.NewDownload(suite.project, suite.dir, suite.hcMock.Requester(hcMock.InvalidURL))
	files, fail := r.FetchArtifacts()
	suite.Require().NoError(fail.ToError())
	_, fail = r.Download(files, suite.prg)
	suite.Require().Error(fail.ToError())

	suite.Equal(model.FailSignS3URL.Name, fail.Type.Name)
}

func (suite *RuntimeDLTestSuite) TestGetRuntimeDLBuildFailure() {
	r := runtime.NewDownload(suite.project, suite.dir, suite.hcMock.Requester(hcMock.BuildFailure))
	_, fail := r.FetchArtifacts()
	suite.Require().Error(fail.ToError())

	suite.Equal(runtime.FailBuild.Name, fail.Type.Name)
}

func (suite *RuntimeDLTestSuite) TestGetRuntimeDLFailure() {
	r := runtime.NewDownload(suite.project, suite.dir, suite.hcMock.Requester(hcMock.RegularFailure))
	_, fail := r.FetchArtifacts()
	suite.Require().Error(fail.ToError())

	suite.Equal(failures.FailDeveloper.Name, fail.Type.Name)
}

func TestRuntimeDLSuite(t *testing.T) {
	suite.Run(t, new(RuntimeDLTestSuite))
}
