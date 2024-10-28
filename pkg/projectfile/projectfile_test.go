package projectfile

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func setCwd(t *testing.T, subdir string) {
	err := os.Chdir(getWd(t, subdir))
	require.NoError(t, err, "Should change dir without issue.")
}

func getWd(t *testing.T, subdir string) string {
	cwd, err := environment.GetRootPath()
	require.NoError(t, err, "Should fetch cwd")
	path := filepath.Join(cwd, "pkg", "projectfile", "testdata")
	if subdir != "" {
		path = filepath.Join(path, subdir)
	}
	return path
}

func TestProjectStruct(t *testing.T) {
	project := Project{}
	dat := strings.TrimSpace(`
project: valueForProject
environments: valueForEnvironments`)

	err := yaml.Unmarshal([]byte(dat), &project)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForProject", project.Project, "Project should be set")
	assert.Equal(t, "valueForEnvironments", project.Environments, "Environments should be set")
	assert.Equal(t, "", project.Path(), "Path should be empty")
}

func TestBuildStruct(t *testing.T) {
	build := make(Build)
	dat := strings.TrimSpace(`
key1: val1
key2: val2`)

	err := yaml.Unmarshal([]byte(dat), &build)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "val1", build["key1"], "Key1 should be set")
	assert.Equal(t, "val2", build["key2"], "Key2 should be set")
}

func TestPackageStruct(t *testing.T) {
	pkg := Package{}
	dat := strings.TrimSpace(`
name: valueForName
version: valueForVersion`)

	err := yaml.Unmarshal([]byte(dat), &pkg)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForName", pkg.Name, "Name should be set")
	assert.Equal(t, "valueForVersion", pkg.Version, "Version should be set")
}

func TestEventStruct(t *testing.T) {
	event := Event{}
	dat := strings.TrimSpace(`
name: valueForName
value: valueForValue`)

	err := yaml.Unmarshal([]byte(dat), &event)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForName", event.Name, "Name should be set")
	assert.Equal(t, "valueForValue", event.Value, "Value should be set")
}

func TestScriptStruct(t *testing.T) {
	script := Script{}
	dat := strings.TrimSpace(`
name: valueForName
language: bash
value: valueForScript
standalone: true`)

	err := yaml.Unmarshal([]byte(dat), &script)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForName", script.Name, "Name should be set")
	assert.Equal(t, "bash", script.Language, "Language should match")
	assert.Equal(t, "valueForScript", script.Value, "Script should be set")
	assert.True(t, script.Standalone, "Standalone should be set")
}

func TestConstantStruct(t *testing.T) {
	constant := Constant{}
	dat := strings.TrimSpace(`
name: valueForName
value: valueForConstant`)

	err := yaml.Unmarshal([]byte(dat), &constant)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForName", constant.Name, "Name should be set")
	assert.Equal(t, "valueForConstant", constant.Value, "Constant should be set")
}

func TestSecretStruct(t *testing.T) {
	secret := Secret{}
	dat := strings.TrimSpace(`
name: valueForName
description: valueForDescription`)

	err := yaml.Unmarshal([]byte(dat), &secret)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForName", secret.Name, "Name should be set")
	assert.Equal(t, "valueForDescription", secret.Description, "Description should be set")
}

func TestParse(t *testing.T) {
	rootpath, err := environment.GetRootPath()

	if err != nil {
		t.Fatal(err)
	}

	_, err = Parse(filepath.Join(rootpath, "activestate.yml.nope"))
	assert.Error(t, err, "Should throw an error")

	project, err := Parse(filepath.Join(rootpath, "pkg", "projectfile", "testdata", "activestate.yaml"))
	require.NoError(t, err, "Should not throw an error")

	assert.NotEmpty(t, project.Project, "Project should be set")
	assert.NotEmpty(t, project.Environments, "Environments should be set")

	assert.NotEmpty(t, project.Constants[0].Name, "Constant name should be set")
	assert.NotEmpty(t, project.Constants[0].Value, "Constant value should be set")

	assert.NotEmpty(t, project.Secrets.User[0].Name, "Variable name should be set")
	assert.NotEmpty(t, project.Secrets.Project[0].Name, "Variable name should be set")

	assert.NotEmpty(t, project.Events[0].Name, "Event name should be set")
	assert.NotEmpty(t, project.Events[0].Value, "Event value should be set")

	assert.NotEmpty(t, project.Scripts[0].Name, "Script name should be set")
	assert.NotEmpty(t, project.Scripts[0].Value, "Script value should be set")
	assert.False(t, project.Scripts[0].Standalone, "Standalone value should be set, but false")

	assert.NotEmpty(t, project.Path(), "Path should be set")
}

