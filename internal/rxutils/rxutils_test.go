package rxutils_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/rxutils"

	"github.com/ActiveState/cli/internal/testhelpers/suite"
)

type RxUtilsTestSuite struct {
	suite.Suite
}

func (suite *RxUtilsTestSuite) TestReplaceAllStringSubmatchFunc() {
	var tests = []struct {
		re          *regexp.Regexp
		input       string
		replacer    func([]string) string
		expected    string
		description string
	}{
		{
			re:          regexp.MustCompile("a(b)(c)(d)"),
			input:       "abcd",
			replacer:    func(groups []string) string { return strings.Join(groups, ",") },
			expected:    "abcd,b,c,d",
			description: "Correctly detects and replaces groups",
		},
		{
			re:          regexp.MustCompile("a(b)?(c)?(d)?"),
			input:       "abcd",
			replacer:    func(groups []string) string { return strings.Join(groups, ",") },
			expected:    "abcd,b,c,d",
			description: "Correctly detects and replaces optional groups",
		},
		{
			re:          regexp.MustCompile("a(b)?(c)?(d)?"),
			input:       "abc",
			replacer:    func(groups []string) string { return strings.Join(groups, ",") },
			expected:    "abc,b,c",
			description: "Correctly detects and replaces unmatched optional groups",
		},
		{
			re:          regexp.MustCompile("a(b)(c)(d)"),
			input:       "efgh",
			replacer:    func(groups []string) string { return strings.Join(groups, ",") },
			expected:    "efgh",
			description: "Doesn't do anything if there are no matches",
		},
	}

	for _, test := range tests {
		suite.Equal(test.expected, rxutils.ReplaceAllStringSubmatchFunc(test.re, test.input, test.replacer), test.description)
	}
}

func TestRxUtilsTestSuite(t *testing.T) {
	suite.Run(t, new(RxUtilsTestSuite))
}
