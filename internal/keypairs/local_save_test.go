package keypairs_test

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
)

type KeypairLocalSaveTestSuite struct {
	suite.Suite
	cfg keypairs.Configurable
}

func (suite *KeypairLocalSaveTestSuite) BeforeTest(suiteName, testName string) {
	var err error
	suite.cfg, err = config.New()
	suite.Require().NoError(err)
}

func (suite *KeypairLocalSaveTestSuite) AfterTest(suiteName, testName string) {
	suite.Require().NoError(suite.cfg.Close())
}

func (suite *KeypairLocalSaveTestSuite) TestSave_Success() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(err)

	err = keypairs.Save(suite.cfg, kp, "save-testing")
	suite.Require().Nil(err)

	kp2, err := keypairs.Load(suite.cfg, "save-testing")
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

	err = keypairs.SaveWithDefaults(suite.cfg, kp)
	suite.Require().Nil(err)

	kp2, err := keypairs.Load(suite.cfg, constants.KeypairLocalFileName)
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

	err = keypairs.SaveWithDefaults(suite.cfg, kp)
	suite.Require().NotNil(err)
	suite.Error(err)
}

func (suite *KeypairLocalSaveTestSuite) statConfigDirFile(keyFile string) os.FileInfo {
	cfg, err := config.New()
	suite.Require().NoError(err)
	defer func() { suite.Require().NoError(cfg.Close()) }()
	keyFileStat, err := osutil.StatConfigFile(cfg.ConfigPath(), keyFile)
	suite.Require().NoError(err)
	return keyFileStat
}

func Test_KeypairLocalSave_TestSuite(t *testing.T) {
	suite.Run(t, new(KeypairLocalSaveTestSuite))
}
