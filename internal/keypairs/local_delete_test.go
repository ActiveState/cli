package keypairs_test

import (
	"os"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/stretchr/testify/suite"
)

type KeypairLocalDeleteTestSuite struct {
	suite.Suite
}

func (suite *KeypairLocalDeleteTestSuite) TestNoKeyFileFound() {
	failure := keypairs.Delete("test-no-such")
	suite.Nil(failure)
}

func (suite *KeypairLocalDeleteTestSuite) Test_Success() {
	osutil.CopyTestFileToConfigDir("test-keypair.key", "custom-name.key", 0600)

	failure := keypairs.Delete("custom-name")
	suite.Require().Nil(failure)

	fileInfo, err := osutil.StatConfigFile("custom-name.key")
	suite.Require().Nil(fileInfo)
	if runtime.GOOS != "windows" {
		suite.Regexp("no such file or directory", err.Error())
	} else {
		suite.Regexp("The system cannot find the file specified", err.Error())
	}
}

func (suite *KeypairLocalLoadTestSuite) TestDelete_Override() {
	os.Setenv(constants.PrivateKeyEnvVarName, "some val")
	defer os.Unsetenv(constants.PrivateKeyEnvVarName)

	fail := keypairs.Delete("nonce")
	suite.Truef(fail.Type.Matches(keypairs.FailHasOverride), "unexpected failure type: %v", fail)
	suite.Contains(fail.Message, locale.T("keypairs_err_override_with_delete"))
}

func (suite *KeypairLocalDeleteTestSuite) TestWithDefaults_Success() {
	osutil.CopyTestFileToConfigDir("test-keypair.key", constants.KeypairLocalFileName+".key", 0600)

	failure := keypairs.DeleteWithDefaults()
	suite.Require().Nil(failure)

	fileInfo, err := osutil.StatConfigFile(constants.KeypairLocalFileName + ".key")
	suite.Require().Nil(fileInfo)
	if runtime.GOOS != "windows" {
		suite.Regexp("no such file or directory", err.Error())
	} else {
		suite.Regexp("The system cannot find the file specified", err.Error())
	}
}

func Test_KeypairLocalDelete_TestSuite(t *testing.T) {
	suite.Run(t, new(KeypairLocalDeleteTestSuite))
}
