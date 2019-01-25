package keypairs_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/keypairs"
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
	suite.Regexp("no such file or directory", err.Error())
}

func (suite *KeypairLocalDeleteTestSuite) TestWithDefaults_Success() {
	osutil.CopyTestFileToConfigDir("test-keypair.key", constants.KeypairLocalFileName+".key", 0600)

	failure := keypairs.DeleteWithDefaults()
	suite.Require().Nil(failure)

	fileInfo, err := osutil.StatConfigFile(constants.KeypairLocalFileName + ".key")
	suite.Require().Nil(fileInfo)
	suite.Regexp("no such file or directory", err.Error())
}

func Test_KeypairLocalDelete_TestSuite(t *testing.T) {
	suite.Run(t, new(KeypairLocalDeleteTestSuite))
}
