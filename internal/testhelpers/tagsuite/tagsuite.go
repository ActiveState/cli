package tagsuite

import (
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/stretchr/testify/suite"
	"github.com/thoas/go-funk"
)

const (
	Activate       = "activate"
	Analytics      = "analytics"
	Alternative    = "alternative"
	Auth           = "auth"
	Branches       = "branches"
	Bundle         = "bundle"
	Carlisle       = "carlisle"
	CLIDeploy      = "cli-deploy"
	Condition      = "condition"
	Config         = "config"
	Critical       = "critical"
	Cve            = "cve"
	Deploy         = "deploy"
	Edit           = "edit"
	Error          = "error"
	Events         = "events"
	Export         = "export"
	ExitCode       = "exit-code"
	Fork           = "fork"
	Headless       = "headless"
	History        = "history"
	Import         = "import"
	Info           = "info"
	Init           = "init"
	InstallScripts = "install-scripts"
	Installer      = "installer"
	Interrupt      = "interrupt"
	JSON           = "json"
	Komodo         = "komodo"
	Languages      = "languages"
	MSI            = "msi"
	Organizations  = "organizations"
	Output         = "output"
	Package        = "package"
	Perl           = "perl"
	Platforms      = "platforms"
	Prepare        = "prepare"
	Pull           = "pull"
	Push           = "push"
	Python         = "python"
	Revert         = "revert"
	Run            = "run"
	Scripts        = "scripts"
	Secrets        = "secrets"
	Shell         = "shell"
	Exec          = "exec"
	Show          = "show"
	Uninstall     = "uninstall"
	Update        = "update"
	Use           = "use"
	VSCode        = "vscode"
	Performance   = "performance"
	Service       = "service"
	Deprecation   = "deprecation"
	Compatibility = "compatibility"
	Automation    = "automation"
)

// Suite extends a testify suite Suite, such that tests allowing for dynamic skipping of tests
type Suite struct {
	suite.Suite
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
