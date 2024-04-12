package main

import (
	"os"
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
)

type MainTestSuite struct {
	suite.Suite
}

func (suite *MainTestSuite) TestOutputer() {
	{
		outputer, err := initOutput(outputFlags{"", false, false}, "", "")
		suite.Require().NoError(err, errs.JoinMessage(err))
		suite.Equal(output.PlainFormatName, outputer.Type(), "Returns Plain outputer")
	}

	{
		outputer, err := initOutput(outputFlags{string(output.PlainFormatName), false, false}, "", "")
		suite.Require().NoError(err)
		suite.Equal(output.PlainFormatName, outputer.Type(), "Returns Plain outputer")
	}

	{
		outputer, err := initOutput(outputFlags{string(output.JSONFormatName), false, false}, "", "")
		suite.Require().NoError(err)
		suite.Equal(output.JSONFormatName, outputer.Type(), "Returns JSON outputer")
	}

	{
		outputer, err := initOutput(outputFlags{"", false, false}, string(output.JSONFormatName), "")
		suite.Require().NoError(err)
		suite.Equal(output.JSONFormatName, outputer.Type(), "Returns JSON outputer")
	}

	{
		outputer, err := initOutput(outputFlags{"", false, false}, string(output.EditorFormatName), "")
		suite.Require().NoError(err)
		suite.Equal(output.EditorFormatName, outputer.Type(), "Returns JSON outputer")
	}
}

func (suite *MainTestSuite) TestParseOutputFlags() {
	suite.Equal(outputFlags{"plain", false, false}, parseOutputFlags([]string{"state", "foo", "-o", "plain"}))
	suite.Equal(outputFlags{"json", false, false}, parseOutputFlags([]string{"state", "foo", "--output", "json"}))
	suite.Equal(outputFlags{"json", false, false}, parseOutputFlags([]string{"state", "foo", "-o", "json"}))
	suite.Equal(outputFlags{"editor", false, false}, parseOutputFlags([]string{"state", "foo", "--output", "editor"}))
	suite.Equal(outputFlags{"", true, false}, parseOutputFlags([]string{"state", "foo", "--mono"}))
	suite.Equal(outputFlags{"", false, true}, parseOutputFlags([]string{"state", "foo", "--non-interactive"}))
	suite.Equal(outputFlags{"", false, true}, parseOutputFlags([]string{"state", "foo", "-n"}))
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
