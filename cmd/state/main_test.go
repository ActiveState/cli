package main

import (
	"os"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/constants/version"
	depMock "github.com/ActiveState/cli/internal/deprecation/mock"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/spf13/viper"

	"github.com/stretchr/testify/suite"
)

type MainTestSuite struct {
	suite.Suite
}

func (suite *MainTestSuite) AfterTest(suiteName, testName string) {
	// Reset viper config so deprecation mock is always used
	viper.Reset()
}

func (suite *MainTestSuite) TestDeprecated() {
	mock := depMock.Init()
	defer mock.Close()
	mock.MockDeprecated()

	catcher := outputhelper.NewCatcher()
	exitCode, commandString, err := run([]string{""}, catcher.Outputer)
	suite.Require().NoError(err)
	suite.Require().Equal(0, exitCode, "Should exit with code 0, output: %s", catcher.CombinedOutput())
	suite.Assert().Equal("", commandString)

	if version.NumberIsProduction(constants.VersionNumber) {
		suite.Require().Contains(catcher.CombinedOutput(), output.StripColorCodes(locale.Tr("warn_deprecation", "")[0:50]))
	}
}

func (suite *MainTestSuite) TestExpired() {
	mock := depMock.Init()
	defer mock.Close()
	mock.MockExpired()

	catcher := outputhelper.NewCatcher()
	exitCode, commandString, err := run([]string{""}, catcher.Outputer)

	if version.NumberIsProduction(constants.VersionNumber) {
		suite.Require().Error(err)
		suite.Require().Equal(1, exitCode, "Should exit with code 1, output: %s", catcher.CombinedOutput())
		suite.Require().Contains(err.Error(), locale.Tr("err_deprecation", "")[0:50])
	} else {
		suite.Require().NoError(err)
		suite.Require().Equal(0, exitCode, "Should exit with code 0, output: %s", catcher.CombinedOutput())
	}
	suite.Assert().Equal("", commandString)
}

func (suite *MainTestSuite) TestOutputer() {
	{
		outputer, fail := initOutput(outputFlags{"", false, false}, "")
		suite.Require().NoError(fail.ToError())
		suite.Equal(output.PlainFormatName, outputer.Type(), "Returns Plain outputer")
	}

	{
		outputer, fail := initOutput(outputFlags{string(output.PlainFormatName), false, false}, "")
		suite.Require().NoError(fail.ToError())
		suite.Equal(output.PlainFormatName, outputer.Type(), "Returns Plain outputer")
	}

	{
		outputer, fail := initOutput(outputFlags{string(output.JSONFormatName), false, false}, "")
		suite.Require().NoError(fail.ToError())
		suite.Equal(output.JSONFormatName, outputer.Type(), "Returns JSON outputer")
	}

	{
		outputer, fail := initOutput(outputFlags{"", false, false}, string(output.JSONFormatName))
		suite.Require().NoError(fail.ToError())
		suite.Equal(output.JSONFormatName, outputer.Type(), "Returns JSON outputer")
	}

	{
		outputer, fail := initOutput(outputFlags{"", false, false}, string(output.EditorFormatName))
		suite.Require().NoError(fail.ToError())
		suite.Equal(output.EditorFormatName, outputer.Type(), "Returns JSON outputer")
	}

	{
		outputer, fail := initOutput(outputFlags{"", false, false}, string(output.EditorV0FormatName))
		suite.Require().NoError(fail.ToError())
		suite.Equal(output.EditorV0FormatName, outputer.Type(), "Returns JSON outputer")
	}
}

func (suite *MainTestSuite) TestParseOutputFlags() {
	suite.Equal(outputFlags{"plain", false, false}, parseOutputFlags([]string{"state", "foo", "-o", "plain"}))
	suite.Equal(outputFlags{"json", false, false}, parseOutputFlags([]string{"state", "foo", "--output", "json"}))
	suite.Equal(outputFlags{"json", false, false}, parseOutputFlags([]string{"state", "foo", "-o", "json"}))
	suite.Equal(outputFlags{"editor", false, false}, parseOutputFlags([]string{"state", "foo", "--output", "editor"}))
	suite.Equal(outputFlags{"editor.v0", false, false}, parseOutputFlags([]string{"state", "foo", "-o", "editor.v0"}))
	suite.Equal(outputFlags{"", true, false}, parseOutputFlags([]string{"state", "foo", "--mono"}))
	suite.Equal(outputFlags{"", false, true}, parseOutputFlags([]string{"state", "foo", "--confirm-exit-on-error"}))
}

func (suite *MainTestSuite) TestDisableColors() {
	monoFlags := outputFlags{Mono: true}
	nonMonoFlags := outputFlags{Mono: false}

	err := os.Setenv("NO_COLOR", "")
	suite.Require().NoError(err)
	suite.True(nonMonoFlags.DisableColor(false), "disable colors if NO_COLOR is set")
	err = os.Unsetenv("NO_COLOR")
	suite.Require().NoError(err)
	suite.False(nonMonoFlags.DisableColor(false), "do not disable colors by default")
	suite.True(monoFlags.DisableColor(false), "disable colors if --mono is set")
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}
