package tagsuite

import (
	"os"
	"strings"

	"github.com/stretchr/testify/suite"
	"github.com/thoas/go-funk"
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
