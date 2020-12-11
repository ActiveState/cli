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
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	pMock "github.com/ActiveState/cli/internal/progress/mock"
	"github.com/ActiveState/cli/pkg/platform/runtime"
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

	err := runtime.Relocate(metaData, func() { counter.Increment() })
	suite.Require().NoError(err)

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
		name          string
		archiveName   string
		expectedError error
	}{
		{"RuntimeMissingPythonExecutable", "python-missing-python-binary.tar.gz", &runtime.ErrMetaData{}},
		{"PythonFoundButNotExecutable", "python-noexec-python.tar.gz", &runtime.ErrNotExecutable{}},
		{"InstallerFailsToGetPrefixes", "python-fail-prefixes.tar.gz", &runtime.ErrNoPrefixes{}},
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

			cr, err := runtime.NewCamelInstall(strfmt.UUID(""), cacheDir, []*runtime.HeadChefArtifact{artifact})
			suite.Require().NoError(err, "camel runtime assembler initialized")
			err = fileutils.MkdirUnlessExists(cacheDir)
			suite.Require().NoError(err, "creating installation directory")
			err = cr.PostUnpackArtifact(artifact, runtimeDir, archivePath, func() { counter.Increment() })

			suite.Require().Error(err)
			suite.ErrorAs(err, &tc.expectedError)
			suite.Assert().Equal(0, counter.Count)
		})
	}
}

func Test_CamelLinuxRuntimeTestSuite(t *testing.T) {
	suite.Run(t, new(CamelLinuxRuntimeTestSuite))
}
