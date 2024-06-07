package osutils_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
)

type ShellEscaperTestSuite struct {
	suite.Suite
}

func (suite *ShellEscaperTestSuite) TestBashEscaper() {
	escaper := osutils.NewBashEscaper()
	suite.Equal(`quoted`, escaper.Quote(`quoted`))
	suite.Equal(`"\"quoted\""`, escaper.Quote(`"quoted"`))
	suite.Equal(`"'quoted'"`, escaper.Quote(`'quoted'`))
	suite.Equal(`"quoted\nquote"`, escaper.Quote("quoted\nquote"))
	suite.Equal(`"quote\\"`, escaper.Quote(`quote\`))
	suite.Equal(`"quote\"quote"`, escaper.Quote(`quote"quote`))
	suite.Equal(`"\$FOO"`, escaper.Quote(`$FOO`))
}

func (suite *ShellEscaperTestSuite) TestBatchEscaper() {
	escaper := osutils.NewBatchEscaper()
	suite.Equal(`quoted`, escaper.Quote(`quoted`))
	suite.Equal(`"""quoted"""`, escaper.Quote(`"quoted"`))
	suite.Equal(`"'quoted'"`, escaper.Quote(`'quoted'`))
	suite.Equal(`"quoted\nquote"`, escaper.Quote("quoted\nquote"))
	suite.Equal(`"quote\"`, escaper.Quote(`quote\`))
	suite.Equal(`"quote""quote"`, escaper.Quote(`quote"quote`))
}

func (suite *ShellEscaperTestSuite) TestCmdEscaper() {
	escaper := osutils.NewCmdEscaper()
	suite.Equal(`quoted`, escaper.Quote(`quoted`))
	suite.Equal(`"quoted quote"`, escaper.Quote(`quoted quote`))
	suite.Equal(`project/org`, escaper.Quote(`project/org`))
	suite.Equal(`cmd.exe`, escaper.Quote(`cmd.exe`))
}

func TestShellEscaperTestSuite(t *testing.T) {
	suite.Run(t, new(ShellEscaperTestSuite))
}
