package integration

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/platform/runtime/envdef"
)

type AlternativeArtifactIntegrationTestSuite struct {
	tagsuite.Suite
}

// TestRelocation currently only tests the relocation mechanic for a Perl artifact.
// The artifact is downloaded directly form S3.  As soon as the artifacts are part of the platform ingredient library, this test should be rewritten, such that it relies on a `state activate` command.
func (suite *AlternativeArtifactIntegrationTestSuite) TestRelocation() {
	suite.OnlyRunForTags(tagsuite.Alternative)
	if runtime.GOOS == "darwin" {
		suite.T().Skip("No relocatable alternative artifacts for MacOS available yet.")
	}

	// IMPORTANT: When the following code is replaced by a simple `state activate` of an alternative project,
	// please ensure that the AWS credentials are removed from `.github/workflows-src/steps.lib.yml`

	shell := "bash"
	shellArg0 := "-c"
	artifactKey := "language/perl/5.32.0/3/7c76e6a6-3c41-5f68-a7f2-5468fe1b0919/artifact.tar.gz"
	matchReString := `-Dprefix=([^ ]+)/installdir`
	if runtime.GOOS == "windows" {
		suite.T().Skip("Temporary artifact tarball for windows is currently broken.")
		shell = "cmd"
		shellArg0 = "/c"
		artifactKey = "language/perl/5.32.0/3/6864c481-ff89-550d-9c61-a17ae57b7024/artifact.tar.gz"
		matchReString = `-L\"([^ ]+)installdir`
	}
	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			CredentialsChainVerboseErrors: p.BoolP(false),
			Region:                        aws.String("us-east-1"),
		},
	})
	suite.Require().NoError(err, "could not create aws session")
	s3c := s3.New(sess)
	object := &s3.GetObjectInput{
		Bucket: aws.String("as-builds"),
		Key:    aws.String(artifactKey),
	}
	resp, err := s3c.GetObject(object)
	suite.Require().NoError(err, "could not download artifact test tarball")

	artBody, err := ioutil.ReadAll(resp.Body)
	suite.Require().NoError(err, "could not read artifact body")
	ts := e2e.New(suite.T(), true)
	artTgz := filepath.Join(ts.Dirs.Work, "artifact.tar.gz")

	err = ioutil.WriteFile(artTgz, artBody, 0666)
	suite.Require().NoError(err, "failed to write artifacts file")

	installDir := filepath.Join(ts.Dirs.Cache, "installdir")
	tgz := unarchiver.NewTarGz()
	artTgzFile, artTgzSize, err := tgz.PrepareUnpacking(artTgz, ts.Dirs.Cache)
	defer artTgzFile.Close()
	suite.Require().NoError(err, "failed to prepare unpacking of artifact tarball")
	err = tgz.Unarchive(artTgzFile, artTgzSize, ts.Dirs.Cache)
	suite.Require().NoError(err, "failed to unarchive the artifact")
	edFile := filepath.Join(ts.Dirs.Cache, "runtime.json")
	ed, err := envdef.NewEnvironmentDefinition(edFile)
	suite.Require().NoError(err, "failed to create environment definition file")

	constants, err := envdef.NewConstants(installDir)
	suite.Require().NoError(err, "failed to get new constants")
	ed = ed.ExpandVariables(constants)
	env := ed.GetEnv(true)

	cp := ts.SpawnCmdWithOpts(shell, e2e.WithArgs(shellArg0, "perl -V"), e2e.AppendEnv(osutils.EnvMapToSlice(env)...))

	// Find prefix directory as returned by `perl -V`

	// Check that the prefix is NOT yet set to the installation directory
	cp.ExpectLongString("installdir")
	matchRe := regexp.MustCompile(matchReString)
	cp.Snapshot()
	res := matchRe.FindStringSubmatch(cp.TrimmedSnapshot())
	suite.Require().Len(res, 2)
	suite.NotEqual(filepath.Clean(ts.Dirs.Cache), filepath.Clean(res[1]))
	cp.ExpectExitCode(0)

	// Apply the file transformations (relocations)
	err = ed.ApplyFileTransforms(ts.Dirs.Cache, constants)
	suite.Require().NoError(err, "failed to apply file transformations.")

	cp = ts.SpawnCmdWithOpts(shell, e2e.WithArgs(shellArg0, "perl -V"), e2e.AppendEnv(osutils.EnvMapToSlice(env)...))

	// Check that the prefix now IS set to the installation directory
	cp.ExpectLongString("installdir")
	res = matchRe.FindStringSubmatch(cp.TrimmedSnapshot())
	suite.Require().Len(res, 2)
	suite.Equal(filepath.Clean(ts.Dirs.Cache), filepath.Clean(res[1]))
	cp.ExpectExitCode(0)

	cp = ts.SpawnCmdWithOpts(shell, e2e.WithArgs(shellArg0, "perl --version"), e2e.AppendEnv(osutils.EnvMapToSlice(env)...))
	cp.Expect("v5.32.0")
	cp.ExpectExitCode(0)
}

func (suite *AlternativeArtifactIntegrationTestSuite) TestActivateRuby() {
	suite.OnlyRunForTags(tagsuite.Alternative)
	suite.T().Skip("requires a working PR branch for now.")
	if runtime.GOOS != "linux" {
		suite.T().Skip("only works on linux")
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	extraEnv := e2e.AppendEnv(
		"ACTIVESTATE_API_HOST=pr3134.activestate.build",
		"ACTIVESTATE_CLI_DISABLE_RUNTIME=false",
	)

	// Download artifacts but interrupt installation step
	cp := ts.SpawnWithOpts(
		e2e.WithArgs("deploy", "install", "martind-stage/ruby"),
		extraEnv,
	)

	// TODO interrupt a download, and ensure that download is retried!
	cp.Expect("Downloading")
	cp.Expect("6 / 6")
	cp.Expect("Installing")
	cp.SendCtrlC()
	cp.ExpectNotExitCode(0)

	// On activation, nothing is downloaded, but installation is completed
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("activate", "martind-stage/ruby", "--path", ts.Dirs.Work),
		extraEnv,
	)
	cp.Expect("Installing")
	cp.Expect("6 / 6")
	cp.Expect("Activated")

	cp.SendLine(`ruby -e 'puts "      world\rhello"'`)
	cp.Expect("hello world")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	// Completely cached activation: no file needs to be downloaded or installed
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("activate", "martind-stage/ruby", "--path", ts.Dirs.Work),
		extraEnv,
	)
	cp.Expect("Activated")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
	suite.NotContains(cp.TrimmedSnapshot(), "Downloading required artifacts")
	suite.NotContains(cp.TrimmedSnapshot(), "Installing")

	// Only one cached download missing
	cachedArtifacts, err := ioutil.ReadDir(filepath.Join(ts.Dirs.Cache, "artifacts"))
	suite.Require().NoError(err, "listing cached artifacts")
	suite.Len(cachedArtifacts, 6, "expected six cached artifacts")

	err = os.RemoveAll(filepath.Join(ts.Dirs.Cache, "artifacts", cachedArtifacts[0].Name()))
	suite.Require().NoError(err, "removing a single artifact")
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("activate", "martind-stage/ruby", "--path", ts.Dirs.Work),
		extraEnv,
	)
	cp.Expect("Downloading")
	cp.Expect("1 / 1")
	cp.Expect("Installing")
	cp.Expect("6 / 6")
	cp.Expect("Activated")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func TestAlternativeArtifactIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(AlternativeArtifactIntegrationTestSuite))
}
