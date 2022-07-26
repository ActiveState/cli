package testsuite

import (
	"fmt"
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/stretchr/testify/suite"
	"github.com/thoas/go-funk"
)

const (
	TagActivate       = "activate"
	TagAnalytics      = "analytics"
	TagAlternative    = "alternative"
	TagAuth           = "auth"
	TagBranches       = "branches"
	TagBundle         = "bundle"
	TagCarlisle       = "carlisle"
	TagCLIDeploy      = "cli-deploy"
	TagCondition      = "condition"
	TagConfig         = "config"
	TagCritical       = "critical"
	TagCve            = "cve"
	TagDeploy         = "deploy"
	TagEdit           = "edit"
	TagError          = "error"
	TagEvents         = "events"
	TagExport         = "export"
	TagExitCode       = "exit-code"
	TagFork           = "fork"
	TagHeadless       = "headless"
	TagHistory        = "history"
	TagImport         = "import"
	TagInfo           = "info"
	TagInit           = "init"
	TagInstallScripts = "install-scripts"
	TagInstaller      = "installer"
	TagInterrupt      = "interrupt"
	TagJSON           = "json"
	TagKomodo         = "komodo"
	TagLanguages      = "languages"
	TagMSI            = "msi"
	TagOrganizations  = "organizations"
	TagOutput         = "output"
	TagPackage        = "package"
	TagPerl           = "perl"
	TagPlatforms      = "platforms"
	TagPrepare        = "prepare"
	TagPull           = "pull"
	TagPush           = "push"
	TagPython         = "python"
	TagRevert         = "revert"
	TagRun            = "run"
	TagScripts        = "scripts"
	TagSecrets        = "secrets"
	TagShell          = "shell"
	TagExec           = "exec"
	TagShow           = "show"
	TagUninstall      = "uninstall"
	TagUpdate         = "update"
	TagUse            = "use"
	TagVSCode         = "vscode"
	TagPerformance    = "performance"
	TagService        = "service"
	TagDeprecation    = "deprecation"
	TagCompatibility  = "compatibility"
	TagAutomation     = "automation"
)

// Suite extends a testify suite Suite, such that tests allowing for dynamic skipping of tests
type Suite struct {
	suite.Suite
	*e2e.Session
}

func New() *Suite {
	return &Suite{}
}

func (suite *Suite) BeforeTest(string, string) {
	suite.Session = e2e.New(suite.T(), false)
}

func (suite *Suite) AfterTest(string, string) {
	if err := suite.Session.Close(); err != nil {
		fmt.Printf("Error closing session: %s\n", errs.JoinMessage(err))
	}
}

// OnlyRunForTags skips a test unless one of the given tags is asked for.
func (suite *Suite) OnlyRunForTags(tags ...string) {
	setTagsString, _ := os.LookupEnv("TEST_SUITE_TAGS")

	setTags := strings.Split(setTagsString, ":")
	// if no tags are defined and we're not on CI; run the test
	if funk.Contains(setTags, "all") || (setTagsString == "" && !condition.OnCI()) {
		return
	}

	for _, tag := range tags {
		if funk.Contains(setTags, tag) {
			return
		}
	}

	suite.T().Skipf("Run only if any of the following tags are set: %s", strings.Join(tags, ", "))
}
