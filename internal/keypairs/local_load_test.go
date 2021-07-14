package keypairs_test

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
)

type KeypairLocalLoadTestSuite struct {
	suite.Suite
	cfg keypairs.Configurable
}

func (suite *KeypairLocalLoadTestSuite) BeforeTest(suiteName, testName string) {
	var err error
	suite.cfg, err = config.New()
	suite.Require().NoError(err)
}

func (suite *KeypairLocalLoadTestSuite) AfterTest(suiteName, testName string) {
	suite.Require().NoError(suite.cfg.Close())
}

func (suite *KeypairLocalLoadTestSuite) TestNoKeyFileFound() {
	kp, err := keypairs.Load(suite.cfg, "test-no-such")
	suite.Nil(kp)
	suite.Error(err)
}

func (suite *KeypairLocalLoadTestSuite) assertTooPermissive(fileMode os.FileMode) {
	tmpKeyName := fmt.Sprintf("test-rsa-%0.4o", fileMode)
	keyFile := suite.createConfigDirFile(tmpKeyName+".key", fileMode)
	defer osutil.RemoveConfigFile(suite.cfg.ConfigPath(), tmpKeyName+".key")
	suite.Require().NoError(keyFile.Close())

	kp, err := keypairs.Load(suite.cfg, tmpKeyName)
	suite.Nil(kp)
	suite.Require().NotNil(err)
	suite.Error(err)
}

func (suite *KeypairLocalLoadTestSuite) TestFileFound_PermsTooPermissive() {
	if runtime.GOOS == "windows" {
		suite.T().Skip("Windows permissions work completely different, and we don't support it atm")
	}
	octalPerms := []os.FileMode{0640, 0650, 0660, 0670, 0604, 0605, 0606, 0607, 0700, 0500}
	for _, perm := range octalPerms {
		suite.assertTooPermissive(perm)
	}
}

func (suite *KeypairLocalLoadTestSuite) TestFileFound_KeypairParseError() {
	keyName := "test-rsa-parse-err"
	keyFile := suite.createConfigDirFile(keyName+".key", 0600)
	defer osutil.RemoveConfigFile(suite.cfg.ConfigPath(), keyName+".key")

	keyFile.WriteString("this will never parse")
	suite.Require().NoError(keyFile.Close())

	kp, err := keypairs.Load(suite.cfg, keyName)
	suite.Nil(kp)
	suite.Error(err)
}

func (suite *KeypairLocalLoadTestSuite) TestFileFound_EncryptedKeypairParseFailure() {
	keyName := "test-rsa-encrypted"
	keyFile := suite.createConfigDirFile(keyName+".key", 0600)
	defer osutil.RemoveConfigFile(suite.cfg.ConfigPath(), keyName+".key")

	keyFile.WriteString(suite.readTestFile("test-keypair-encrypted.key"))
	suite.Require().NoError(keyFile.Close())

	kp, err := keypairs.Load(suite.cfg, keyName)
	suite.Nil(kp)
	suite.Error(err)
}

func (suite *KeypairLocalLoadTestSuite) TestFileFound_UnencryptedKeypairParseSuccess() {
	keyName := "test-rsa-success"
	keyFile := suite.createConfigDirFile(keyName+".key", 0600)
	defer osutil.RemoveConfigFile(suite.cfg.ConfigPath(), keyName+".key")

	keyFile.WriteString(suite.readTestFile("test-keypair.key"))
	suite.Require().NoError(keyFile.Close())

	kp, err := keypairs.Load(suite.cfg, keyName)
	suite.NoError(err)
	suite.NotNil(kp)
}

func (suite *KeypairLocalLoadTestSuite) TestFileFound_WithDefaults() {
	keyName := constants.KeypairLocalFileName
	keyFile := suite.createConfigDirFile(keyName+".key", 0600)
	defer osutil.RemoveConfigFile(suite.cfg.ConfigPath(), keyName+".key")

	keyFile.WriteString(suite.readTestFile("test-keypair.key"))
	suite.Require().NoError(keyFile.Close())

	kp, err := keypairs.LoadWithDefaults(suite.cfg)
	suite.Require().Nil(err)
	suite.NotNil(kp)
}

func (suite *KeypairLocalLoadTestSuite) TestLoadWithDefaults_Override() {
	os.Setenv(constants.PrivateKeyEnvVarName, "nonce")
	defer os.Unsetenv(constants.PrivateKeyEnvVarName)

	kp, err := keypairs.LoadWithDefaults(suite.cfg)
	suite.Error(err)
	suite.Nil(kp)

	kprsa, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().NoError(err)

	os.Setenv(constants.PrivateKeyEnvVarName, kprsa.EncodePrivateKey())

	kp, err = keypairs.LoadWithDefaults(suite.cfg)
	suite.Require().NoError(err)
	suite.NotNil(kp)
}

func (suite *KeypairLocalLoadTestSuite) createConfigDirFile(keyFile string, fileMode os.FileMode) *os.File {
	file, err := osutil.CreateConfigFile(suite.cfg.ConfigPath(), keyFile, fileMode)
	suite.Require().NoError(err)
	return file
}

func (suite *KeypairLocalLoadTestSuite) readTestFile(fileName string) string {
	contents, err := osutil.ReadTestFile(fileName)
	suite.Require().NoError(err)
	return string(contents)
}

func Test_KeypairLocalLoad_TestSuite(t *testing.T) {
	suite.Run(t, new(KeypairLocalLoadTestSuite))
}
