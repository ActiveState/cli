package integration

import (
	"regexp"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/stretchr/testify/suite"
)

type ExportIntegrationTestSuite struct {
	integration.Suite
}

func (suite *ExportIntegrationTestSuite) TestExport_EditorV0() {
	suite.LoginAsPersistentUser()
	suite.Spawn("export", "jwt", "--output", "editor.v0")
	suite.Wait()
	jwtRe := regexp.MustCompile("^[A-Za-z0-9-_=]+\\.[A-Za-z0-9-_=]+\\.?[A-Za-z0-9-_.+/=]*$")
	suite.True(jwtRe.Match([]byte(suite.TrimSpaceOutput())))
}

func TestExportIntegrationTestSuite(t *testing.T) {
	_ = suite.Run

	integration.RunParallel(t, new(ExportIntegrationTestSuite))
}
