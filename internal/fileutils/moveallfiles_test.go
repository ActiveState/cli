package fileutils_test

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
)

type MoveAllFilesTestSuite struct {
	suite.Suite

	fromDir string
	toDir   string
}

func (suite *MoveAllFilesTestSuite) BeforeTest(suiteName, testName string) {
	var err error

	suite.fromDir, err = ioutil.TempDir("", "mvallfiles-from")
	suite.Require().NoError(err, "creating temp from-dir")

	suite.toDir, err = ioutil.TempDir("", "mvallfiles-to")
	suite.Require().NoError(err, "creating temp to-dir")
}

func (suite *MoveAllFilesTestSuite) AfterTest(suiteName, testName string) {
	os.RemoveAll(suite.toDir)
	os.RemoveAll(suite.fromDir)
}

func (suite *MoveAllFilesTestSuite) TestFromDir_IsNotDirectory() {
	tmpFile, err := ioutil.TempFile("", "mvallfiles-tmpfile")
	suite.Require().NoError(err, "creating fake from-dir as a file")

	failure := fileutils.MoveAllFiles(tmpFile.Name(), suite.toDir)
	suite.Require().NotNil(failure, "moving files")
	suite.Equal(fileutils.FailMoveSourceNotDirectory, failure.Type)
	suite.Equal(locale.Tr("err_os_not_a_directory", tmpFile.Name()), failure.Error())
}

func (suite *MoveAllFilesTestSuite) TestToDir_IsNotDirectory() {
	tmpFile, err := ioutil.TempFile("", "mvallfiles-tmpfile")
	suite.Require().NoError(err, "creating fake from-dir as a file")

	failure := fileutils.MoveAllFiles(suite.fromDir, tmpFile.Name())
	suite.Require().NotNil(failure, "moving files")
	suite.Equal(fileutils.FailMoveDestinationNotDirectory, failure.Type)
	suite.Equal(locale.Tr("err_os_not_a_directory", tmpFile.Name()), failure.Error())
}

func (suite *MoveAllFilesTestSuite) addFileToFrom(relPath string) {
	fail := fileutils.Touch(path.Join(suite.fromDir, relPath))
	suite.Require().Nil(fail, "touching test file: %s", relPath)
}

func (suite *MoveAllFilesTestSuite) addDirToFrom(relPath string) {
	failure := fileutils.Mkdir(path.Join(suite.fromDir, relPath))
	suite.Require().Nil(failure, "creating test dir: %s", relPath)
}

func (suite *MoveAllFilesTestSuite) TestSuccess() {
	suite.addFileToFrom("a")
	suite.addDirToFrom("dir1")
	suite.addFileToFrom("dir1/b")
	suite.addDirToFrom("dir1/dir1.1")
	suite.addFileToFrom("dir1/dir1.1/c")
	suite.addDirToFrom("dir2")
	suite.addFileToFrom("dir2/d")

	suite.False(fileutils.FileExists(path.Join(suite.toDir, "a")))
	suite.False(fileutils.FileExists(path.Join(suite.toDir, "dir1/b")))
	suite.False(fileutils.FileExists(path.Join(suite.toDir, "dir1/dir1.1/c")))
	suite.False(fileutils.FileExists(path.Join(suite.toDir, "dir2/d")))

	suite.Require().Nil(fileutils.MoveAllFiles(suite.fromDir, suite.toDir))
	suite.True(fileutils.FileExists(path.Join(suite.toDir, "a")))
	suite.True(fileutils.FileExists(path.Join(suite.toDir, "dir1/b")))
	suite.True(fileutils.FileExists(path.Join(suite.toDir, "dir1/dir1.1/c")))
	suite.True(fileutils.FileExists(path.Join(suite.toDir, "dir2/d")))
}

func Test_MoveAllFilesTestSuite(t *testing.T) {
	suite.Run(t, new(MoveAllFilesTestSuite))
}
