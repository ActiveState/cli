package runtime_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/progress/mock"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/envdef"
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

func (suite *AlternativeRuntimeTestSuite) mockEnvDefs(num int) (defs []*envdef.EnvironmentDefinition, merged *envdef.EnvironmentDefinition) {
	defs = make([]*envdef.EnvironmentDefinition, 0, num)
	merged = &envdef.EnvironmentDefinition{Env: []envdef.EnvironmentVariable{}, InstallDir: "installdir"}
	for i := 0; i < num; i++ {
		def := &envdef.EnvironmentDefinition{
			Env: []envdef.EnvironmentVariable{
				{
					Name:    "COMMON",
					Values:  []string{fmt.Sprintf("%02d", i)},
					Inherit: false,
				},
				{
					Name:    fmt.Sprintf("VAR%02d", i),
					Values:  []string{"set"},
					Inherit: false,
				},
			},
			InstallDir: "installdir",
		}

		var err error
		merged, err = merged.Merge(def)
		suite.Require().NoError(err)

		defs = append(defs, def)
	}
	return defs, merged
}

func (suite *AlternativeRuntimeTestSuite) mockTemporaryRuntimeDirs(defs []*envdef.EnvironmentDefinition) []string {
	tmpRuntimeBase := filepath.Join(suite.cacheDir, "temp-runtime-base")
	dirs := make([]string, 0, len(defs))

	for i, def := range defs {
		tmpRuntimeDir := filepath.Join(tmpRuntimeBase, fmt.Sprintf("%02d", i))
		fail := fileutils.MkdirUnlessExists(tmpRuntimeDir)
		suite.Require().NoError(fail.ToError())

		// create runtime.json
		err := def.WriteFile(filepath.Join(tmpRuntimeDir, constants.RuntimeDefinitionFilename))
		suite.Require().NoError(err)

		// create one installation file
		fail = fileutils.MkdirUnlessExists(filepath.Join(tmpRuntimeDir, "installdir", "bin"))
		suite.Require().NoError(fail.ToError())

		err = ioutil.WriteFile(filepath.Join(tmpRuntimeDir, "installdir", "bin", fmt.Sprintf("executable%02d", i)), []byte{}, 0555)
		suite.Require().NoError(err)

		dirs = append(dirs, tmpRuntimeDir)
	}
	return dirs
}

func (suite *AlternativeRuntimeTestSuite) Test_GetEnv() {
	numArtifacts := 2
	artifacts := mockFetchArtifactsResult(withRegularArtifacts(numArtifacts))
	ar, fail := runtime.NewAlternativeRuntime(artifacts.Artifacts, suite.cacheDir, artifacts.RecipeID)
	suite.Require().NoError(fail.ToError())

	suite.Require().NoError(fail.ToError())
	envDefs, merged := suite.mockEnvDefs(numArtifacts)

	runtimeDirs := suite.mockTemporaryRuntimeDirs(envDefs)

	for i := numArtifacts - 1; i >= 0; i-- {
		counter := mock.NewMockIncrementer()
		fail := ar.PostUnpackArtifact(artifacts.Artifacts[i], runtimeDirs[i], "", func() { counter.Increment() })
		suite.Assert().NoError(fail.ToError())
		suite.Assert().Equal(1, counter.Count, "one executable moved to final installation directory")
	}

	expectedEnv := merged.GetEnv(true)

	mergedFilePath := filepath.Join(suite.cacheDir, constants.LocalRuntimeEnvironmentDirectory, constants.RuntimeDefinitionFilename)
	firstEnvDefPath := filepath.Join(suite.cacheDir, constants.LocalRuntimeEnvironmentDirectory, fmt.Sprintf("%06d.json", 0))

	suite.Assert().False(fileutils.FileExists(mergedFilePath))
	// installation complete marker is missing
	env, err := ar.GetEnv(true, "")
	suite.Require().Error(err, "installation complete marker is missing")

	err = ar.PostInstall()
	suite.Require().NoError(err, "merged runtime environment definition is created")
	suite.Assert().True(fileutils.FileExists(mergedFilePath))

	env, err = ar.GetEnv(true, "")
	suite.Require().NoError(err)

	suite.Assert().Equal(expectedEnv, env)
	err = os.Remove(firstEnvDefPath)
	suite.Assert().NoError(err, "removing cached runtime definition file for first artifact")

	// This should still work, as we have cached the merged result by now
	env, err = ar.GetEnv(true, "")
	suite.Require().NoError(err)
	suite.Assert().Equal(expectedEnv, env)
}

func (suite *AlternativeRuntimeTestSuite) Test_InitializationFailure() {
	cases := []struct {
		name   string
		option artifactsResultMockOption
	}{
		{"filter empty URIs", withURIArtifact("")},
		{"filter invalid URIs", withURIArtifact("https://test.tld/artifact.invalid")},
		{"filter terminal artifacts", withTerminalArtifacts(1)},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			artifactsResult := mockFetchArtifactsResult(tc.option)
			_, fail := runtime.NewAlternativeRuntime(artifactsResult.Artifacts, suite.cacheDir, artifactsResult.RecipeID)
			suite.Require().Error(fail.ToError())
			suite.Assert().Equal(runtime.FailNoValidArtifact, fail.Type)
		})

	}
}

func (suite *AlternativeRuntimeTestSuite) Test_ArtifactsToDownloadAndUnpack() {
	artifactsRes := mockFetchArtifactsResult(withRegularArtifacts(2))
	suite.Require().Len(artifactsRes.Artifacts, 2)
	ar, fail := runtime.NewAlternativeRuntime(artifactsRes.Artifacts, suite.cacheDir, artifactsRes.RecipeID)
	suite.Require().NoError(fail.ToError())
	suite.Require().NotNil(ar)

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
				downloadDir, fail := ar.DownloadDirectory(artifactsRes.Artifacts[i])
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
		}, nil},
		{"InstallationDirectoryIsOkay", func(installDir string) {}, nil},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			artifactsRes := mockFetchArtifactsResult(withRegularArtifacts(2))
			ar, fail := runtime.NewAlternativeRuntime(artifactsRes.Artifacts, suite.cacheDir, artifactsRes.RecipeID)
			suite.Require().NoError(fail.ToError())

			defer os.RemoveAll(suite.cacheDir)

			tc.prepFunc(suite.cacheDir)
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
