package integration

import (
	"regexp"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/stretchr/testify/suite"
)

type ExportIntegrationTestSuite struct {
	suite.Suite
}

func (suite *ExportIntegrationTestSuite) TestExport_EditorV0() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()
	cp := ts.Spawn("export", "jwt", "--output", "editor.v0")
	cp.ExpectExitCode(0)
	jwtRe := regexp.MustCompile("^[A-Za-z0-9-_=]+\\.[A-Za-z0-9-_=]+\\.?[A-Za-z0-9-_.+/=]*$")
	suite.True(jwtRe.Match([]byte(cp.TrimmedSnapshot())))
}

func TestExportIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ExportIntegrationTestSuite))
}
