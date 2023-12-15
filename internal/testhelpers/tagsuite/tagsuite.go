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
	Alternative     = "alternative"
	Auth            = "auth"
	Branches        = "branches"
	Bundle          = "bundle"
	Carlisle        = "carlisle"
	CLIDeploy       = "cli-deploy"
	Condition       = "condition"
	Config          = "config"
	Critical        = "critical"
	Cve             = "cve"
	DeleteProjects  = "delete-uuid-projects"
	Deploy          = "deploy"
	Edit            = "edit"
	Errors          = "error"
	Events          = "events"
	Export          = "export"
	ExitCode        = "exit-code"
	Fork            = "fork"
	Headless        = "headless"
	History         = "history"
	Import          = "import"
	Info            = "info"
	Init            = "init"
	InstallScripts  = "install-scripts"
	Installer       = "installer"
	Install         = "install"
	Invite          = "invite"
	RemoteInstaller = "remote-installer"
	Interrupt       = "interrupt"
	JSON            = "json"
	Languages       = "languages"
	MSI             = "msi"
	Organizations   = "organizations"
	Output          = "output"
	Package         = "package"
	Perl            = "perl"
	Platforms       = "platforms"
	Prepare         = "prepare"
	Progress        = "progress"
	Projects        = "projects"
	Projectfile     = "projectfile"
	Publish         = "publish"
	Pull            = "pull"
	Push            = "push"
	Python          = "python"
	Refresh         = "refresh"
	Revert          = "revert"
	Run             = "run"
	Scripts         = "scripts"
	Secrets         = "secrets"
	Switch          = "switch"
	Shell           = "shell"
	Exec            = "exec"
	Show            = "show"
	Uninstall       = "uninstall"
	Update          = "update"
	Use             = "use"
	Commit          = "commit"
	Performance     = "performance"
	Service         = "service"
	Executor        = "executor"
	Deprecation     = "deprecation"
	Compatibility   = "compatibility"
	Automation      = "automation"
	Checkout        = "checkout"
	OffInstall      = "offline-install"
	Help            = "help"
	Messaging       = "messaging"
	HelloExample    = "hello_example"
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
