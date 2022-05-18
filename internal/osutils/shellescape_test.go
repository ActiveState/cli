package osutils_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/osutils"
	"github.com/stretchr/testify/suite"
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

func TestShellEscaperTestSuite(t *testing.T) {
	suite.Run(t, new(ShellEscaperTestSuite))
}
