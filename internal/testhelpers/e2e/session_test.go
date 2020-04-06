package e2e_test

import (
	"os"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type E2eSessionTestSuite struct {
	suite.Suite
}

func (suite *E2eSessionTestSuite) TestE2eSession() {
	cases := []struct {
		name     string
		args     []string
		exitCode int
	}{
		{"expect a string", []string{}, 0},
		{"exit 1", []string{"-exit1"}, 1},
		{"with filled buffer", []string{"-fill-buffer"}, 0},
		{"stuttering", []string{"-stutter"}, 0},
	}

	for _, c := range cases {
		suite.Run(c.name, func() {
			// create a new test-session
			ts := e2e.New(suite.T(), false)
			defer ts.Close()
			wd, err := os.Getwd()
			require.NoError(suite.T(), err)

			cp := ts.SpawnCmdWithOpts(
				"go",
				e2e.WithArgs(append([]string{"run", "./session-tester"}, c.args...)...),
				e2e.WithWorkDirectory(wd),
			)
			cp.Expect("an expected string", 1*time.Second)
			cp.ExpectExitCode(c.exitCode, 1*time.Second)
		})
	}
}

func TestE2eSessionTestSuite(t *testing.T) {
	suite.Run(t, new(E2eSessionTestSuite))
}
