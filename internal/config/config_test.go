package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	C "github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/print"
	"github.com/shibukawa/configdir"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	funk "github.com/thoas/go-funk"
)

func init() {
	defer shutdown()
}

func setup(t *testing.T) {
	configDirs = configdir.New(configNamespace, "cli")
	configDirs.LocalPath, _ = filepath.Abs(".")
	configDir = configDirs.QueryFolders(configdir.Global)[0]

	viper.Reset()

	configPath := filepath.Join(configDir.Path, C.ConfigFileName)

	if _, err := os.Stat(configPath); err == nil {
		err := os.Remove(configPath)
		if err != nil {
			panic(err.Error())
		}
	}
}

func shutdown() {
	os.RemoveAll(configDir.Path)
}

func TestInit(t *testing.T) {
	setup(t)

	assert := assert.New(t)

	assert.Equal(false, configDir.Exists(C.ConfigFileName), "Config dir should not exist")

	ensureConfigExists()

	assert.Equal(true, configDir.Exists(C.ConfigFileName), "Config dir should exist")
}

func TestInitCorrupt(t *testing.T) {
	setup(t)

	assert := assert.New(t)

	file, _ := configDir.Create(C.ConfigFileName)
	file.Close()

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

	viper.Reset()
	readInConfig()

	assert.Equal(1, exitCode, "Config should fail to parse")
}

func TestSave(t *testing.T) {
	setup(t)

	assert := assert.New(t)
	path := filepath.Join(configDir.Path, C.ConfigFileName)

	if !configDir.Exists(C.ConfigFileName) {
		file, _ := configDir.Create(C.ConfigFileName)
		file.Close()
	}

	// Prepare viper, which is a library that automates configuration
	// management between files, env vars and the CLI
	viper.SetConfigName(C.ConfigName)
	viper.SetConfigType(C.ConfigFileType)
	viper.AddConfigPath(configDir.Path)

	print.Line(configDir.Path)

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
