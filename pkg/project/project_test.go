package project

import (
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

func loadProject(t *testing.T, contentType string) *projectfile.Project {
	project := &projectfile.Project{}
	contentsGeneral := strings.TrimSpace(`
platforms:
  - name: OSX
    os: darwin
    version: 10.0
    architecture: x386
    libc: "gnu"
    compiler: "gcc"
name: foo
environments: "something"
version: "1.0"
namespace: "my/name/space"
owner: "carey"
`)
	contentsVars := strings.TrimSpace(`
platforms:
  - name: Windows
    os: windows
  - name: Linux
    os: linux
  - name: OSX
    os: darwin
variables:
  - name: foo
    value: foo
    constraints:
      platform: Linux
  - name: bar
    value: bar
    constraints:
      platform: Windows
  - name: baz
    value: "baz"
    constraints:
      platform: OSX
`)
	contentsHooks := strings.TrimSpace(`
platforms:
  - name: Windows
    os: windows
  - name: Linux
    os: linux
  - name: OSX
    os: darwin
hooks:
  - name: baz
    value: quux
    constraints:
      platform: Windows
  - name: gonzo
    value: "echo 'something cool'"
    constraints:
      platform: OSX
  - name: bar
    value: baz
    constraints:
      platform: Linux
`)
	contentsCmds := strings.TrimSpace(`
platforms:
  - name: Windows
    os: windows
  - name: Linux
    os: linux
  - name: OSX
    os: darwin
commands:
  - name: foo
    value: foo
    standalone: true
    constraints:
      platform: Linux
  - name: bar
    value: bar
    standalone: true
    constraints:
      platform: Windows
  - name: baz
    value: baz
    standalone: true
    constraints:
      platform: OSX
`)
	contentsLangs := strings.TrimSpace(`
platforms:
  - name: Windows
    os: windows
  - name: Linux
    os: linux
  - name: OSX
    os: darwin
languages:
  - name: bar
    version: "1.1"
    build:
      override: --foo
    constraints:
      platform: Linux
  - name: baz
    version: "1.2"
    build:
      override: --bar
    constraints:
      platform: Windows
  - name: quiznar
    version: "1.3"
    build:
      override: --baz
    constraints:
      platform: OSX
`)
	contentsPkgs := strings.TrimSpace(`
platforms:
  - name: Windows
    os: windows
  - name: Linux
    os: linux
  - name: OSX
    os: darwin
languages:
  - name: foo
    version: "1.0"
    packages:
      - name: foo
        version: "1.1"
        constraints:
          platform: Linux
        build:
          override: --foo
      - name: bar
        version: "1.2"
        constraints:
            platform: Windows
        build:
          override: --bar
      - name: baz
        version: "1.3"
        constraints:
          platform: OSX
        build:
          override: --baz
`)
	var contents map[string]string
	contents = make(map[string]string)
	contents["Vars"] = contentsVars
	contents["Langs"] = contentsLangs
	contents["Cmds"] = contentsCmds
	contents["Pkgs"] = contentsPkgs
	contents["Hooks"] = contentsHooks
	contents["General"] = contentsGeneral
	fail := yaml.Unmarshal([]byte(contents[contentType]), project)
	assert.Nil(t, fail, "Unmarshalled YAML")
	project.Persist()

	return project
}

func TestGet(t *testing.T) {
	loadProject(t, "General")
	val := Get()
	assert.IsType(t, &Project{}, val, "Should be a project.go.Project")
}

func TestGetSafe(t *testing.T) {
	loadProject(t, "General")
	val, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	assert.IsType(t, &Project{}, val, "Should be a project.go.Project")
}

func TestProject(t *testing.T) {
	loadProject(t, "General")
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	assert.Equal(t, "foo", prj.Name(), "Values should match")
	assert.Equal(t, "carey", prj.Owner(), "Values should match")
	assert.Equal(t, "my/name/space", prj.Namespace(), "Values should match")
	assert.Equal(t, "something", prj.Environments(), "Values should match")
	assert.Equal(t, "1.0", prj.Version(), "Values should match")
}

func TestPlatforms(t *testing.T) {
	loadProject(t, "General")
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	val := prj.Platforms()
	plat := val[0]
	assert.Equal(t, 1, len(val), "Values should match")

	name := plat.Name()
	assert.Equal(t, "OSX", name, "Values should match")

	os, fail := plat.Os()
	assert.Nil(t, fail, "Run without failure")
	assert.Equal(t, "darwin", os, "Values should match")

	var version string
	version, fail = plat.Version()
	assert.Nil(t, fail, "Run without failure")
	assert.Equal(t, "10.0", version, "Values should match")

	var arch string
	arch, fail = plat.Architecture()
	assert.Nil(t, fail, "Run without failure")
	assert.Equal(t, "x386", arch, "Values should match")

	var libc string
	libc, fail = plat.Libc()
	assert.Nil(t, fail, "Run without failure")
	assert.Equal(t, "gnu", libc, "Values should match")

	var compiler string
	compiler, fail = plat.Compiler()
	assert.Nil(t, fail, "Run without failure")
	assert.Equal(t, "gcc", compiler, "Values should match")
}

func TestHooks(t *testing.T) {
	loadProject(t, "Hooks")
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")

	hooks := prj.Hooks()
	assert.Equal(t, 1, len(hooks), "Should match 1 out of three constrained items")

	hook := hooks[0]

	if runtime.GOOS == "linux" {
		name := hook.Name()
		assert.Equal(t, "bar", name, "Names should match (Linux)")
		var value string
		value, fail = hook.Value()
		assert.Nil(t, fail, "Run without failure")
		assert.Equal(t, "baz", value, "Value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		name := hook.Name()
		assert.Equal(t, "baz", name, "Name should match (Windows)")
		var value string
		value, fail = hook.Value()
		assert.Nil(t, fail, "Run without failure")
		assert.Equal(t, "quux", value, "Value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		name := hook.Name()
		assert.Equal(t, "gonzo", name, "Names should match (OSX)")
		var value string
		value, fail = hook.Value()
		assert.Nil(t, fail, "Run without failure")
		assert.Equal(t, "echo 'something cool'", value, "Value should match (OSX)")
	}
}

func TestLanguages(t *testing.T) {
	loadProject(t, "Langs")
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")

	languages := prj.Languages()
	assert.Equal(t, 1, len(languages), "Should match 1 out of three constrained items")

	lang := languages[0]

	if runtime.GOOS == "linux" {
		name := lang.Name()
		assert.Equal(t, "bar", name, "Names should match (Linux)")
		version := lang.Version()
		assert.Equal(t, "1.1", version, "Version should match (Linux)")
		build := lang.Build()
		assert.Equal(t, "--foo", (*build)["override"], "Build value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		name := lang.Name()
		assert.Equal(t, "baz", name, "Name should match (Windows)")
		version := lang.Version()
		assert.Equal(t, "1.2", version, "Version should match (Windows)")
		build := lang.Build()
		assert.Equal(t, "--bar", (*build)["override"], "Build value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		name := lang.Name()
		assert.Equal(t, "gonzo", name, "Names should match (OSX)")
		version := lang.Version()
		assert.Equal(t, "1.3 'something cool'", version, "Version should match (OSX)")
		build := lang.Build()
		assert.Equal(t, "--baz", (*build)["override"], "Build value should match (OSX)")
	}
}

func TestPackages(t *testing.T) {
	loadProject(t, "Pkgs")
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	languages := prj.Languages()
	language := languages[0]
	packages, fail := language.Packages()
	assert.Nil(t, fail, "Run without failure")
	assert.Equal(t, 1, len(packages), "Should match 1 out of three constrained items")

	pkg := packages[0]

	if runtime.GOOS == "linux" {
		name := pkg.Name()
		assert.Equal(t, "foo", name, "Names should match (Linux)")
		version := pkg.Version()
		assert.Equal(t, "1.1", version, "Version should match (Linux)")
		build := pkg.Build()
		assert.Equal(t, "--foo", (*build)["override"], "Build value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		name := pkg.Name()
		assert.Equal(t, "bar", name, "Name should match (Windows)")
		version := pkg.Version()
		assert.Equal(t, "1.2", version, "Version should match (Windows)")
		build := pkg.Build()
		assert.Equal(t, "--bar", (*build)["override"], "Build value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		name := pkg.Name()
		assert.Equal(t, "baz", name, "Names should match (OSX)")
		version := pkg.Version()
		assert.Equal(t, "1.3 'something cool'", version, "Version should match (OSX)")
		build := pkg.Build()
		assert.Equal(t, "--baz", (*build)["override"], "Build value should match (OSX)")
	}
}

func TestCommands(t *testing.T) {
	loadProject(t, "Cmds")
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	commands := prj.Commands()
	assert.Equal(t, 1, len(commands), "Should match 1 out of three constrained items")

	cmd := commands[0]

	if runtime.GOOS == "linux" {
		name := cmd.Name()
		assert.Equal(t, "foo", name, "Names should match (Linux)")
		version, fail := cmd.Value()
		assert.Nil(t, fail, "Run without failure")
		assert.Equal(t, "foo", version, "Value should match (Linux)")
		standalone := cmd.Standalone()
		assert.True(t, standalone, "Standalone value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		name := cmd.Name()
		assert.Equal(t, "bar", name, "Name should match (Windows)")
		version, fail := cmd.Value()
		assert.Nil(t, fail, "Run without failure")
		assert.Equal(t, "bar", version, "Value should match (Windows)")
		standalone := cmd.Standalone()
		assert.True(t, standalone, "Standalone value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		name := cmd.Name()
		assert.Equal(t, "baz", name, "Names should match (OSX)")
		version, fail := cmd.Value()
		assert.Nil(t, fail, "Run without failure")
		assert.Equal(t, "baz", version, "Value should match (OSX)")
		standalone := cmd.Standalone()
		assert.True(t, standalone, "Standalone value should match (OSX)")
	}
}

func TestVariables(t *testing.T) {
	loadProject(t, "Vars")
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	variables := prj.Variables()
	assert.Equal(t, 1, len(variables), "Should match 1 out of three constrained items")

	variable := variables[0]

	if runtime.GOOS == "linux" {
		name := variable.Name()
		assert.Equal(t, "foo", name, "Names should match (Linux)")
		value, fail := variable.Value()
		assert.Nil(t, fail, "Run without failure")
		assert.Equal(t, "foo", value, "Value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		name := variable.Name()
		assert.Equal(t, "bar", name, "Name should match (Windows)")
		value, fail := variable.Value()
		assert.Nil(t, fail, "Run without failure")
		assert.Equal(t, "bar", value, "Value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		name := variable.Name()
		assert.Equal(t, "baz", name, "Names should match (OSX)")
		value, fail := variable.Value()
		assert.Nil(t, fail, "Run without failure")
		assert.Equal(t, "baz", value, "Value should match (OSX)")
	}
}
