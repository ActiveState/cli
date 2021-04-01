package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/colorize"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/constants/version"
	depMock "github.com/ActiveState/cli/internal/deprecation/mock"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"

	"github.com/stretchr/testify/suite"
)

type MainTestSuite struct {
	suite.Suite
}

func (suite *MainTestSuite) cleanDeprecationFile() {
	cfg, err := config.Get()
	suite.Require().NoError(err)
	// force fetching of deprecation info
	err = os.Remove(filepath.Join(cfg.ConfigPath(), "deprecation.json"))
	if err != nil && !os.IsNotExist(err) {
		suite.T().Logf("Could not remove deprecation file")
	}
}

func (suite *MainTestSuite) TestDeprecated() {
	suite.cleanDeprecationFile()
	mock := depMock.Init()
	defer mock.Close()
	mock.MockDeprecated()

	catcher := outputhelper.NewCatcher()
	err := run([]string{""}, true, catcher.Outputer)
	exitCode := errs.UnwrapExitCode(err)
	suite.Require().NoError(err)
	suite.Require().Equal(0, exitCode, "Should exit with code 0, output: %s", catcher.CombinedOutput())

	if version.NumberIsProduction(constants.VersionNumber) {
		suite.Require().Contains(catcher.CombinedOutput(), colorize.StripColorCodes(locale.Tr("warn_deprecation", "")[0:50]))
	}
}

func (suite *MainTestSuite) TestExpired() {
	suite.cleanDeprecationFile()

	mock := depMock.Init()
	defer mock.Close()
	mock.MockExpired()

	catcher := outputhelper.NewCatcher()
	err := run([]string{""}, true, catcher.Outputer)
	exitCode := errs.UnwrapExitCode(err)

	if version.NumberIsProduction(constants.VersionNumber) {
		suite.Require().Error(err)
		suite.Require().Equal(1, exitCode, "Should exit with code 1, output: %s", catcher.CombinedOutput())
		suite.Require().Contains(err.Error(), locale.Tr("err_deprecation", "")[0:50])
	} else {
		suite.Require().NoError(err)
		suite.Require().Equal(0, exitCode, "Should exit with code 0, output: %s", catcher.CombinedOutput())
	}
}

func (suite *MainTestSuite) TestOutputer() {
	{
		outputer, err := initOutput(outputFlags{"", false, false, false}, "")
		suite.Require().NoError(err, errs.Join(err, "\n").Error())
		suite.Equal(output.PlainFormatName, outputer.Type(), "Returns Plain outputer")
	}

	{
		outputer, err := initOutput(outputFlags{string(output.PlainFormatName), false, false, false}, "")
		suite.Require().NoError(err)
		suite.Equal(output.PlainFormatName, outputer.Type(), "Returns Plain outputer")
	}

	{
		outputer, err := initOutput(outputFlags{string(output.JSONFormatName), false, false, false}, "")
		suite.Require().NoError(err)
		suite.Equal(output.JSONFormatName, outputer.Type(), "Returns JSON outputer")
	}

	{
		outputer, err := initOutput(outputFlags{"", false, false, false}, string(output.JSONFormatName))
		suite.Require().NoError(err)
		suite.Equal(output.JSONFormatName, outputer.Type(), "Returns JSON outputer")
	}

	{
		outputer, err := initOutput(outputFlags{"", false, false, false}, string(output.EditorFormatName))
		suite.Require().NoError(err)
		suite.Equal(output.EditorFormatName, outputer.Type(), "Returns JSON outputer")
	}

	{
		outputer, err := initOutput(outputFlags{"", false, false, false}, string(output.EditorV0FormatName))
		suite.Require().NoError(err)
		suite.Equal(output.EditorV0FormatName, outputer.Type(), "Returns JSON outputer")
	}
}

func (suite *MainTestSuite) TestParseOutputFlags() {
	suite.Equal(outputFlags{"plain", false, false, false}, parseOutputFlags([]string{"state", "foo", "-o", "plain"}))
	suite.Equal(outputFlags{"json", false, false, false}, parseOutputFlags([]string{"state", "foo", "--output", "json"}))
	suite.Equal(outputFlags{"json", false, false, false}, parseOutputFlags([]string{"state", "foo", "-o", "json"}))
	suite.Equal(outputFlags{"editor", false, false, false}, parseOutputFlags([]string{"state", "foo", "--output", "editor"}))
	suite.Equal(outputFlags{"editor.v0", false, false, false}, parseOutputFlags([]string{"state", "foo", "-o", "editor.v0"}))
	suite.Equal(outputFlags{"", true, false, false}, parseOutputFlags([]string{"state", "foo", "--mono"}))
	suite.Equal(outputFlags{"", false, true, false}, parseOutputFlags([]string{"state", "foo", "--confirm-exit-on-error"}))
	suite.Equal(outputFlags{"", false, false, true}, parseOutputFlags([]string{"state", "foo", "--non-interactive"}))
	suite.Equal(outputFlags{"", false, false, true}, parseOutputFlags([]string{"state", "foo", "-n"}))
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
