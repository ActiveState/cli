package project

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

var cwd string

func setProjectDir(t *testing.T) {
	var err error
	cwd, err = os.Getwd()
	assert.NoError(t, err, "Should fetch cwd")
	os.Chdir(filepath.Join(cwd, "testdata"))
}

func resetProjectDir(t *testing.T) {
	os.Chdir(cwd)
}

func TestGet(t *testing.T) {
	setProjectDir(t)
	config := Get()
	assert.NotNil(t, config, "Config should be set")
	resetProjectDir(t)
}

func TestGetSafe(t *testing.T) {
	setProjectDir(t)
	val, fail := GetSafe()
	assert.NoError(t, fail.ToError(), "Run without failure")
	assert.NotNil(t, val, "Config should be set")
	resetProjectDir(t)
}

func TestProject(t *testing.T) {
	setProjectDir(t)
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	assert.Equal(t, "foo", prj.Name(), "Values should match")
	assert.Equal(t, "carey", prj.Owner(), "Values should match")
	assert.Equal(t, "my/name/space", prj.Namespace(), "Values should match")
	assert.Equal(t, "something", prj.Environments(), "Values should match")
	assert.Equal(t, "1.0", prj.Version(), "Values should match")
	resetProjectDir(t)
}

func TestPlatforms(t *testing.T) {
	setProjectDir(t)
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	val := prj.Platforms()
	plat := val[0]
	assert.Equal(t, 3, len(val), "Values should match")

	name := plat.Name()
	assert.Equal(t, "OSX", name, "Names should match")

	os := plat.Os()
	assert.Equal(t, "darwin", os, "OS should match")

	var version string
	version = plat.Version()
	assert.Equal(t, "10.0", version, "Version should match")

	var arch string
	arch = plat.Architecture()
	assert.Equal(t, "x386", arch, "Arch should match")

	var libc string
	libc = plat.Libc()
	assert.Equal(t, "gnu", libc, "Libc should match")

	var compiler string
	compiler = plat.Compiler()
	assert.Equal(t, "gcc", compiler, "Compiler should match")
	resetProjectDir(t)
}

func TestHooks(t *testing.T) {
	setProjectDir(t)
	prj, fail := GetSafe()
	assert.NoError(t, fail.ToError(), "Run without failure")

	hooks := prj.Hooks()
	assert.Equal(t, 1, len(hooks), "Should match 1 out of three constrained items")

	hook := hooks[0]

	if runtime.GOOS == "linux" {
		name := hook.Name()
		assert.Equal(t, "foo", name, "Names should match (Linux)")
		value := hook.Value()
		assert.Equal(t, "foo Linux", value, "Value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		name := hook.Name()
		assert.Equal(t, "bar", name, "Name should match (Windows)")
		value := hook.Value()
		assert.Equal(t, "bar Windows", value, "Value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		name := hook.Name()
		assert.Equal(t, "baz", name, "Names should match (OSX)")
		value := hook.Value()
		assert.Equal(t, "baz OSX", value, "Value should match (OSX)")
	}
	resetProjectDir(t)
}

func TestLanguages(t *testing.T) {
	setProjectDir(t)
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")

	languages := prj.Languages()
	assert.Equal(t, 2, len(languages), "Should match 1 out of three constrained items")

	lang := languages[0]

	if runtime.GOOS == "linux" {
		name := lang.Name()
		assert.Equal(t, "foo", name, "Names should match (Linux)")
		version := lang.Version()
		assert.Equal(t, "1.1", version, "Version should match (Linux)")
		build := lang.Build()
		assert.Equal(t, "--foo Linux", (*build)["override"], "Build value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		name := lang.Name()
		assert.Equal(t, "bar", name, "Name should match (Windows)")
		version := lang.Version()
		assert.Equal(t, "1.2", version, "Version should match (Windows)")
		build := lang.Build()
		assert.Equal(t, "--bar Windows", (*build)["override"], "Build value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		name := lang.Name()
		assert.Equal(t, "baz", name, "Names should match (OSX)")
		version := lang.Version()
		assert.Equal(t, "1.3", version, "Version should match (OSX)")
		build := lang.Build()
		assert.Equal(t, "--baz OSX", (*build)["override"], "Build value should match (OSX)")
	}
	resetProjectDir(t)
}

func TestPackages(t *testing.T) {
	setProjectDir(t)
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	languages := prj.Languages()
	var language *Language
	for _, l := range languages {
		if l.Name() == "packages" {
			language = l
		}
	}
	packages := language.Packages()
	assert.Equal(t, 1, len(packages), "Should match 1 out of three constrained items")

	pkg := packages[0]

	if runtime.GOOS == "linux" {
		name := pkg.Name()
		assert.Equal(t, "foo", name, "Names should match (Linux)")
		version := pkg.Version()
		assert.Equal(t, "1.1", version, "Version should match (Linux)")
		build := pkg.Build()
		assert.Equal(t, "--foo Linux", (*build)["override"], "Build value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		name := pkg.Name()
		assert.Equal(t, "bar", name, "Name should match (Windows)")
		version := pkg.Version()
		assert.Equal(t, "1.2", version, "Version should match (Windows)")
		build := pkg.Build()
		assert.Equal(t, "--bar Windows", (*build)["override"], "Build value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		name := pkg.Name()
		assert.Equal(t, "baz", name, "Names should match (OSX)")
		version := pkg.Version()
		assert.Equal(t, "1.3", version, "Version should match (OSX)")
		build := pkg.Build()
		assert.Equal(t, "--baz OSX", (*build)["override"], "Build value should match (OSX)")
	}
	resetProjectDir(t)
}

func TestCommands(t *testing.T) {
	setProjectDir(t)
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	commands := prj.Commands()
	assert.Equal(t, 1, len(commands), "Should match 1 out of three constrained items")

	cmd := commands[0]

	if runtime.GOOS == "linux" {
		name := cmd.Name()
		assert.Equal(t, "foo", name, "Names should match (Linux)")
		version := cmd.Value()
		assert.Equal(t, "foo Linux", version, "Value should match (Linux)")
		standalone := cmd.Standalone()
		assert.True(t, standalone, "Standalone value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		name := cmd.Name()
		assert.Equal(t, "bar", name, "Name should match (Windows)")
		version := cmd.Value()
		assert.Equal(t, "bar Windows", version, "Value should match (Windows)")
		standalone := cmd.Standalone()
		assert.True(t, standalone, "Standalone value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		name := cmd.Name()
		assert.Equal(t, "baz", name, "Names should match (OSX)")
		version := cmd.Value()
		assert.Equal(t, "baz OSX", version, "Value should match (OSX)")
		standalone := cmd.Standalone()
		assert.True(t, standalone, "Standalone value should match (OSX)")
	}
	resetProjectDir(t)
}

func TestVariables(t *testing.T) {
	setProjectDir(t)
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	variables := prj.Variables()
	assert.Equal(t, 1, len(variables), "Should match 1 out of three constrained items")

	variable := variables[0]

	if runtime.GOOS == "linux" {
		name := variable.Name()
		assert.Equal(t, "foo", name, "Names should match (Linux)")
		value := variable.Value()
		assert.Equal(t, "foo Linux", value, "Value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		name := variable.Name()
		assert.Equal(t, "bar", name, "Name should match (Windows)")
		value := variable.Value()
		assert.Equal(t, "bar Windows", value, "Value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		name := variable.Name()
		assert.Equal(t, "baz", name, "Names should match (OSX)")
		value := variable.Value()
		assert.Equal(t, "baz OSX", value, "Value should match (OSX)")
	}
	resetProjectDir(t)
}
