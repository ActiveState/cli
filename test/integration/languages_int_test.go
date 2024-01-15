package integration

import (
	"regexp"
	"testing"
	"time"

	"github.com/ActiveState/termtest"
	goversion "github.com/hashicorp/go-version"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type LanguagesIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *LanguagesIntegrationTestSuite) TestLanguages_list() {
	suite.OnlyRunForTags(tagsuite.Languages)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/Languages", "1eb82b25-a564-42ee-a7d4-d51d2ea73cd5")

	cp := ts.Spawn("languages")
	cp.Expect("Name")
	cp.Expect("Python")
	cp.Expect("3.9.15")
	cp.ExpectExitCode(0)
}

func (suite *LanguagesIntegrationTestSuite) TestLanguages_listNoCommitID() {
	suite.OnlyRunForTags(tagsuite.Languages)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/Languages", "")

	cp := ts.Spawn("languages")
	cp.ExpectNotExitCode(0)
	ts.IgnoreLogErrors()
}

func (suite *LanguagesIntegrationTestSuite) TestLanguages_install() {
	suite.OnlyRunForTags(tagsuite.Languages)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/Languages", "1eb82b25-a564-42ee-a7d4-d51d2ea73cd5")

	ts.LoginAsPersistentUser()

	cp := ts.Spawn("languages")
	cp.Expect("Name")
	cp.Expect("Python")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("languages", "install", "python")
	cp.Expect("Language: python is already installed")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	cp = ts.Spawn("languages", "install", "python@3.9.16")
	cp.Expect("Language added: python@3.9.16")
	// This can take a little while
	cp.ExpectExitCode(0, termtest.OptExpectTimeout(60*time.Second))

	cp = ts.Spawn("languages")
	cp.Expect("Name")
	cp.Expect("Python")
	versionRe := regexp.MustCompile(`(\d+)\.(\d+).(\d+)`)
	cp.ExpectRe(versionRe.String())
	cp.ExpectExitCode(0)

	// assert that version number changed
	output := cp.Output()
	vs := versionRe.FindString(output)
	v, err := goversion.NewVersion(vs)
	suite.Require().NoError(err, "parsing version %s", vs)
	minVersion := goversion.Must(goversion.NewVersion("3.8.1"))
	suite.True(!v.LessThan(minVersion), "%v >= 3.8.1", v)
}

func (suite *LanguagesIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Languages, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Python3", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("languages", "-o", "json")
	cp.Expect(`[{"name":"Python","version":"Auto →"`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)

	cp = ts.Spawn("languages", "search", "--output", "json")
	cp.Expect(`[{"name":"perl","version":"Auto →"`)
	cp.ExpectExitCode(0)
	//AssertValidJSON(suite.T(), cp) // currently too big to fit in the terminal window for validation
}

func (suite *LanguagesIntegrationTestSuite) TestSearch() {
	suite.OnlyRunForTags(tagsuite.Languages)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("languages", "search")
	cp.Expect("perl")
	cp.Expect("5.32")
	cp.Expect("python")
	cp.Expect("3.11")
	cp.Expect("ruby")
	cp.Expect("3.2")
	cp.ExpectExitCode(0)
}

func TestLanguagesIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(LanguagesIntegrationTestSuite))
}
