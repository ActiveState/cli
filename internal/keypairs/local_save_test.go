package keypairs_test

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
)

type KeypairLocalSaveTestSuite struct {
	suite.Suite
}

func (suite *KeypairLocalSaveTestSuite) TestSave_Success() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(err)

	err = keypairs.Save(kp, "save-testing")
	suite.Require().Nil(err)

	kp2, err := keypairs.Load("save-testing")
	suite.Require().Nil(err)
	suite.Equal(kp, kp2)

	fileInfo := suite.statConfigDirFile("save-testing.key")
	if runtime.GOOS != "windows" {
		suite.Equal(os.FileMode(0600), fileInfo.Mode())
	}
}

func (suite *KeypairLocalSaveTestSuite) TestSaveWithDefaults_Success() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(err)

	err = keypairs.SaveWithDefaults(kp)
	suite.Require().Nil(err)

	kp2, err := keypairs.Load(constants.KeypairLocalFileName)
	suite.Require().Nil(err)
	suite.Equal(kp, kp2)

	fileInfo := suite.statConfigDirFile(constants.KeypairLocalFileName + ".key")
	if runtime.GOOS != "windows" {
		suite.Equal(os.FileMode(0600), fileInfo.Mode())
	}
}

func (suite *KeypairLocalLoadTestSuite) TestSaveWithDefaults_Override() {
	os.Setenv(constants.PrivateKeyEnvVarName, "some val")
	defer os.Unsetenv(constants.PrivateKeyEnvVarName)

	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(err)

	err = keypairs.SaveWithDefaults(kp)
	suite.Require().NotNil(err)
	suite.Error(err)
}

func (suite *KeypairLocalSaveTestSuite) statConfigDirFile(keyFile string) os.FileInfo {
	keyFileStat, err := osutil.StatConfigFile(keyFile)
	suite.Require().NoError(err)
	return keyFileStat
}

func Test_KeypairLocalSave_TestSuite(t *testing.T) {
	suite.Run(t, new(KeypairLocalSaveTestSuite))
}
