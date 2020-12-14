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

type KeypairLocalDeleteTestSuite struct {
	suite.Suite
}

func (suite *KeypairLocalDeleteTestSuite) TestNoKeyFileFound() {
	err := keypairs.Delete("test-no-such")
	suite.Nil(err)
}

func (suite *KeypairLocalDeleteTestSuite) Test_Success() {
	osutil.CopyTestFileToConfigDir("test-keypair.key", "custom-name.key", 0600)

	err := keypairs.Delete("custom-name")
	suite.Require().Nil(err)

	fileInfo, err := osutil.StatConfigFile("custom-name.key")
	suite.Require().Nil(fileInfo)
	if runtime.GOOS != "windows" {
		suite.Regexp("no such file or directory", err.Error())
	} else {
		suite.Regexp("The system cannot find the file specified", err.Error())
	}
}

func (suite *KeypairLocalDeleteTestSuite) TestWithDefaults_Success() {
	osutil.CopyTestFileToConfigDir("test-keypair.key", constants.KeypairLocalFileName+".key", 0600)

	err := keypairs.DeleteWithDefaults()
	suite.Require().Nil(err)

	fileInfo, err := osutil.StatConfigFile(constants.KeypairLocalFileName + ".key")
	suite.Require().Nil(fileInfo)
	if runtime.GOOS != "windows" {
		suite.Regexp("no such file or directory", err.Error())
	} else {
		suite.Regexp("The system cannot find the file specified", err.Error())
	}
}

func (suite *KeypairLocalLoadTestSuite) TestDeleteWithDefaults_Override() {
	os.Setenv(constants.PrivateKeyEnvVarName, "some val")
	defer os.Unsetenv(constants.PrivateKeyEnvVarName)

	err := keypairs.DeleteWithDefaults()
	suite.Require().NotNil(err)
	suite.Error(err)
}

func Test_KeypairLocalDelete_TestSuite(t *testing.T) {
	suite.Run(t, new(KeypairLocalDeleteTestSuite))
}
