package keypairs_test

import (
	"os"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/stretchr/testify/suite"
)

type KeypairLocalSaveTestSuite struct {
	suite.Suite
}

func (suite *KeypairLocalSaveTestSuite) TestSave_Success() {
	kp, failure := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(failure)

	failure = keypairs.Save(kp, "save-testing")
	suite.Require().Nil(failure)

	kp2, failure := keypairs.Load("save-testing")
	suite.Require().Nil(failure)
	suite.Equal(kp, kp2)

	fileInfo := suite.statConfigDirFile("save-testing.key")
	if runtime.GOOS != "windows" {
		suite.Equal(os.FileMode(0600), fileInfo.Mode())
	}
}

func (suite *KeypairLocalLoadTestSuite) TestSave_Override() {
	os.Setenv(constants.PrivateKeyEnvVarName, "some val")
	defer os.Unsetenv(constants.PrivateKeyEnvVarName)

	kp, failure := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(failure)

	fail := keypairs.Save(kp, "nonce")
	suite.Truef(fail.Type.Matches(keypairs.FailHasOverride), "unexpected failure type: %v", fail)
}

func (suite *KeypairLocalLoadTestSuite) TestSaveWithDefaults_Override() {
	os.Setenv(constants.PrivateKeyEnvVarName, "some val")
	defer os.Unsetenv(constants.PrivateKeyEnvVarName)

	kp, failure := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(failure)

	fail := keypairs.SaveWithDefaults(kp)
	suite.Truef(fail.Type.Matches(keypairs.FailHasOverride), "unexpected failure type: %v", fail)
}

func (suite *KeypairLocalSaveTestSuite) TestSaveWithDefaults_Success() {
	kp, failure := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(failure)

	failure = keypairs.SaveWithDefaults(kp)
	suite.Require().Nil(failure)

	kp2, failure := keypairs.Load(constants.KeypairLocalFileName)
	suite.Require().Nil(failure)
	suite.Equal(kp, kp2)

	fileInfo := suite.statConfigDirFile(constants.KeypairLocalFileName + ".key")
	if runtime.GOOS != "windows" {
		suite.Equal(os.FileMode(0600), fileInfo.Mode())
	}
}

func (suite *KeypairLocalSaveTestSuite) statConfigDirFile(keyFile string) os.FileInfo {
	keyFileStat, err := osutil.StatConfigFile(keyFile)
	suite.Require().NoError(err)
	return keyFileStat
}

func Test_KeypairLocalSave_TestSuite(t *testing.T) {
	suite.Run(t, new(KeypairLocalSaveTestSuite))
}
