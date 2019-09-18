package activate

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/hail"
	"github.com/ActiveState/cli/internal/locale"
	promptMock "github.com/ActiveState/cli/internal/prompt/mock"
	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/pkg/platform/api"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	rMock "github.com/ActiveState/cli/pkg/platform/runtime/mock"
	"github.com/ActiveState/cli/pkg/projectfile"
)

const ProjectNamespace = "string/string"

type ActivateTestSuite struct {
	suite.Suite
	authMock   *authMock.Mock
	apiMock    *apiMock.Mock
	rMock      *rMock.Mock
	promptMock *promptMock.Mock
	dir        string
	origDir    string
}

func (suite *ActivateTestSuite) SetupSuite() {
	if os.Getenv("CI") == "true" {
		os.Setenv("SHELL", "/bin/bash")
	}

	authMock.Init().MockLoggedin()
}

func (suite *ActivateTestSuite) BeforeTest(suiteName, testName string) {
	suite.authMock = authMock.Init()
	suite.apiMock = apiMock.Init()
	suite.rMock = rMock.Init()
	suite.promptMock = promptMock.Init()
	prompter = suite.promptMock

	var err error

	suite.origDir, err = os.Getwd()
	suite.Require().NoError(err)
	suite.dir, err = ioutil.TempDir("", "activate-test")
	suite.Require().NoError(err)

	err = os.Chdir(suite.dir)
	suite.Require().NoError(err)

	// For some reason the working directory looks different once you cd into it (on mac), so ensure we use the right version
	suite.dir, err = os.Getwd()
	suite.Require().NoError(err)

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})

	Args.Namespace = ""

}

func (suite *ActivateTestSuite) AfterTest(suiteName, testName string) {
	os.Chdir(suite.origDir)

	suite.authMock.Close()
	suite.apiMock.Close()
	suite.rMock.Close()
	suite.promptMock.Close()
	err := os.RemoveAll(suite.dir)
	if err != nil {
		fmt.Printf("WARNING: Could not remove temp dir: %s, error: %v", suite.dir, err)
	}

	projectfile.Reset()
	failures.ResetHandled()
}

func (suite *ActivateTestSuite) TestExecute() {
	suite.rMock.MockFullRuntime()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "organizations/ActiveState/projects/CodeIntel")

	authentication.Get().AuthenticateWithToken("")

	dir := filepath.Join(environment.GetRootPathUnsafe(), "state", "activate", "testdata")
	err := os.Chdir(dir)
	suite.Require().NoError(err, "unable to chdir to testdata dir")
	suite.Require().FileExists(filepath.Join(dir, constants.ConfigFileName))

	Command.Execute()

	suite.Equal(true, true, "Execute didn't panic")
	suite.NoError(failures.Handled(), "No failure occurred")
}

func (suite *ActivateTestSuite) testExecuteWithNamespace(withLang bool) *projectfile.Project {
	suite.rMock.MockFullRuntime()

	if !withLang {
		suite.apiMock.MockGetProjectNoLanguage()
		suite.apiMock.MockVcsGetCheckpointCustomReq(nil)
	}

	targetDir := filepath.Join(suite.dir, ProjectNamespace)
	suite.promptMock.OnMethod("Input").Return(targetDir, nil)

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{ProjectNamespace})
	err := Command.Execute()
	suite.Require().NoError(err)

	suite.Equal(true, true, "Execute didn't panic")
	suite.NoError(failures.Handled(), "No failure occurred")

	configFile := filepath.Join(targetDir, constants.ConfigFileName)
	suite.FileExists(configFile)
	pjfile, fail := projectfile.Parse(configFile)
	suite.Require().NoError(fail.ToError())
	suite.Require().Equal("https://platform.activestate.com/string/string?commitID=00010001-0001-0001-0001-000100010001", pjfile.Project, "Project field should have been populated properly.")
	return pjfile
}

func (suite *ActivateTestSuite) TestExecuteWithNamespace() {
	suite.testExecuteWithNamespace(false)
}

func (suite *ActivateTestSuite) TestExecuteWithNamespaceDirExists() {
	targetDir := filepath.Join(suite.dir, ProjectNamespace)
	fail := fileutils.WriteFile(filepath.Join(targetDir, constants.ConfigFileName), []byte{})
	suite.Require().NoError(fail.ToError())

	ex := exiter.New()
	Command.Exiter = ex.Exit
	stderr := capturer.CaptureStderr(func() {
		code := ex.WaitForExit(func() {
			suite.testExecuteWithNamespace(false)
		})
		suite.Require().Equal(1, code, "Exits with code 1")
	})
	suite.Contains(stderr, locale.Tr("err_namespace_dir_inuse"))
}

