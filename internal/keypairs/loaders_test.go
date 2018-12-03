package keypairs_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/stretchr/testify/suite"
)

type KeypairLoadersTestSuite struct {
	suite.Suite
}

func (suite *KeypairLoadersTestSuite) TearDownSuite() {
	os.RemoveAll(config.GetDataDir())
}

func (suite *KeypairLoadersTestSuite) TestNoKeyFileFound() {
	kp, failure := keypairs.Load("test-no-such")
	suite.Nil(kp)
	suite.Truef(failure.Type.Matches(keypairs.FailLoadNotFound), "unexpected failure type: %v", failure)
}

func (suite *KeypairLoadersTestSuite) assertTooPermissive(fileMode os.FileMode) {
	tmpKeyName := fmt.Sprintf("test-rsa-%0.4o", fileMode)
	keyFile := suite.createConfigDirFile(tmpKeyName+".key", fileMode)
	suite.Require().NoError(keyFile.Close())

	kp, failure := keypairs.Load(tmpKeyName)
	suite.Nil(kp)
	suite.Require().NotNil(failure)
	suite.Truef(failure.Type.Matches(keypairs.FailLoadFileTooPermissive), "unexpected failure type: %v", failure)
}

func (suite *KeypairLoadersTestSuite) TestFileFound_PermsTooPermissive() {
	octalPerms := []os.FileMode{0640, 0650, 0660, 0670, 0604, 0605, 0606, 0607, 0700, 0500}
	for _, perm := range octalPerms {
		suite.assertTooPermissive(perm)
	}
}

func (suite *KeypairLoadersTestSuite) TestFileFound_KeypairParseError() {
	keyFile := suite.createConfigDirFile("test-rsa-parse-err.key", 0600)

	keyFile.WriteString("this will never parse")
	suite.Require().NoError(keyFile.Close())

	kp, failure := keypairs.Load("test-rsa-parse-err")
	suite.Nil(kp)
	suite.Require().NotNil(failure)
	suite.Truef(failure.Type.Matches(keypairs.FailKeypairParse), "unexpected failure type: %v", failure)
}

func (suite *KeypairLoadersTestSuite) TestFileFound_EncryptedKeypairParseFailure() {
	keyFile := suite.createConfigDirFile("test-rsa-encrypted.key", 0600)
	keyFile.WriteString(suite.readTestFile("test-keypair-encrypted.key"))
	suite.Require().NoError(keyFile.Close())

	kp, failure := keypairs.Load("test-rsa-encrypted")
	suite.Nil(kp)
	suite.Require().NotNil(failure)
	suite.Truef(failure.Type.Matches(keypairs.FailKeypairPassphrase), "unexpected failure type: %v", failure)
}

func (suite *KeypairLoadersTestSuite) TestFileFound_UnencryptedKeypairParseSuccess() {
	keyFile := suite.createConfigDirFile("test-rsa-success.key", 0600)
	keyFile.WriteString(suite.readTestFile("test-keypair.key"))
	suite.Require().NoError(keyFile.Close())

	kp, failure := keypairs.Load("test-rsa-success")
	suite.Require().Nil(failure)
	suite.NotNil(kp)
}

func Test_KeypairLoaders_TestSuite(t *testing.T) {
	suite.Run(t, new(KeypairLoadersTestSuite))
}

func (suite *KeypairLoadersTestSuite) createConfigDirFile(keyFile string, fileMode os.FileMode) *os.File {
	filename := filepath.Join(config.GetDataDir(), keyFile)
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileMode)
	suite.Require().NoError(err)
	return file
}

func (suite *KeypairLoadersTestSuite) readTestFile(fileName string) string {
	_, currentFile, _, _ := runtime.Caller(0)
	contents, err := ioutil.ReadFile(filepath.Join(filepath.Dir(currentFile), "testdata", fileName))
	suite.Require().NoError(err)
	return string(contents)
}
