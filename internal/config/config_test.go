package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	C "github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/shibukawa/configdir"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	funk "github.com/thoas/go-funk"
)

func setup(t *testing.T) {
	configNamespace = C.ConfigNamespace + "-test"

	configDirs = configdir.New(configNamespace, "cli")
	configDirs.LocalPath, _ = filepath.Abs(".")
	configDir = configDirs.QueryFolders(configdir.Global)[0]

	os.RemoveAll(configDir.Path)

	defer shutdown(t)
}

func shutdown(t *testing.T) {
	os.RemoveAll(configDir.Path)
}

func TestInit(t *testing.T) {
	setup(t)

	assert := assert.New(t)

	assert.Equal(false, configDir.Exists(C.ConfigFileName), "Config dir should not exist")

	Init()

	assert.Equal(true, configDir.Exists(C.ConfigFileName), "Config dir should exist")
}

func TestInitCorrupt(t *testing.T) {
	setup(t)

	assert := assert.New(t)

	configDir.Create(C.ConfigFileName)

	data := []byte("&")
	path := filepath.Join(configDir.Path, C.ConfigFileName)
	err := ioutil.WriteFile(path, data, 0644)

	if err != nil {
		t.Fatal(err)
	}

	exitCode := 0
	exit = func(code int) {
		exitCode = 1
	}

	Init()

	assert.Equal(1, exitCode, "Config should fail to parse")
}

func TestSave(t *testing.T) {
	setup(t)

	assert := assert.New(t)
	path := filepath.Join(configDir.Path, C.ConfigFileName)

	if !configDir.Exists(C.ConfigFileName) {
		configDir.Create(C.ConfigFileName)
	}

	// Prepare viper, which is a library that automates configuration
	// management between files, env vars and the CLI
	viper.SetConfigName(C.ConfigName)
	viper.SetConfigType(C.ConfigFileType)
	viper.AddConfigPath(configDir.Path)

	if err := viper.ReadInConfig(); err != nil {
		t.Fatal(err)
	}

	viper.Set("Foo", "bar")

	Save()

	dat, err := ioutil.ReadFile(path)

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(true, funk.Contains(string(dat), "foo: bar"), "Config should contain our newly added field")
}
