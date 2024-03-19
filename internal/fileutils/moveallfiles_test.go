package fileutils_test

import (
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

	suite.fromDir, err = os.MkdirTemp("", "mvallfiles-from")
	suite.Require().NoError(err, "creating temp from-dir")

	suite.toDir, err = os.MkdirTemp("", "mvallfiles-to")
	suite.Require().NoError(err, "creating temp to-dir")
}

func (suite *MoveAllFilesTestSuite) AfterTest(suiteName, testName string) {
	os.RemoveAll(suite.toDir)
	os.RemoveAll(suite.fromDir)
}

func (suite *MoveAllFilesTestSuite) TestFromDir_IsNotDirectory() {
	tmpFile, err := os.CreateTemp("", "mvallfiles-tmpfile")
	suite.Require().NoError(err, "creating fake from-dir as a file")

	err = fileutils.MoveAllFiles(tmpFile.Name(), suite.toDir)
	suite.Require().NotNil(err, "moving files")
	suite.Equal(locale.Tr("err_os_not_a_directory", tmpFile.Name()), err.Error())
}

func (suite *MoveAllFilesTestSuite) TestToDir_IsNotDirectory() {
	tmpFile, err := os.CreateTemp("", "mvallfiles-tmpfile")
	suite.Require().NoError(err, "creating fake from-dir as a file")

	err = fileutils.MoveAllFiles(suite.fromDir, tmpFile.Name())
	suite.Require().NotNil(err, "moving files")
	suite.Equal(locale.Tr("err_os_not_a_directory", tmpFile.Name()), err.Error())
}

func (suite *MoveAllFilesTestSuite) addFileToFrom(relPath string) {
	err := fileutils.Touch(path.Join(suite.fromDir, relPath))
	suite.Require().Nil(err, "touching test file: %s", relPath)
}

func (suite *MoveAllFilesTestSuite) addDirToFrom(relPath string) {
	err := fileutils.Mkdir(path.Join(suite.fromDir, relPath))
	suite.Require().Nil(err, "creating test dir: %s", relPath)
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
