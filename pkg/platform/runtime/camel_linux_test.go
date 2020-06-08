// +build linux

package runtime_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/archiver"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	pMock "github.com/ActiveState/cli/internal/progress/mock"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/stretchr/testify/suite"
)

type CamelLinuxRuntimeTestSuite struct {
	suite.Suite

	dataDir string
}

func (suite *CamelLinuxRuntimeTestSuite) BeforeTest(suiteName, testName string) {
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "failure obtaining root path")

	suite.dataDir = filepath.Join(root, "pkg", "platform", "runtime", "testdata")

}

func (suite *CamelLinuxRuntimeTestSuite) TestRelocate() {

	cacheDir, err := ioutil.TempDir("", "cli-installer-test-cache")
	suite.Require().NoError(err)
	defer os.RemoveAll(cacheDir)

	relocationPrefix := "######################################## RELOCATE ME ########################################"

	fileutils.CopyFile(filepath.Join(suite.dataDir, "relocate/bin/python3"), filepath.Join(cacheDir, "relocate/bin/python3"))

	binary := "relocate/binary"
	fileutils.CopyFile(filepath.Join(suite.dataDir, binary), filepath.Join(cacheDir, binary))

	text := "relocate/text.go"
	fileutils.CopyFile(filepath.Join(suite.dataDir, text), filepath.Join(cacheDir, text))

	// Mock metaData
	metaData := &runtime.MetaData{
		Path:          filepath.Join(cacheDir, "relocate"),
		RelocationDir: relocationPrefix,
		BinaryLocations: []runtime.MetaDataBinary{
			runtime.MetaDataBinary{
				Path:     "bin",
				Relative: true,
			},
		},
		Env: map[string]string{},
	}

	metaData.Prepare()
	suite.Equal("lib", metaData.RelocationTargetBinaries)

	installDir := filepath.Join(cacheDir, "relocate")

	counter := pMock.NewMockIncrementer()

	fail := runtime.Relocate(metaData, func() { counter.Increment() })
	suite.Require().NoError(fail.ToError())

	suite.Assert().Equal(3, counter.Count, "3 files relocated")

	// test text
	suite.Contains(string(fileutils.ReadFileUnsafe(filepath.Join(cacheDir, text))), fmt.Sprintf("-- %s --", installDir))

	// test binary
	libDir := filepath.Join(cacheDir, "relocate/lib")
	binaryData := fileutils.ReadFileUnsafe(filepath.Join(cacheDir, binary))
	suite.True(len(bytes.Split(binaryData, []byte(libDir))) > 1, "Correctly injects "+libDir)
}

func (suite *CamelLinuxRuntimeTestSuite) genCacheDir() (string, func()) {

	cacheDir, err := ioutil.TempDir("", "cli-camel-cache-dir")
	suite.Require().NoError(err, "cache dir created")
	return cacheDir, func() { os.RemoveAll(cacheDir) }
}

func (suite *CamelLinuxRuntimeTestSuite) Test_PostUnpackWithFailures() {
	cases := []struct {
		name            string
		archiveName     string
		expectedFailure *failures.FailureType
	}{
		{"RuntimeMissingPythonExecutable", "python-missing-python-binary.tar.gz", runtime.FailMetaDataNotDetected},
		{"PythonFoundButNotExecutable", "python-noexec-python.tar.gz", runtime.FailRuntimeNotExecutable},
		{"InstallerFailsToGetPrefixes", "python-fail-prefixes.tar.gz", runtime.FailRuntimeNoPrefixes},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			cacheDir, cacheCleanup := suite.genCacheDir()
			defer cacheCleanup()
			runtimeDir, err := ioutil.TempDir("", "cli-camel-runtime-dir")
			defer os.RemoveAll(runtimeDir)

			archivePath := filepath.Join(suite.dataDir, tc.archiveName)
			suite.Require().NoError(err, "runtime dir created")
			err = archiver.Unarchive(archivePath, runtimeDir)
			suite.Require().NoError(err, "could not unarchive test archive %s", archivePath)

			artifact, _ := headchefArtifact(archivePath)
			counter := pMock.NewMockIncrementer()

			cr, fail := runtime.NewCamelRuntime([]*runtime.HeadChefArtifact{artifact}, cacheDir)
			suite.Require().NoError(fail.ToError(), "camel runtime assembler initialized")
			fail = fileutils.MkdirUnlessExists(cr.InstallationDirectory(artifact))
			suite.Require().NoError(fail.ToError(), "creating installation directory")
			fail = cr.PostUnpackArtifact(artifact, runtimeDir, archivePath, func() { counter.Increment() })

			suite.Require().Error(fail.ToError())
			suite.Equal(tc.expectedFailure, fail.Type)
			suite.Assert().Equal(0, counter.Count)
		})
	}
}

func Test_CamelLinuxRuntimeTestSuite(t *testing.T) {
	suite.Run(t, new(CamelLinuxRuntimeTestSuite))
}
