package runtime_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"

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

	_, err := runtime.NewCamelInstall(strfmt.UUID(""), cacheDir, []*runtime.HeadChefArtifact{
		invalidExtension,
		testArtifact,
		noURIArtifact,
	})
	suite.Error(err, runtime.ErrInvalidArtifact)
}

func (suite *CamelRuntimeTestSuite) Test_PreUnpackArtifact() {
	cacheDir, cacheCleanup := suite.genCacheDir()
	defer cacheCleanup()

	cases := []struct {
		name          string
		prepFunc      func(installDir string)
		expectedError error
	}{
		{"InstallationDirectoryIsFile", func(installDir string) {
			baseDir := filepath.Dir(installDir)
			err := fileutils.MkdirUnlessExists(baseDir)
			suite.Require().NoError(err)
			err = ioutil.WriteFile(installDir, []byte{}, 0666)
			suite.Require().NoError(err)
		}, runtime.ErrInstallDirInvalid},
		{"InstallationDirectoryIsNotEmpty", func(installDir string) {
			err := fileutils.MkdirUnlessExists(installDir)
			suite.Require().NoError(err)
			err = ioutil.WriteFile(filepath.Join(installDir, "dummy"), []byte{}, 0666)
			suite.Require().NoError(err)
		}, nil},
		{"InstallationDirectoryIsOkay", func(installDir string) {}, nil},
	}

	archivePath := "does-not-matter-here" + camelInstallerExtension()
	artifact, _ := headchefArtifact(archivePath)
	for _, tc := range cases {
		suite.Run(tc.name, func() {
			cr, err := runtime.NewCamelInstall(strfmt.UUID(""), cacheDir, []*runtime.HeadChefArtifact{artifact})
			suite.Require().NoError(err)

			os.RemoveAll(cacheDir)
			defer os.RemoveAll(cacheDir)

			tc.prepFunc(cacheDir)
			err = cr.PreUnpackArtifact(artifact)
			if tc.expectedError == nil {
				suite.Require().NoError(err)
			} else {
				suite.ErrorIs(err, tc.expectedError)
			}
		})
	}
}

func Test_CamelRuntimeTestSuite(t *testing.T) {
	suite.Run(t, new(CamelRuntimeTestSuite))
}
