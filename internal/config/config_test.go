package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfig_SetGetString(t *testing.T) {
	dir := os.TempDir()
	defer os.RemoveAll(dir)

	_, err := os.Create(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("Could not create config file: %v", err)
	}

	config := NewConfig([]string{dir}...)
	err = config.Set("TestKey", "TestValue")
	if err != nil {
		t.Fatalf("Could not set config value: %v", err)
	}

	val := config.GetString("TestKey")
	if val != "TestValue" {
		t.Fail()
	}

	val = config.GetString("testkey")
	if val != "TestValue" {
		t.Fail()
	}
}

func TestConfig_SetWrite(t *testing.T) {
	dir := os.TempDir()
	defer os.RemoveAll(dir)

	_, err := os.Create(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("Could not create config file: %v", err)
	}

	config := NewConfig([]string{dir}...)
	err = config.Set("testkey", "value")
	if err != nil {
		t.Fatalf("Could not set config value: %v", err)
	}

	configFilePath := filepath.Join(dir, config.configName)
	data, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		t.Fatalf("Could not read config file data: %v", err)
	}

	if !strings.Contains(string(data), "testkey: value") {
		t.Fatalf("Config file data does not contain expected values, got: %s", string(data))
	}
}

func TestConfig_MergeExisting(t *testing.T) {
	dir := os.TempDir()
	defer os.RemoveAll(dir)

	f, err := os.Create(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("Could not create config file: %v", err)
	}

	_, err = f.Write([]byte("exists: value"))
	if err != nil {
		t.Fatalf("Could not write data to config file: %v", err)
	}

	config := NewConfig([]string{dir}...)
	err = config.Set("newkey", "newValue")
	if err != nil {
		t.Fatalf("Could not set config value: %v", err)
	}

	configFilePath := filepath.Join(dir, config.configName)
	data, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		t.Fatalf("Could not read config file data: %v", err)
	}

	if !strings.Contains(string(data), "exists: value") {
		t.Fatalf("Config file data does not contain existing value: %s", string(data))
	}

	if !strings.Contains(string(data), "newkey: newValue") {
		t.Fatalf("Config file data does not contain updated values: %s", string(data))
	}
}
