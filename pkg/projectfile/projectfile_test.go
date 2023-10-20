package projectfile

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
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

	project, err := Parse(filepath.Join(rootpath, "activestate.yml.nope"))
	assert.Error(t, err, "Should throw an error")

	project, err = Parse(filepath.Join(rootpath, "pkg", "projectfile", "testdata", "activestate.yaml"))
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

	tmpfile, err := ioutil.TempFile("", "test")
	require.NoError(t, err, errs.JoinMessage(err))

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	project.path = tmpfile.Name()
	project.Save(cfg)

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

	os.Remove(tmpfile.Name())
}

// Call getProjectFilePath
func TestGetProjectFilePath(t *testing.T) {
	Reset()

	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	cwd, err := os.Getwd()
	assert.NoError(t, err, "Should fetch cwd")
	defer os.Chdir(cwd) // restore
	os.Chdir(filepath.Join(root, "pkg", "projectfile", "testdata"))

	configPath, err := GetProjectFilePath()
	require.Nil(t, err)
	expectedPath := filepath.Join(root, "pkg", "projectfile", "testdata", constants.ConfigFileName)
	assert.Equal(t, expectedPath, configPath, "Project path is properly detected")

	defer os.Unsetenv(constants.ProjectEnvVarName)

	os.Setenv(constants.ProjectEnvVarName, "/some/path")
	configPath, err = GetProjectFilePath()
	errt := &ErrorNoProjectFromEnv{}
	require.ErrorAs(t, err, &errt)

	expectedPath = filepath.Join(root, "pkg", "projectfile", "testdata", constants.ConfigFileName)
	os.Setenv(constants.ProjectEnvVarName, expectedPath)
	configPath, err = GetProjectFilePath()
	require.Nil(t, err)
	assert.Equal(t, expectedPath, configPath, "Project path is properly detected using the ProjectEnvVarName")

	os.Unsetenv(constants.ProjectEnvVarName)
	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	cfg.Set(constants.GlobalDefaultPrefname, "") // ensure it is unset
	tmpDir, err := ioutil.TempDir("", "")
	assert.NoError(t, err, "Should create temp dir")
	defer os.RemoveAll(tmpDir)
	os.Chdir(tmpDir)
	_, err = GetProjectFilePath()
	assert.Error(t, err, "GetProjectFilePath should fail")
	cfg.Set(constants.GlobalDefaultPrefname, expectedPath)
	configPath, err = GetProjectFilePath()
	assert.NoError(t, err, "GetProjectFilePath should succeed")
	assert.Equal(t, expectedPath, configPath, "Project path is properly detected using default path from config")
}

// TestGet the config
func TestGet(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	cwd, _ := os.Getwd()
	os.Chdir(filepath.Join(root, "pkg", "projectfile", "testdata"))

	config := Get()
	assert.NotNil(t, config, "Config should be set")
	assert.NotEqual(t, "", os.Getenv(constants.ProjectEnvVarName), "The project env var should be set")

	os.Chdir(cwd) // restore

	Reset()
}

func TestGetActivated(t *testing.T) {
	root, _ := environment.GetRootPath()
	cwd, _ := os.Getwd()
	os.Chdir(filepath.Join(root, "pkg", "projectfile", "testdata"))

	config1 := Get()
	assert.Equal(t, filepath.Join(root, "pkg", "projectfile", "testdata", constants.ConfigFileName), os.Getenv(constants.ProjectEnvVarName), "The activated state's config file is set")

	os.Chdir(root)
	config2, err := GetSafe()
	assert.NoError(t, err, "No error even if no activestate.yaml does not exist")
	assert.Equal(t, config1, config2, "The same activated state is returned")

	expected := filepath.Join(root, "pkg", "projectfile", "testdata", constants.ConfigFileName)
	actual := os.Getenv(constants.ProjectEnvVarName)
	assert.Equal(t, expected, actual, "The activated state's config file is still set properly")

	os.Chdir(cwd) // restore

	Reset()
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

	path, err := ioutil.TempDir("", "ParseVersionInfoTest")
	require.NoError(t, err)
	versionInfo, err = ParseVersionInfo(filepath.Join(path, constants.ConfigFileName))
	require.NoError(t, err)
	assert.Nil(t, versionInfo, "No version exists, because no project file exists")
}

func TestNewProjectfile(t *testing.T) {
	dir, err := ioutil.TempDir("", "projectfile-test")
	assert.NoError(t, err, "Should be no error when getting a temp directory")
	os.Chdir(dir)

	pjFile, err := testOnlyCreateWithProjectURL("https://platform.activestate.com/xowner/xproject", dir)
	assert.NoError(t, err, "There should be no error when loading from a path")
	assert.Equal(t, "activationMessage", pjFile.Scripts[0].Name)

	_, err = testOnlyCreateWithProjectURL("https://platform.activestate.com/xowner/xproject", "")
	assert.Error(t, err, "We don't accept blank paths")

	setCwd(t, "")
	dir, err = os.Getwd()
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
		name    string
		rawURL  string
		want    projectURL
		wantErr bool
	}{
		{
			"Valid full legacy URL",
			"https://platform.activestate.com/Owner/Name?commitID=7BA74758-8665-4D3F-921C-757CD271A0C1&branch=main",
			projectURL{
				Owner:          "Owner",
				Name:           "Name",
				LegacyCommitID: "7BA74758-8665-4D3F-921C-757CD271A0C1",
				BranchName:     "main",
			},
			false,
		},
		{
			"Valid commit URL",
			"https://platform.activestate.com/commit/7BA74758-8665-4D3F-921C-757CD271A0C1",
			projectURL{
				Owner:          "",
				Name:           "",
				LegacyCommitID: "7BA74758-8665-4D3F-921C-757CD271A0C1",
				BranchName:     "",
			},
			false,
		},
		{
			"Invalid commit",
			"https://platform.activestate.com/commit/nope",
			projectURL{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseURL(tt.rawURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
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
