package runtime_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/platform/runtime"
)

type CamelRuntimeTestSuite struct {
	suite.Suite
}

func (suite *CamelRuntimeTestSuite) genCacheDir() (string, func()) {

	cacheDir, err := ioutil.TempDir("", "cli-camel-cache-dir")
	suite.Require().NoError(err, "cache dir created")
	return cacheDir, func() { os.RemoveAll(cacheDir) }
}

func (suite *CamelRuntimeTestSuite) Test_InitializeWithInvalidArtifacts() {

	invalidExtension, _ := headchefArtifact("invalid-artifact.unknown")
	testArtifact, _ := headchefArtifact("invalid-artifact-tests.tar.gz")
	noURIArtifact, _ := headchefArtifact("")

	cacheDir, cacheCleanup := suite.genCacheDir()
	defer cacheCleanup()

	_, fail := runtime.NewCamelRuntime([]*runtime.HeadChefArtifact{
		invalidExtension,
		testArtifact,
		noURIArtifact,
	}, cacheDir)
	suite.Require().Error(fail.ToError(), "error in initialization of camel runtime assembler")
	suite.Assert().IsType(runtime.FailNoValidArtifact, fail.Type)
}

func (suite *CamelRuntimeTestSuite) Test_PreUnpackArtifact() {
	cacheDir, cacheCleanup := suite.genCacheDir()
	defer cacheCleanup()

	cases := []struct {
		name            string
		prepFunc        func(installDir string)
		expectedFailure *failures.FailureType
	}{
		{"InstallationDirectoryIsFile", func(installDir string) {
			baseDir := filepath.Dir(installDir)
			fail := fileutils.MkdirUnlessExists(baseDir)
			suite.Require().NoError(fail.ToError())
			err := ioutil.WriteFile(installDir, []byte{}, 0666)
			suite.Require().NoError(err)
		}, runtime.FailInstallDirInvalid},
		{"InstallationDirectoryIsNotEmpty", func(installDir string) {
			fail := fileutils.MkdirUnlessExists(installDir)
			suite.Require().NoError(fail.ToError())
			err := ioutil.WriteFile(filepath.Join(installDir, "dummy"), []byte{}, 0666)
			suite.Require().NoError(err)
		}, nil},
		{"InstallationDirectoryIsOkay", func(installDir string) {}, nil},
	}

	archivePath := "does-not-matter-here" + camelInstallerExtension()
	artifact, _ := headchefArtifact(archivePath)
	for _, tc := range cases {
		suite.Run(tc.name, func() {
			cr, fail := runtime.NewCamelRuntime([]*runtime.HeadChefArtifact{artifact}, cacheDir)
			suite.Require().NoError(fail.ToError())
			defer os.RemoveAll(cacheDir)

			tc.prepFunc(cacheDir)
			fail = cr.PreUnpackArtifact(artifact)
			if tc.expectedFailure == nil {
				suite.Require().NoError(fail.ToError())
				return
			}
			suite.Require().Error(fail.ToError())
			suite.Equal(tc.expectedFailure, fail.Type)
		})
	}
}

func Test_CamelRuntimeTestSuite(t *testing.T) {
	suite.Run(t, new(CamelRuntimeTestSuite))
}
