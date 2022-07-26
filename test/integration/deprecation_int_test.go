package integration

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/testsuite"
)

type DeprecationIntegrationTestSuite struct {
	testsuite.Suite
}

func (suite *DeprecationIntegrationTestSuite) TestHardDeprecation() {
	suite.OnlyRunForTags(testsuite.TagCritical, testsuite.TagDeprecation)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	f, err := fileutils.WriteTempFile("", "TestDeprecation", []byte(fmt.Sprintf(`
		[
		  {
		    "version": "%s",
		    "date": "%s",
		    "reason": "Deprecation test"
		  }
		]
	`, constants.VersionNumber, time.Now().Format(time.RFC3339))), os.ModePerm)
	suite.Require().NoError(err)

	cp := ts.SpawnWithOpts(e2e.WithArgs("--version"), e2e.AppendEnv(
		constants.DeprecationOverrideEnvVarName+"="+f,
	))
	cp.Send("")
	cp.Expect("Deprecation test")
	cp.ExpectExitCode(1) // Should exit non-zero because hard deprecation interrupts process
}

func (suite *DeprecationIntegrationTestSuite) TestDeprecationVersionTooLow() {
	suite.OnlyRunForTags(testsuite.TagDeprecation)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	f, err := fileutils.WriteTempFile("", "TestDeprecation", []byte(fmt.Sprintf(`
		[
		  {
		    "version": "9999.9999.9999",
		    "date": "%s",
		    "reason": "Deprecation test"
		  }
		]
	`, time.Now().Format(time.RFC3339))), os.ModePerm)
	suite.Require().NoError(err)

	cp := ts.SpawnWithOpts(e2e.WithArgs("--version"), e2e.AppendEnv(
		constants.DeprecationOverrideEnvVarName+"="+f,
	))
	cp.Send("")
	cp.Expect("Deprecation test")
	cp.ExpectExitCode(1) // Should exit non-zero because hard deprecation interrupts process
}

func (suite *DeprecationIntegrationTestSuite) TestDeprecationVersionHigher() {
	suite.OnlyRunForTags(testsuite.TagDeprecation)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	f, err := fileutils.WriteTempFile("", "TestDeprecation", []byte(fmt.Sprintf(`
		[
		  {
		    "version": "0.0.0",
		    "date": "%s",
		    "reason": "Deprecation test"
		  }
		]
	`, time.Now().Format(time.RFC3339))), os.ModePerm)
	suite.Require().NoError(err)

	cp := ts.SpawnWithOpts(e2e.WithArgs("--version"), e2e.AppendEnv(
		constants.DeprecationOverrideEnvVarName+"="+f,
	))
	cp.Send("")
	cp.ExpectExitCode(0) // Should exit non-zero because hard deprecation interrupts process
}

func (suite *DeprecationIntegrationTestSuite) TestSoftDeprecation() {
	suite.OnlyRunForTags(testsuite.TagDeprecation)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	f, err := fileutils.WriteTempFile("", "TestDeprecation", []byte(fmt.Sprintf(`
		[
		  {
		    "version": "%s",
		    "date": "%s",
		    "reason": "Deprecation test"
		  }
		]
	`, constants.VersionNumber, time.Now().Add(time.Hour).Format(time.RFC3339))), os.ModePerm)
	suite.Require().NoError(err)

	cp := ts.SpawnWithOpts(e2e.WithArgs("--version"), e2e.AppendEnv(
		constants.DeprecationOverrideEnvVarName+"="+f,
	))
	cp.Send("")
	cp.Expect("Deprecation test")
	cp.ExpectExitCode(0) // Should exit zero because soft deprecation only warns
}

func TestDeprecationIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(DeprecationIntegrationTestSuite))
}