func (suite *ActivateTestSuite) TestActivateFromNamespaceDontUseExisting() {
	suite.rMock.MockFullRuntime()
	suite.apiMock.MockGetProjectNoLanguage()
	suite.apiMock.MockVcsGetCheckpointCustomReq(nil)

	targetDirOrig := filepath.Join(suite.dir, ProjectNamespace)
	suite.promptMock.OnMethod("Input").Once().Return(targetDirOrig, nil)

	// Set up first checkout
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{ProjectNamespace})
	err := Command.Execute()
	suite.Require().NoError(err)

	suite.FileExists(filepath.Join(targetDirOrig, constants.ConfigFileName))
	savePathForNamespace(ProjectNamespace, targetDirOrig)

	// Now set up the second
	targetDirNew, err := ioutil.TempDir("", "DontUseExisting")
	suite.Require().NoError(err)
	suite.Require().NoError(os.Remove(targetDirNew))

	suite.promptMock.OnMethod("Select").Once().Return("", nil)
	suite.promptMock.OnMethod("Input").Once().Return(targetDirNew, nil)

	err = Command.Execute()
	suite.Require().NoError(err)

	suite.FileExists(filepath.Join(targetDirNew, constants.ConfigFileName))

	os.Chdir(suite.origDir)
	err = os.RemoveAll(targetDirNew) // clean up after
	if err != nil {
		fmt.Printf("WARNING: Could not remove temp dir: %s, error: %v", targetDirNew, err)
	}
}

func (suite *ActivateTestSuite) TestActivateFromNamespaceInvalidNamespace() {
	fail := activateFromNamespace("foo")
	suite.Equal(failInvalidNamespace.Name, fail.Type.Name)
}

func (suite *ActivateTestSuite) TestActivateFromNamespaceNoProject() {
	suite.authMock.MockLoggedin()
	suite.apiMock.MockGetProject404()

	fail := activateFromNamespace(ProjectNamespace)
	suite.Equal(api.FailProjectNotFound.Name, fail.Type.Name)
}

// lfrValOk calls listenForReactivation in such a way that we can be sure it
// will not hang forever.
func lfrValOk(max time.Duration, id string, rs <-chan *hail.Received, ss subShell) (bool, bool) {
	c := make(chan bool)
	go func() {
		defer close(c)
		c <- listenForReactivation(id, rs, ss)
	}()

	select {
	case <-time.After(max):
		return false, false
	case val := <-c:
		return val, true
	}
}

type lfrParams struct {
	id   string
	rcvs chan *hail.Received
	subs *mockSubShell
}

func makeLFRParams() lfrParams {
	return lfrParams{
		id:   "identifier",
		rcvs: make(chan *hail.Received, 1),
		subs: newMockSubShell(),
	}
}

