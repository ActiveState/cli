package main

import (
	"os"
	"testing"

	depMock "github.com/ActiveState/cli/internal/deprecation/mock"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"

	"github.com/stretchr/testify/suite"
)

type MainTestSuite struct {
	suite.Suite
}

func (suite *MainTestSuite) TestUnknownCommand() {
	exitCode, err := run([]string{"", "IdontExist"}, nil)
	suite.Contains(err.Error(), `unknown command "IdontExist"`)
	suite.Equal(1, exitCode)
}

func (suite *MainTestSuite) TestDeprecated() {
	mock := depMock.Init()
	defer mock.Close()
	mock.MockDeprecated()

	catcher := outputhelper.NewCatcher()
	exitCode, err := run([]string{""}, catcher.Outputer)
	suite.Require().NoError(err)
	suite.Require().Equal(0, exitCode, "Should exit with code 0, output: %s", catcher.CombinedOutput())
	suite.Require().Contains(catcher.Output(), output.StripColorCodes(locale.Tr("warn_deprecation", "")[0:50]))
}

func (suite *MainTestSuite) TestExpired() {
	mock := depMock.Init()
	defer mock.Close()
	mock.MockExpired()

	catcher := outputhelper.NewCatcher()
	exitCode, err := run([]string{""}, catcher.Outputer)
	suite.Require().NoError(err)
	suite.Require().Equal(0, exitCode, "Should exit with code 0, output: %s", catcher.CombinedOutput())
	suite.Require().Contains(catcher.ErrorOutput(), locale.Tr("err_deprecation", "")[0:50])
}

func (suite *MainTestSuite) TestOutputer() {
	{
		outputer, fail := initOutputer(outputFlags{"", false}, "")
		suite.Require().NoError(fail.ToError())
		suite.IsType(&output.Plain{}, outputer, "Returns Plain outputer")
	}

	{
		outputer, fail := initOutputer(outputFlags{string(output.PlainFormatName), false}, "")
		suite.Require().NoError(fail.ToError())
		suite.IsType(&output.Plain{}, outputer, "Returns Plain outputer")
	}

	{
		outputer, fail := initOutputer(outputFlags{string(output.JSONFormatName), false}, "")
		suite.Require().NoError(fail.ToError())
		suite.IsType(&output.JSON{}, outputer, "Returns JSON outputer")
	}

	{
		outputer, fail := initOutputer(outputFlags{"", false}, output.JSONFormatName)
		suite.Require().NoError(fail.ToError())
		suite.IsType(&output.JSON{}, outputer, "Returns JSON outputer")
	}

	{
		outputer, fail := initOutputer(outputFlags{"", false}, output.EditorFormatName)
		suite.Require().NoError(fail.ToError())
		suite.IsType(&output.JSON{}, outputer, "Returns JSON outputer")
	}

	{
		outputer, fail := initOutputer(outputFlags{"", false}, output.EditorV0FormatName)
		suite.Require().NoError(fail.ToError())
		suite.IsType(&output.JSON{}, outputer, "Returns JSON outputer")
	}
}

func (suite *MainTestSuite) TestParseOutputFlags() {
	suite.Equal(outputFlags{"plain", false}, parseOutputFlags([]string{"state", "foo", "-o", "plain"}))
	suite.Equal(outputFlags{"json", false}, parseOutputFlags([]string{"state", "foo", "--output", "json"}))
	suite.Equal(outputFlags{"json", false}, parseOutputFlags([]string{"state", "foo", "-o", "json"}))
	suite.Equal(outputFlags{"editor", false}, parseOutputFlags([]string{"state", "foo", "--output", "editor"}))
	suite.Equal(outputFlags{"editor.v0", false}, parseOutputFlags([]string{"state", "foo", "-o", "editor.v0"}))
	suite.Equal(outputFlags{"", true}, parseOutputFlags([]string{"state", "foo", "--mono"}))
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
