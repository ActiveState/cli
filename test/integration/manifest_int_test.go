package integration

import (
	"encoding/json"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type ManifestIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ManifestIntegrationTestSuite) TestManifest() {
	suite.OnlyRunForTags(tagsuite.Manifest)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState/cli#9eee7512-b2ab-4600-b78b-ab0cf2e817d8", "."),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("manifest"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Operating on project: ActiveState/cli", e2e.RuntimeSourcingTimeoutOpt)
	cp.Expect("Name")
	cp.Expect("python")
	cp.Expect("3.9.13")
	cp.Expect("1 Critical,")
	cp.Expect("psutil")
	cp.Expect("auto → 5.9.0")
	cp.Expect("None detected")
	cp.ExpectExitCode(0)
}

func (suite *ManifestIntegrationTestSuite) TestManifest_JSON() {
	suite.OnlyRunForTags(tagsuite.Manifest)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState/cli#9eee7512-b2ab-4600-b78b-ab0cf2e817d8", "."),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)

	type version struct {
		Requested string `json:"requested"`
		Resolved  string `json:"resolved"`
	}

	type vulnerabilities struct {
		Count map[string]int `json:"count"`
	}

	type requirement struct {
		Name            string          `json:"name"`
		Version         version         `json:"version"`
		Vulnerabilities vulnerabilities `json:"vulnerabilities"`
	}

	type requirements struct {
		Requirements []requirement `json:"requirements"`
	}

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("manifest", "--output", "json"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.ExpectExitCode(0)

	snapshot := cp.StrippedSnapshot()
	var result requirements
	err := json.Unmarshal([]byte(snapshot), &result)
	suite.Require().NoError(err)

	for _, req := range result.Requirements {
		suite.Require().NotEmpty(req.Name)

		if req.Name == "python" {
			suite.Require().Equal("3.9.13", req.Version.Requested)
			suite.Require().Equal("3.9.13", req.Version.Resolved)
			suite.Require().NotEmpty(req.Vulnerabilities.Count)
		}

		if req.Name == "psutil" {
			suite.Require().Empty(req.Version.Requested)
			suite.Require().Equal("5.9.0", req.Version.Resolved)
			suite.Require().Empty(req.Vulnerabilities.Count)
		}
	}
}

func TestManifestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ManifestIntegrationTestSuite))
}
