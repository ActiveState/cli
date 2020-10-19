package tagsuite

import (
	"os"
	"strings"

	"github.com/stretchr/testify/suite"
	"github.com/thoas/go-funk"
)

const (
	Activate      = "activate"
	Alternative   = "alternative"
	Auth          = "auth"
	CLIDeploy     = "cli-deploy"
	Condition     = "condition"
	Critical      = "critical"
	Deploy        = "deploy"
	Edit          = "edit"
	Error         = "error"
	Events        = "events"
	Export        = "export"
	Fork          = "fork"
	History       = "history"
	Init          = "init"
	Interrupt     = "interrupt"
	JSON          = "json"
	Komodo        = "komodo"
	Languages     = "languages"
	MSI           = "msi"
	Organizations = "organizations"
	Output        = "output"
	Package       = "package"
	Perl          = "perl"
	Platforms     = "platforms"
	Prepare       = "prepare"
	Pull          = "pull"
	Push          = "push"
	Python        = "python"
	Run           = "run"
	Scripts       = "scripts"
	Secrets       = "secrets"
	Shell         = "shell"
	Shim          = "shim"
	Show          = "show"
	Update        = "Update"
	VSCode        = "vscode"
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
