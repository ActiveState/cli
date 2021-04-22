package tagsuite

import (
	"os"
	"strings"

	"github.com/stretchr/testify/suite"
	"github.com/thoas/go-funk"
)

const (
	Activate       = "activate"
	Alternative    = "alternative"
	Auth           = "auth"
	Branches       = "branches"
	Bundle         = "bundle"
	Carlisle       = "carlisle"
	CLIDeploy      = "cli-deploy"
	Condition      = "condition"
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
	Shell          = "shell"
	Exec           = "exec"
	Show           = "show"
	Update         = "update"
	VSCode         = "vscode"
)

// Suite extends a testify suite Suite, such that tests allowing for dynamic skipping of tests
type Suite struct {
	suite.Suite
}

// OnlyRunForTags skips a test unless one of the given tags is asked for.
func (suite *Suite) OnlyRunForTags(tags ...string) {
	setTagsString, _ := os.LookupEnv("TEST_SUITE_TAGS")

	// if no tags are defined, run the test
	if setTagsString == "" {
		return
	}
	setTags := strings.Split(setTagsString, ":")

	for _, tag := range tags {
		if funk.Contains(setTags, tag) {
			return
		}
	}

	suite.T().Skipf("Run only if any of the following tags are set: %s", strings.Join(tags, ", "))
}
