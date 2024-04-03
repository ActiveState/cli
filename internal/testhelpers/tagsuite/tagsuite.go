package tagsuite

import (
	"fmt"
	"os"
	"strings"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/errs"
)

const (
	Activate        = "activate"
	Analytics       = "analytics"
	Artifacts       = "artifacts"
	Auth            = "auth"
	Automation      = "automation"
	Branches        = "branches"
	Bundle          = "bundle"
	BuildScripts    = "buildscripts"
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
	Migrations      = "migrations"
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
	Reset           = "reset"
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

func (suite *Suite) NoError(err error, msgAndArgs ...interface{}) {
	if err == nil {
		return
	}

	errMsg := fmt.Sprintf("All error messages: %s", errs.JoinMessage(err))
	msgAndArgs = append(msgAndArgs, errMsg)
	suite.Suite.NoError(err, msgAndArgs...)
}

func (suite *Suite) Require() *Assertions {
	return &Assertions{suite.Suite.Require()}
}

type Assertions struct {
	*require.Assertions
}

func (a *Assertions) NoError(err error, msgAndArgs ...interface{}) {
	if err == nil {
		return
	}

	errMsg := fmt.Sprintf("All error messages: %s", errs.JoinMessage(err))
	msgAndArgs = append(msgAndArgs, errMsg)
	a.Assertions.NoError(err, msgAndArgs...)
}