func TestSave(t *testing.T) {
	rootpath, err := environment.GetRootPath()

	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(rootpath, "pkg", "projectfile", "testdata", "activestate.yaml")
	project, err := Parse(path)
	require.NoError(t, err, errs.JoinMessage(err))

	tmpfile, err := os.CreateTemp("", "test")
	require.NoError(t, err, errs.JoinMessage(err))

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	project.path = tmpfile.Name()
	err = project.Save(cfg)
	require.NoError(t, err)

	stat, err := tmpfile.Stat()
	assert.NoError(t, err, "Should be able to stat file")

	cfg2, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg2.Close()) }()

	projectURL := project.Project
	project.Project = "thisisnotatallaprojectURL"
	err = project.Save(cfg)
	assert.Error(t, err, "Saving project should fail due to bad projectURL format")
	project.Project = projectURL
	err = project.Save(cfg)
	assert.NoError(t, err, "Saving project should now pass")

	err = tmpfile.Close()
	assert.NoError(t, err, "Should close our temp file")

	assert.FileExists(t, tmpfile.Name(), "Project file is saved")
	assert.NotZero(t, stat.Size(), "Project file should have data")

	err = os.Remove(tmpfile.Name())
	assert.NoError(t, err, "Should remove our temp file")
}

func TestGetProjectFilePath(t *testing.T) {
	currentDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err = os.Chdir(currentDir)
		require.NoError(t, err)
	}()

	rootDir, err := fileutils.ResolvePath(fileutils.TempDirUnsafe())
	assert.NoError(t, err)
	defer os.RemoveAll(rootDir)

	// First, set up a new project with a subproject.
	projectDir := filepath.Join(rootDir, "project")
	require.NoError(t, fileutils.Mkdir(projectDir))
	projectYaml := filepath.Join(projectDir, constants.ConfigFileName)
	require.NoError(t, fileutils.Touch(projectYaml))
	subprojectDir := filepath.Join(projectDir, "subproject")
	require.NoError(t, fileutils.Mkdir(subprojectDir))
	subprojectYaml := filepath.Join(subprojectDir, constants.ConfigFileName)
	require.NoError(t, fileutils.Mkdir(subprojectDir))
	require.NoError(t, fileutils.Touch(subprojectYaml))

	// Then set up a separate, default project.
	defaultDir := filepath.Join(rootDir, "default")
	require.NoError(t, fileutils.Mkdir(defaultDir))
	defaultYaml := filepath.Join(defaultDir, constants.ConfigFileName)
	require.NoError(t, fileutils.Touch(defaultYaml))
	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	err = cfg.Set(constants.GlobalDefaultPrefname, defaultDir)
	require.NoError(t, err)

	// Now set up an empty directory.
	emptyDir := filepath.Join(rootDir, "empty")
	require.NoError(t, fileutils.Mkdir(emptyDir))

	// Now change to the project directory and assert GetProjectFilePath() returns it over the
	// default project.
	require.NoError(t, os.Chdir(projectDir))
	path, err := GetProjectFilePath()
	assert.NoError(t, err)
	assert.Equal(t, projectYaml, path)

	// `state shell` sets an environment variable, so run `state shell` in this project and then
	// change to the subproject directory. Assert GetProjectFilePath() still returns the parent
	// project.
	require.NoError(t, os.Setenv(constants.ActivatedStateEnvVarName, projectDir))
	defer os.Unsetenv(constants.ProfileEnvVarName)
	require.NoError(t, os.Chdir(subprojectDir))
	path, err = GetProjectFilePath()
	assert.NoError(t, err)
	assert.Equal(t, projectYaml, path)

	// If the project were to not exist, GetProjectFilePath() should return a typed error.
	require.NoError(t, os.Setenv(constants.ActivatedStateEnvVarName, filepath.Join(rootDir, "does-not-exist")))
	_, err = GetProjectFilePath()
	errNoProjectFromEnv := &ErrorNoProjectFromEnv{}
	assert.ErrorAs(t, err, &errNoProjectFromEnv)

	// After exiting out of the shell, the environment variable is no longer set. Assert
	// GetProjectFilePath() returns the subproject.
	require.NoError(t, os.Unsetenv(constants.ActivatedStateEnvVarName))
	path, err = GetProjectFilePath()
	assert.NoError(t, err)
	assert.Equal(t, subprojectYaml, path)

	// If a project's subdirectory does not contain an activestate.yaml file, GetProjectFilePath()
	// should walk up the tree until it finds one.
	require.NoError(t, os.Remove(subprojectYaml))
	path, err = GetProjectFilePath()
	assert.NoError(t, err)
	assert.Equal(t, projectYaml, path)

	// Change to an empty directory and assert GetProjectFilePath() returns the default project.
	require.NoError(t, os.Chdir(emptyDir))
	path, err = GetProjectFilePath()
	assert.NoError(t, err)
	assert.Equal(t, defaultYaml, path)

	// If the default project no longer exists, GetProjectFilePath() should return a typed error.
	err = cfg.Set(constants.GlobalDefaultPrefname, filepath.Join(rootDir, "does-not-exist"))
	require.NoError(t, err)
	_, err = GetProjectFilePath()
	errNoDefaultProject := &ErrorNoDefaultProject{}
	assert.ErrorAs(t, err, &errNoDefaultProject)

	// If none of the above, GetProjectFilePath() should return a typed error.
	err = cfg.Set(constants.GlobalDefaultPrefname, "")
	require.NoError(t, err)
	_, err = GetProjectFilePath()
	errNoProject := &ErrorNoProject{}
	assert.ErrorAs(t, err, &errNoProject)
}

