package main

import (
	"testing"

	depMock "github.com/ActiveState/cli/internal/deprecation/mock"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	"github.com/kami-zh/go-capturer"

	"github.com/stretchr/testify/suite"
)

type MainTestSuite struct {
	suite.Suite
}

func (suite *MainTestSuite) TestPanicCaught() {
	exitCode := -1
	exiter := func(code int) {
		if exitCode == -1 {
			// The first call to exit is cause we're running cobra with `go test` args
			// the second is called from the panic defer, which shouldn't panic again
			exitCode = code
			panic("Exit")
		}
	}
	out := capturer.CaptureOutput(func() {
		runAndExit([]string{"IdontExist"}, exiter)
	})
	suite.Contains(out, locale.T("err_main_panic"))
	suite.Contains(out, `unknown command "IdontExist"`)
	suite.Equal(1, exitCode)
}

func (suite *MainTestSuite) TestDeprecated() {
	mock := depMock.Init()
	defer mock.Close()
	mock.MockDeprecated()

	ex := exiter.New()
	out, code := ex.Capture(func() {
		runAndExit([]string{}, ex.Exit)
	})
	suite.Require().Equal(0, code)
	suite.Require().Contains(out, locale.Tr("warn_deprecation", "")[0:50])
}

func (suite *MainTestSuite) TestExpired() {
	mock := depMock.Init()
	defer mock.Close()
	mock.MockExpired()

	ex := exiter.New()
	out, code := ex.Capture(func() {
		runAndExit([]string{}, ex.Exit)
	})
	suite.Require().Equal(0, code)
	suite.Require().Contains(out, locale.Tr("err_deprecation", "")[0:50])
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}
