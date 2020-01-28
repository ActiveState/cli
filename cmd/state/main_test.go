package main

import (
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
		outputer, fail := initOutputer([]string{"state", "foo"}, output.FormatUnset)
		suite.Require().NoError(fail.ToError())
		suite.IsType(&output.Plain{}, outputer, "Returns Plain outputer")
	}

	{
		format := output.FormatPlain
		outputer, fail := initOutputer([]string{"state", "foo", "--output", format.String()}, output.FormatUnset)
		suite.Require().NoError(fail.ToError())
		suite.IsType(&output.Plain{}, outputer, "Returns Plain outputer")
	}

	{
		format := output.FormatJSON
		outputer, fail := initOutputer([]string{"state", "foo", "--output", format.String()}, output.FormatUnset)
		suite.Require().NoError(fail.ToError())
		suite.IsType(&output.JSON{}, outputer, "Returns JSON outputer")
	}

	{
		outputer, fail := initOutputer([]string{"state", "foo"}, output.FormatJSON)
		suite.Require().NoError(fail.ToError())
		suite.IsType(&output.JSON{}, outputer, "Returns JSON outputer")
	}

	{
		outputer, fail := initOutputer([]string{"state", "foo"}, output.FormatEditorV0)
		suite.Require().NoError(fail.ToError())
		suite.IsType(&output.Plain{}, outputer, "Returns Plain outputer, as editor.v0 is not supported at this level")
	}
}

func (suite *MainTestSuite) TestParseOutputFlag() {
	suite.Equal(output.FormatPlain, parseOutputFlag([]string{"state", "foo", "-o", "plain"}))
	suite.Equal(output.FormatJSON, parseOutputFlag([]string{"state", "foo", "--output", "json"}))
	suite.Equal(output.FormatJSON, parseOutputFlag([]string{"state", "foo", "-o", "json"}))
	suite.Equal(output.FormatEditorV0, parseOutputFlag([]string{"state", "foo", "-o", "editor.v0"}))
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}