// TestFromEnv the config
func TestFromEnv(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	cwd, _ := osutils.Getwd()
	err = os.Chdir(filepath.Join(root, "pkg", "projectfile", "testdata"))
	require.NoError(t, err, "Should change dir without issue.")

	config, err := FromEnv()
	require.NoError(t, err)
	assert.NotNil(t, config, "Config should be set")

	err = os.Chdir(cwd) // restore
	require.NoError(t, err, "Should change dir without issue.")
}

func TestParseVersionInfo(t *testing.T) {
	versionInfo, err := ParseVersionInfo(filepath.Join(getWd(t, ""), constants.ConfigFileName))
	require.NoError(t, err)
	assert.Nil(t, versionInfo, "No version exists")

	versionInfo, err = ParseVersionInfo(filepath.Join(getWd(t, "withversion"), constants.ConfigFileName))
	require.NoError(t, err)
	assert.NotNil(t, versionInfo, "Version exists")

	versionInfo, err = ParseVersionInfo(filepath.Join(getWd(t, "withbadversion"), constants.ConfigFileName))
	assert.Error(t, err)
	assert.Nil(t, versionInfo, "No version exists, because bad version")

	path, err := os.MkdirTemp("", "ParseVersionInfoTest")
	require.NoError(t, err)
	versionInfo, err = ParseVersionInfo(filepath.Join(path, constants.ConfigFileName))
	require.NoError(t, err)
	assert.Nil(t, versionInfo, "No version exists, because no project file exists")
}

func TestNewProjectfile(t *testing.T) {
	dir, err := os.MkdirTemp("", "projectfile-test")
	assert.NoError(t, err, "Should be no error when getting a temp directory")
	err = os.Chdir(dir)
	assert.NoError(t, err, "Should be no error when changing to the temp directory")

	pjFile, err := testOnlyCreateWithProjectURL("https://platform.activestate.com/xowner/xproject", dir)
	assert.NoError(t, err, "There should be no error when loading from a path")
	assert.Equal(t, "activationMessage", pjFile.Scripts[0].Name)

	_, err = testOnlyCreateWithProjectURL("https://platform.activestate.com/xowner/xproject", "")
	assert.Error(t, err, "We don't accept blank paths")

	setCwd(t, "")
	dir, err = osutils.Getwd()
	assert.NoError(t, err, "Should be no error when getting the CWD")
	_, err = testOnlyCreateWithProjectURL("https://platform.activestate.com/xowner/xproject", dir)
	assert.Error(t, err, "Cannot create new project if existing as.yaml ...exists")
}

