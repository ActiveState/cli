package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/imdario/mergo"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v2"
)

type Config struct {
	configName  string
	configFile  string
	configPaths []string
	rwLock      *sync.RWMutex
	data        map[string]interface{}
}

// TODO: Maybe require a config path?
func NewConfig(configPaths ...string) *Config {
	return &Config{
		configName:  "config.yaml",
		data:        make(map[string]interface{}),
		configPaths: configPaths,
		rwLock:      &sync.RWMutex{},
	}
}

func (c *Config) AddConfigPaths(paths ...string) {
	c.configPaths = append(c.configPaths, paths...)
}

func (c *Config) ReadInConfig() error {
	var err error
	c.configFile, err = c.getConfigFile()
	if err != nil {
		return errs.Wrap(err, "Could not find config file")
	}

	configData, err := ioutil.ReadFile(c.configFile)
	if err != nil {
		return errs.Wrap(err, "Could not read config file")
	}

	err = yaml.Unmarshal(configData, c.data)
	if err != nil {
		return errs.Wrap(err, "Could not unmarshall config data")
	}

	return nil
}

func (c *Config) AllKeys() []string {
	// TODO: This will change drastically if we use nested keys
	var keys []string
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}

func (c *Config) get(key string) interface{} {
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()
	key = strings.ToLower(key)
	return c.data[key]
}

func (c *Config) GetString(key string) string {
	return cast.ToString(c.get(key))
}

func (c *Config) GetStringMapStringSlice(key string) map[string][]string {
	return cast.ToStringMapStringSlice(c.get(key))
}

func (c *Config) GetBool(key string) bool {
	return cast.ToBool(c.get(key))
}

func (c *Config) GetStringSlice(key string) []string {
	return cast.ToStringSlice(c.get(key))
}

func (c *Config) GetTime(key string) time.Time {
	return cast.ToTime(c.get(key))
}

func (c *Config) GetStringMap(key string) map[string]interface{} {
	return cast.ToStringMap(c.get(key))
}

func (c *Config) getConfigFile() (string, error) {
	if c.configFile != "" {
		return c.configFile, nil
	}

	if len(c.configPaths) < 1 {
		return "", errs.New("No config path(s) set")
	}

	for _, path := range c.configPaths {
		path = filepath.Join(path, c.configName)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			c.configFile = path
			return path, nil
		}
	}

	c.configFile = filepath.Join(c.configPaths[0], c.configName)
	return c.configFile, nil
}

func (c *Config) Set(key string, value interface{}) error {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()
	key = strings.ToLower(key)

	err := c.ReadInConfig()
	if err != nil {
		return err
	}

	c.data[key] = value

	return c.save()
}

func (c *Config) SetDefault(key string, value interface{}) error {
	// TODO: This should be writing to a separate map that contains
	// default values.
	// It does not appear that we are using the defaul functionality
	// anywhere. Further when the Config is written there is nothing in
	// the Config that denotes a default key as the default.
	return c.Set(key, value)
}

// TODO: This function may not be necessary since we are reading in
// the config everytime we write
func (c *Config) mergeInConfig() error {
	f, err := c.getConfigFile()
	if err != nil {
		return err
	}

	configData, err := ioutil.ReadFile(f)
	if err != nil {
		return errs.Wrap(err, "Could not read config file")
	}

	fileData := make(map[string]interface{})
	err = yaml.Unmarshal(configData, fileData)
	if err != nil {
		return errs.Wrap(err, "Could not unmarshall config data")
	}

	err = mergo.Merge(&c.data, fileData)
	if err != nil {
		return errs.Wrap(err, "Could not merge config maps")
	}

	return nil
}

func (c *Config) save() error {
	err := c.ReadInConfig()
	if err != nil {
		return err
	}

	f, err := os.Create(c.configFile)
	if err != nil {
		return errs.Wrap(err, "Could not create/open config file")
	}
	defer f.Close()

	data, err := yaml.Marshal(c.data)
	if err != nil {
		return errs.Wrap(err, "Could not marshal config data")
	}

	_, err = f.Write(data)
	if err != nil {
		return errs.Wrap(err, "Could not write config file")
	}

	return nil
}
