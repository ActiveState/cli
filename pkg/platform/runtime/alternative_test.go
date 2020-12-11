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
		err := fileutils.MkdirUnlessExists(tmpRuntimeDir)
		suite.Require().NoError(err)

		// create runtime.json
		err = def.WriteFile(filepath.Join(tmpRuntimeDir, constants.RuntimeDefinitionFilename))
		suite.Require().NoError(err)

		// create one installation file
		err = fileutils.MkdirUnlessExists(filepath.Join(tmpRuntimeDir, "installdir", "bin"))
		suite.Require().NoError(err)

		err = ioutil.WriteFile(filepath.Join(tmpRuntimeDir, "installdir", "bin", fmt.Sprintf("executable%02d", i)), []byte{}, 0555)
		suite.Require().NoError(err)

		dirs = append(dirs, tmpRuntimeDir)
	}
	return dirs
}

func (suite *AlternativeRuntimeTestSuite) Test_GetEnv() {
	numArtifacts := 2
	artifacts := mockFetchArtifactsResult(withRegularArtifacts(numArtifacts))
	ar, err := runtime.NewAlternativeInstall(suite.cacheDir, artifacts.Artifacts, artifacts.RecipeID)
	suite.Require().NoError(err)

	suite.Require().NoError(err)
	envDefs, merged := suite.mockEnvDefs(numArtifacts)

	runtimeDirs := suite.mockTemporaryRuntimeDirs(envDefs)

	for i := numArtifacts - 1; i >= 0; i-- {
		counter := mock.NewMockIncrementer()
		fail := ar.PostUnpackArtifact(artifacts.Artifacts[i], runtimeDirs[i], "", func() { counter.Increment() })
		suite.Assert().NoError(fail)
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
			_, err := runtime.NewAlternativeInstall(suite.cacheDir, artifactsResult.Artifacts, artifactsResult.RecipeID)
			errt := &runtime.ErrInvalidArtifact{}
			suite.Error(err, &errt)
		})

	}
}

func (suite *AlternativeRuntimeTestSuite) Test_PreInstall() {
	cases := []struct {
		name          string
		prepFunc      func(installDir string)
		expectedError error
	}{
		{"InstallationDirectoryIsFile", func(installDir string) {
			baseDir := filepath.Dir(installDir)
			err := fileutils.MkdirUnlessExists(baseDir)
			suite.Require().NoError(err)
			err = ioutil.WriteFile(installDir, []byte{}, 0666)
			suite.Require().NoError(err)
		}, &runtime.ErrInstallDirInvalid{}},
		{"InstallationDirectoryIsNotEmpty", func(installDir string) {
			err := fileutils.MkdirUnlessExists(installDir)
			suite.Require().NoError(err)
			err = ioutil.WriteFile(filepath.Join(installDir, "dummy"), []byte{}, 0666)
			suite.Require().NoError(err)
		}, nil},
		{"InstallationDirectoryIsOkay", func(installDir string) {}, nil},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			artifactsRes := mockFetchArtifactsResult(withRegularArtifacts(2))
			ar, err := runtime.NewAlternativeInstall(suite.cacheDir, artifactsRes.Artifacts, artifactsRes.RecipeID)
			suite.Require().NoError(err)

			os.RemoveAll(suite.cacheDir)
			defer os.RemoveAll(suite.cacheDir)

			tc.prepFunc(suite.cacheDir)
			err = ar.PreInstall()
			if tc.expectedError == nil {
				suite.Require().NoError(err)
			} else {
				suite.ErrorAs(err, &tc.expectedError)
			}
		})
	}
}
func Test_AlternativeRuntimeTestSuite(t *testing.T) {
	suite.Run(t, new(AlternativeRuntimeTestSuite))
}