func TestValidateProjectURL(t *testing.T) {
	err := ValidateProjectURL("https://example.com/")
	assert.Error(t, err, "This should be an invalid project URL")

	err = ValidateProjectURL("https://platform.activestate.com/xowner/xproject")
	assert.NoError(t, err, "This should not be an invalid project URL")

	err = ValidateProjectURL("https://platform.activestate.com/commit/commitid")
	assert.NoError(t, err, "This should not be an invalid project URL using the commit path")

	err = ValidateProjectURL("https://pr1234.activestate.build/commit/commitid")
	assert.NoError(t, err, "This should not be an invalid project URL using the commit path")
}

func Test_parseURL(t *testing.T) {
	tests := []struct {
		name   string
		rawURL string
		want   projectURL
	}{
		{
			"Full URL",
			"https://platform.activestate.com/Owner/Name?commitID=7BA74758-8665-4D3F-921C-757CD271A0C1&branch=main",
			projectURL{
				Owner:          "Owner",
				Name:           "Name",
				LegacyCommitID: "7BA74758-8665-4D3F-921C-757CD271A0C1",
				BranchName:     "main",
			},
		},
		{
			"Legacy commit URL",
			"https://platform.activestate.com/commit/7BA74758-8665-4D3F-921C-757CD271A0C1",
			projectURL{
				Owner:          "",
				Name:           "",
				LegacyCommitID: "7BA74758-8665-4D3F-921C-757CD271A0C1",
				BranchName:     "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseURL(tt.rawURL)
			if err != nil {
				t.Errorf("parseURL() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseURL() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_detectDeprecations(t *testing.T) {
	tests := []struct {
		name           string
		dat            string
		wantMatchError []string
	}{
		{
			"Constraints",
			`constraints: 0`,
			[]string{
				locale.Tr("pjfile_deprecation_entry", "constraints", "0"),
			},
		},
		{
			"Constraints Commented Out",
			`#constraints: 0`,
			[]string{},
		},
		{
			"Platforms",
			`platforms: 0"`,
			[]string{
				locale.Tr("pjfile_deprecation_entry", "platforms", "0"),
			},
		},
		{
			"Languages",
			`languages: 0`,
			[]string{
				locale.Tr("pjfile_deprecation_entry", "languages", "0"),
			},
		},
		{
			"Mixed",
			"foo: 0\nconstraints: 0\nbar: 0\nlanguages: 0\nplatforms: 0",
			[]string{
				locale.Tr("pjfile_deprecation_entry", "constraints", "7"),
				locale.Tr("pjfile_deprecation_entry", "languages", "29"),
				locale.Tr("pjfile_deprecation_entry", "platforms", "42"),
			},
		},
		{
			"Real world",
			`project: https://platform.activestate.com/ActiveState-CLI/test
platforms:
  - name: Linux64Label
languages:
  - name: Go
    constraints:
        platform: Windows10Label,Linux64Label`,
			[]string{
				locale.Tr("pjfile_deprecation_entry", "platforms", "63"),
				locale.Tr("pjfile_deprecation_entry", "languages", "97"),
				locale.Tr("pjfile_deprecation_entry", "constraints", "121"),
			},
		},
		{
			"Valid",
			"foo: 0\nbar: 0",
			[]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := detectDeprecations([]byte(tt.dat), "activestate.yaml")
			if len(tt.wantMatchError) == 0 {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			for _, want := range tt.wantMatchError {
				assert.Contains(t, err.Error(), want)
			}
		})
	}
}

// testOnlyCreateWithProjectURL a new activestate.yaml with default content
func testOnlyCreateWithProjectURL(projectURL, path string) (*Project, error) {
	return createCustom(&CreateParams{
		ProjectURL: projectURL,
		Directory:  path,
	}, language.Python3)
}
