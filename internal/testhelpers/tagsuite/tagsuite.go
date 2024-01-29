package tagsuite

import (
	"os"
	"strings"

	"github.com/stretchr/testify/suite"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/condition"
)

const (
	Activate        = "activate"
	Analytics       = "analytics"
	Auth            = "auth"
	Automation      = "automation"
	Branches        = "branches"
	Bundle          = "bundle"
	Builds          = "builds"
	Carlisle        = "carlisle"
	Checkout        = "checkout"
	CLIDeploy       = "cli-deploy"
	Commit          = "commit"
	Compatibility   = "compatibility"
	Condition       = "condition"
	Config          = "config"
	Critical        = "critical"
	Cve             = "cve"
	DeleteProjects  = "delete-uuid-projects"
	Deploy          = "deploy"
	Edit            = "edit"
	Errors          = "error"
	Events          = "events"
	Exec            = "exec"
	Executor        = "executor"
	ExitCode        = "exit-code"
	Export          = "export"
	Fork            = "fork"
	HelloExample    = "hello_example"
	Help            = "help"
	History         = "history"
	Import          = "import"
	Info            = "info"
	Init            = "init"
	Install         = "install"
	Installer       = "installer"
	InstallScripts  = "install-scripts"
	Interrupt       = "interrupt"
	Invite          = "invite"
	JSON            = "json"
	Languages       = "languages"
	Messaging       = "messaging"
	Organizations   = "organizations"
	Output          = "output"
	Package         = "package"
	Performance     = "performance"
	Perl            = "perl"
	Platforms       = "platforms"
	Prepare         = "prepare"
	Progress        = "progress"
	Projects        = "projects"
	Publish         = "publish"
	Pull            = "pull"
	Push            = "push"
	Python          = "python"
	Refresh         = "refresh"
	RemoteInstaller = "remote-installer"
	Revert          = "revert"
	Run             = "run"
	Scripts         = "scripts"
	Secrets         = "secrets"
	Service         = "service"
	Shell           = "shell"
	Show            = "show"
	Switch          = "switch"
	Uninstall       = "uninstall"
	Update          = "update"
	Use             = "use"
)

// Suite extends a testify suite Suite, such that tests allowing for dynamic skipping of tests
type Suite struct {
	suite.Suite
}

// OnlyRunForTags skips a test unless one of the given tags is asked for.
func (suite *Suite) OnlyRunForTags(tags ...string) {
	setTagsString, _ := os.LookupEnv("TEST_SUITE_TAGS")

	setTags := strings.Split(strings.ToLower(setTagsString), ":")
	// if no tags are defined and we're not on CI; run the test
	if funk.Contains(setTags, "all") || (setTagsString == "" && !condition.OnCI()) {
		return
	}

	for _, tag := range tags {
		if funk.Contains(setTags, strings.ToLower(tag)) {
			return
		}
	}

	suite.T().Skipf("Run only if any of the following tags are set: %s", strings.Join(tags, ", "))
}
