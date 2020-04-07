package e2e_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type E2eSessionTestSuite struct {
	suite.Suite
	sessionTester string
	tmpDir        string
}

func (suite *E2eSessionTestSuite) SetupSuite() {
	dir, err := ioutil.TempDir("", "")
	require.NoError(suite.T(), err)
	suite.tmpDir = dir

	suite.sessionTester = filepath.Join(dir, "sessionTester")
	fmt.Println(suite.sessionTester)

	cmd := exec.Command("go", "build", "-o", suite.sessionTester, "./session-tester")
	err = cmd.Start()
	suite.Require().NoError(err)
	err = cmd.Wait()
	suite.Require().NoError(err)
	suite.Require().Equal(0, cmd.ProcessState.ExitCode())
}

func (suite *E2eSessionTestSuite) TearDownSuite() {
	err := os.RemoveAll(suite.tmpDir)
	suite.Require().NoError(err)
}

func (suite *E2eSessionTestSuite) TestE2eSession() {
	// terminal size is 80*30 (one newline at end of stream)
	fillbufferOutput := string(bytes.Repeat([]byte("a"), 80*29))
	// match at least two consecutive space character
	spaceRe := regexp.MustCompile("  +")
	cases := []struct {
		name           string
		args           []string
		exitCode       int
		terminalOutput string
	}{
		{"expect a string", []string{}, 0, "an expected string"},
		{"exit 1", []string{"-exit1"}, 1, "an expected string"},
		{"with filled buffer", []string{"-fill-buffer"}, 0, fillbufferOutput},
		{"stuttering", []string{"-stutter"}, 0, "an expected string stuttered 1 times stuttered 2 times stuttered 3 times stuttered 4 times stuttered 5 times"},
	}

	for _, c := range cases {
		suite.Run(c.name, func() {
			// create a new test-session
			ts := e2e.New(suite.T(), false)
			defer ts.Close()

			cp := ts.SpawnCmd(suite.sessionTester, c.args...)
			cp.Expect("an expected string", 10*time.Second)
			cp.ExpectExitCode(c.exitCode, 20*time.Second)
			// XXX: On Azure CI pipelines, the terminal output cannot be matched.  Needs investigation and a fix.
			if os.Getenv("CI") != "azure" {
				suite.Equal(c.terminalOutput, spaceRe.ReplaceAllString(cp.TrimmedSnapshot(), " "))
			}
		})
	}
}

func (suite *E2eSessionTestSuite) TestE2eSessionInterrupt() {
	if os.Getenv("CI") == "azure" {
		suite.T().Skip("session interrupt not working on Azure CI ATM")
	}
	// create a new test-session
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnCmd(suite.sessionTester, "-sleep", "-exit1")
	cp.Expect("an expected string", 10*time.Second)
	cp.SendCtrlC()
	cp.ExpectExitCode(123, 10*time.Second)
}
func TestE2eSessionTestSuite(t *testing.T) {
	suite.Run(t, new(E2eSessionTestSuite))
}
