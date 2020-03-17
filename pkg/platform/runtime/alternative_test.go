package runtime_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"
)

type AlternativeRuntimeTestSuite struct {
	suite.Suite

	cacheDir string
	recipeID strfmt.UUID
}

func (suite *AlternativeRuntimeTestSuite) BeforeTest(suiteName, testName string) {
	suite.recipeID = strfmt.UUID("00020002-0002-0002-0002-0002-00020000200002")
	var err error
	suite.cacheDir, err = ioutil.TempDir("", "cli-alternative-cache-dir")
	suite.Require().NoError(err, "cache dir created")
}

func (suite *AlternativeRuntimeTestSuite) AfterTest(suiteName, testName string) {
	err := os.RemoveAll(suite.cacheDir)
	suite.Assert().NoError(err, "cache dir removed")
}

func (suite *AlternativeRuntimeTestSuite) initWith(numArtifacts int, numTerminalArtifacts int, uriOverwrite ...string) ([]*runtime.HeadChefArtifact, *runtime.AlternativeRuntime, *failures.Failure) {
	artifacts := make([]*runtime.HeadChefArtifact, 0, numArtifacts+numTerminalArtifacts)
	for i := 0; i < numArtifacts; i++ {
		uri := fmt.Sprintf("https://test.tld/artifact%d/artifact.tar.gz", i)
		if len(uriOverwrite) == 1 {
			uri = uriOverwrite[0]
		}
		artifactID := strfmt.UUID(fmt.Sprintf("00010001-0001-0001-0001-00010001000%d", i))
		ingredientVersionID := strfmt.UUID(fmt.Sprintf("00020001-0001-0001-0001-00010001000%d", i))
		artifacts = append(artifacts, &runtime.HeadChefArtifact{
			ArtifactID:          &artifactID,
			IngredientVersionID: ingredientVersionID,
			URI:                 strfmt.URI(uri),
		})
	}
	for i := 0; i < numTerminalArtifacts; i++ {
		artifactID := strfmt.UUID(fmt.Sprintf("00010002-0001-0001-0001-00010001000%d", i))
		artifacts = append(artifacts, &runtime.HeadChefArtifact{
			ArtifactID: &artifactID,
			URI:        strfmt.URI("https://test.tld/terminal/artifact.tar.gz"),
		})

	}
	ar, fail := runtime.NewAlternativeRuntime(artifacts, suite.cacheDir, suite.recipeID)
	return artifacts, ar, fail
}

func (suite *AlternativeRuntimeTestSuite) Test_InitializationFailure() {
	cases := []struct {
		name         string
		uriOverwrite string
	}{
		{"filter empty URIs and terminal artifacts", ""},
		{"filter invalid URIs and terminal artifacts", "https://test.tld/artifact.invalid"},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			_, _, fail := suite.initWith(0, 2, tc.uriOverwrite)
			suite.Require().Error(fail.ToError())
			suite.Assert().Equal(runtime.FailNoValidArtifact, fail.Type)
		})

	}
}

func (suite *AlternativeRuntimeTestSuite) Test_ArtifactsToDownloadAndUnpack() {

	artifacts, ar, fail := suite.initWith(2, 0)
	suite.Require().NoError(fail.ToError())
	suite.Require().NotNil(ar)
	suite.Require().Len(artifacts, 2)

	cases := []struct {
		name        string
		preExisting int
	}{
		{"no cached artifacts", 0},
		{"one cached artifact", 1},
		{"all artifacts cached", 2},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			for i := 0; i < tc.preExisting; i++ {
				downloadDir, fail := ar.DownloadDirectory(artifacts[i])
				suite.Require().NoError(fail.ToError())
				fail = fileutils.MkdirUnlessExists(downloadDir)
				suite.Require().NoError(fail.ToError())

				err := ioutil.WriteFile(filepath.Join(downloadDir, constants.ArtifactArchiveName), []byte{}, 0666)
				suite.Require().NoError(err)
			}

			downloadArtfs, unpackArchives := ar.ArtifactsToDownloadAndUnpack()
			suite.Assert().Len(downloadArtfs, 2-tc.preExisting)
			suite.Assert().Len(unpackArchives, tc.preExisting)
		})
	}
}

func (suite *AlternativeRuntimeTestSuite) Test_PreInstall() {
	cases := []struct {
		name            string
		prepFunc        func(installDir string)
		expectedFailure *failures.FailureType
	}{
		{"InstallationDirectoryIsFile", func(installDir string) {
			baseDir := filepath.Dir(installDir)
			fail := fileutils.MkdirUnlessExists(baseDir)
			suite.Require().NoError(fail.ToError())
			err := ioutil.WriteFile(installDir, []byte{}, 0666)
			suite.Require().NoError(err)
		}, runtime.FailInstallDirInvalid},
		{"InstallationDirectoryIsNotEmpty", func(installDir string) {
			fail := fileutils.MkdirUnlessExists(installDir)
			suite.Require().NoError(fail.ToError())
			err := ioutil.WriteFile(filepath.Join(installDir, "dummy"), []byte{}, 0666)
			suite.Require().NoError(err)
		}, runtime.FailInstallDirInvalid},
		{"InstallationDirectoryIsOkay", func(installDir string) {}, nil},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			artifacts, ar, fail := suite.initWith(2, 1)
			suite.Require().NoError(fail.ToError())

			installDir := ar.InstallationDirectory(artifacts[0])
			defer os.RemoveAll(installDir)

			tc.prepFunc(installDir)
			fail = ar.PreInstall()
			if tc.expectedFailure == nil {
				suite.Require().NoError(fail.ToError())
				return
			}
			suite.Require().Error(fail.ToError())
			suite.Equal(tc.expectedFailure, fail.Type)
		})
	}
}
func Test_AlternativeRuntimeTestSuite(t *testing.T) {
	suite.Run(t, new(AlternativeRuntimeTestSuite))
}