func (suite *ActivateTestSuite) TestListenForReactivation() {
	t := suite.T() // used for subtests
	timeout := time.Millisecond * 1200
	timeoutFail := func() { suite.FailNow("timedout") }

	wg := &sync.WaitGroup{}
	wg.Add(4)

	go t.Run("hail received (failure)", func(t *testing.T) {
		defer wg.Done()

		lp := makeLFRParams()
		lp.rcvs <- &hail.Received{Fail: failures.FailDeveloper.New("hail failure")}

		if _, ok := lfrValOk(timeout, lp.id, lp.rcvs, lp.subs); ok {
			suite.Fail("should timeout")
		}

		suite.Equal(0, len(lp.rcvs), "channel should be empty")
	})

	go t.Run("hail received (invalid id)", func(t *testing.T) {
		defer wg.Done()

		lp := makeLFRParams()
		lp.rcvs <- &hail.Received{}

		if _, ok := lfrValOk(timeout, lp.id, lp.rcvs, lp.subs); ok {
			suite.Fail("should timeout")
		}

		suite.Equal(0, len(lp.rcvs), "channel should be empty")
	})

	go t.Run("hail received (no wait/wait)", func(t *testing.T) {
		defer wg.Done()

		lp := makeLFRParams()
		r := &hail.Received{Data: []byte(lp.id)}
		lp.rcvs <- r

		if _, ok := lfrValOk(time.Millisecond*500, lp.id, lp.rcvs, lp.subs); ok {
			suite.Fail("should take more than 500ms")
		}

		suite.Equal(0, len(lp.rcvs), "channel should be empty")

		lp.rcvs <- r

		if v, ok := lfrValOk(timeout, lp.id, lp.rcvs, lp.subs); ok {
			suite.True(v)
			suite.Equal(2, lp.subs.deacts)
			return
		}
		timeoutFail()
	})

	go t.Run("hail received (deactivation failure)", func(t *testing.T) {
		defer wg.Done()

		lp := makeLFRParams()
		lp.rcvs <- &hail.Received{Data: []byte(lp.id)}
		lp.subs.failNext = true

		if v, ok := lfrValOk(timeout, lp.id, lp.rcvs, lp.subs); ok {
			suite.False(v)
			return
		}
		timeoutFail()
	})

	wg.Wait()

	t.Run("close hails", func(t *testing.T) {
		lp := makeLFRParams()
		go close(lp.rcvs)

		if v, ok := lfrValOk(timeout, lp.id, lp.rcvs, lp.subs); ok {
			suite.False(v)
			return
		}
		timeoutFail()
	})

	t.Run("subs failure received", func(t *testing.T) {
		lp := makeLFRParams()
		lp.subs.fails <- failures.FailDeveloper.New("subs failure")

		if v, ok := lfrValOk(timeout, lp.id, lp.rcvs, lp.subs); ok {
			suite.False(v)
			return
		}
		timeoutFail()
	})

	t.Run("close subs failures", func(t *testing.T) {
		lp := makeLFRParams()
		go close(lp.subs.fails)

		if v, ok := lfrValOk(timeout, lp.id, lp.rcvs, lp.subs); ok {
			suite.False(v)
			return
		}
		timeoutFail()
	})
}

func (suite *ActivateTestSuite) TestUnstableWarning() {
	suite.rMock.MockFullRuntime()
	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "organizations/ActiveState/projects/CodeIntel")

	authentication.Get().AuthenticateWithToken("")

	defer func() { branchName = constants.BranchName }()
	branchName = "anything-but-stable"

	err := os.Chdir(filepath.Join(environment.GetRootPathUnsafe(), "state", "activate", "testdata"))
	suite.Require().NoError(err, "unable to chdir to testdata dir")

	out, err := osutil.CaptureStderr(func() {
		Command.Execute()
	})
	suite.Require().NoError(err)

	suite.Contains(out, locale.Tr("unstable_version_warning", constants.BugTrackerURL), "Prints our unstable warning")
}

func (suite *ActivateTestSuite) TestPromptCreateProjectFail() {
	projectFile := &projectfile.Project{}
	contents := strings.TrimSpace(`project: "https://platform.activestate.com/string/string"`)

	err := yaml.Unmarshal([]byte(contents), projectFile)
	suite.Require().NoError(err, "unexpected error marshalling yaml")

	projectFile.SetPath(filepath.Join(suite.dir, constants.ConfigFileName))
	projectFile.Save()
	suite.Require().NoError(err, "should be able to save in suite dir")
	defer os.Remove(filepath.Join(suite.dir, constants.ConfigFileName))

	suite.authMock.MockLoggedin()
	suite.apiMock.MockGetProject404()

	suite.promptMock.OnMethod("Confirm").Once().Return(false, nil)

	Command.Execute()

	suite.Require().Error(failures.Handled())
	suite.Require().Equal(failures.Handled().Error(), locale.T("err_must_create_project"))

}

func TestActivateSuite(t *testing.T) {
	suite.Run(t, new(ActivateTestSuite))
}

type mockSubShell struct {
	deacts   int
	failNext bool
	fails    chan *failures.Failure
}

func newMockSubShell() *mockSubShell {
	return &mockSubShell{
		deacts:   0,
		failNext: false,
		fails:    make(chan *failures.Failure, 1),
	}
}

func (ss *mockSubShell) Deactivate() *failures.Failure {
	ss.deacts++
	if ss.failNext {
		ss.failNext = false
		return failures.FailDeveloper.New("deactivation error")
	}
	return nil
}

func (ss *mockSubShell) Failures() <-chan *failures.Failure {
	return ss.fails
}
