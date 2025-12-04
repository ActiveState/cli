package tagsuite

import (
	"os"
	"runtime"
	"strings"

	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
)

const (
	Activate        = "activate"
	Analytics       = "analytics"
	API             = "api"
	Artifacts       = "artifacts"
	Auth            = "auth"
	Automation      = "automation"
	Branches        = "branches"
	Bundle          = "bundle"
	BuildScripts    = "buildscripts"
	BuildInProgress = "buildinprogress"
	Carlisle        = "carlisle"
	Checkout        = "checkout"
	CLIDeploy       = "cli-deploy"
	Commit          = "commit"
	Compatibility   = "compatibility"
	Condition       = "condition"
	Config          = "config"
	Critical        = "critical"
	Cve             = "cve"
	Debug           = "debug"
	DeleteProjects  = "delete-uuid-projects"
	Deploy          = "deploy"
	Edit            = "edit"
	Environment     = "environment"
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
	Manifest        = "manifest"
	Migrations      = "migrations"
	Notifications   = "notifications"
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
	SolverV2        = "solver-v2"
	SolverV3        = "solver-v3"
	Switch          = "switch"
	Uninstall       = "uninstall"
	Upgrade         = "upgrade"
	Update          = "update"
	Use             = "use"
)

// Suite extends a testify suite Suite, such that tests allowing for dynamic skipping of tests
type Suite struct {
	suite.Suite
}

// ARM-supported tags for integration tests
var armSupportedTags = []string{
	Install,
	Installer,
	InstallScripts,
	Update,
}

// OnlyRunForTags skips a test unless one of the given tags is asked for.
func (suite *Suite) OnlyRunForTags(tags ...string) {
	// Skip tests on ARM64 if they don't have an ARM-supported tag
	if runtime.GOOS == "linux" && runtime.GOARCH == "arm64" {
		hasArmSupportedTag := false
		for _, tag := range tags {
			if funk.Contains(armSupportedTags, tag) {
				hasArmSupportedTag = true
				break
			}
		}
		if !hasArmSupportedTag {
			suite.T().Skipf("Skipping test on Linux/arm64 - only tags %s are supported", strings.Join(armSupportedTags, ", "))
		}
	}

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
