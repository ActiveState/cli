package main

import (
	"testing"

	"github.com/kami-zh/go-capturer"

	depMock "github.com/ActiveState/cli/internal/deprecation/mock"
	"github.com/ActiveState/cli/internal/locale"

	"github.com/stretchr/testify/suite"
)

type MainTestSuite struct {
	suite.Suite
}

func (suite *MainTestSuite) TestPanicCaught() {
	out, exitCode := run([]string{"", "IdontExist"})
	suite.Contains(out, locale.T("err_cmdtree"))
	suite.Contains(out, `unknown command "IdontExist"`)
	suite.Equal(1, exitCode)
}

func (suite *MainTestSuite) TestDeprecated() {
	mock := depMock.Init()
	defer mock.Close()
	mock.MockDeprecated()

	var exitCode = -1
	out := capturer.CaptureOutput(func() {
		var err error
		err, exitCode = run([]string{""})
		suite.Require().NoError(err)
	})
	suite.Require().Equal(0, exitCode, "Should exit with code 0, output: %s", out)
	suite.Require().Contains(out, locale.Tr("warn_deprecation", "")[0:50])
}

func (suite *MainTestSuite) TestExpired() {
	mock := depMock.Init()
	defer mock.Close()
	mock.MockExpired()

	var exitCode = -1
	out := capturer.CaptureOutput(func() {
		var err error
		err, exitCode = run([]string{""})
		suite.Require().NoError(err)
	})
	suite.Require().Equal(0, exitCode, "Should exit with code 0, output: %s", out)
	suite.Require().Contains(out, locale.Tr("err_deprecation", "")[0:50])
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}
