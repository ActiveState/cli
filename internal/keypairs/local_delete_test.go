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

type KeypairLocalDeleteTestSuite struct {
	suite.Suite
	cfg keypairs.Configurable
}

func (suite *KeypairLocalDeleteTestSuite) BeforeTest(suiteName, testName string) {
	var err error
	suite.cfg, err = config.New()
	suite.Require().NoError(err)
}

func (suite *KeypairLocalDeleteTestSuite) AfterTest(suiteName, testName string) {
	suite.Require().NoError(suite.cfg.Close())
}

func (suite *KeypairLocalDeleteTestSuite) TestNoKeyFileFound() {
	err := keypairs.Delete(suite.cfg, "test-no-such")
	suite.Nil(err)
}

func (suite *KeypairLocalDeleteTestSuite) Test_Success() {
	cfg, err := config.New()
	suite.Require().NoError(err)
	defer suite.Require().NoError(cfg.Close())
	osutil.CopyTestFileToConfigDir(cfg.ConfigPath(), "test-keypair.key", "custom-name.key", 0600)

	err = keypairs.Delete(suite.cfg, "custom-name")
	suite.Require().Nil(err)

	fileInfo, err := osutil.StatConfigFile(cfg.ConfigPath(), "custom-name.key")
	suite.Require().Nil(fileInfo)
	if runtime.GOOS != "windows" {
		suite.Regexp("no such file or directory", err.Error())
	} else {
		suite.Regexp("The system cannot find the file specified", err.Error())
	}
}

func (suite *KeypairLocalDeleteTestSuite) TestWithDefaults_Success() {
	cfg, err := config.New()
	suite.Require().NoError(err)
	defer suite.Require().NoError(cfg.Close())
	osutil.CopyTestFileToConfigDir(cfg.ConfigPath(), "test-keypair.key", constants.KeypairLocalFileName+".key", 0600)

	err = keypairs.DeleteWithDefaults(suite.cfg)
	suite.Require().Nil(err)

	fileInfo, err := osutil.StatConfigFile(cfg.ConfigPath(), constants.KeypairLocalFileName+".key")
	suite.Require().Nil(fileInfo)
	if runtime.GOOS != "windows" {
		suite.Regexp("no such file or directory", err.Error())
	} else {
		suite.Regexp("The system cannot find the file specified", err.Error())
	}
}

func (suite *KeypairLocalDeleteTestSuite) TestDeleteWithDefaults_Override() {
	os.Setenv(constants.PrivateKeyEnvVarName, "some val")
	defer os.Unsetenv(constants.PrivateKeyEnvVarName)

	err := keypairs.DeleteWithDefaults(suite.cfg)
	suite.Require().NotNil(err)
	suite.Error(err)
}

func Test_KeypairLocalDelete_TestSuite(t *testing.T) {
	suite.Run(t, new(KeypairLocalDeleteTestSuite))
}
