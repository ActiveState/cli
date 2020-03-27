package integration

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

type Dirs struct {
	base   string
	Config string
	Cache  string
	Bin    string
	Work   string
}

func NewDirs(base string) (*Dirs, error) {
	if base == "" {
		tmpDir, err := ioutil.TempDir("", "")
		if err != nil {
			return nil, err
		}
		base = tmpDir
	}

	config := filepath.Join(base, "config")
	cache := filepath.Join(base, "cache")
	bin := filepath.Join(base, "bin")
	work := filepath.Join(base, "work")

	subdirs := []string{config, cache, bin, work}
	for _, subdir := range subdirs {
		if err := os.MkdirAll(subdir, 0700); err != nil {
			return nil, err
		}
	}

	dirs := Dirs{
		base:   base,
		Config: config,
		Cache:  cache,
		Bin:    bin,
		Work:   work,
	}

	return &dirs, nil
}

func (d *Dirs) Close() error {
	return os.RemoveAll(d.base)
}
