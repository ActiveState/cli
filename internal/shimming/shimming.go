package shimming

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

type Shimable interface {
	Binaries() []string

	BinariesToShim(relativePaths []string) map[string]string
}

type Shim struct {
	binaries []string
}

func (s *Shim) Binaries() []string {
	return s.binaries
}

func (s *Shim) BinariesToShim(relativePaths []string) map[string]string {
	result := map[string]string{}
	for _, bin := range s.binaries {
		if _, exists := result[bin]; exists {
			continue
		}
		for _, path := range relativePaths {
			for _, suffix := range binarySuffixes {
				binPath := filepath.Join(path, bin+suffix)
				if fileutils.FileExists(binPath) {
					result[bin] = binPath
				}
			}
		}
	}

	return result
}

func NewShim(binaries []string) Shimable {
	return &Shim{binaries}
}

type Collection struct {
	shims []Shimable
}

func InitCollection() *Collection {
	c := NewCollection()
	c.RegisterDefaults()
	return c
}

func NewCollection() *Collection {
	return &Collection{}
}

func (c *Collection) RegisterShim(shim Shimable) {
	c.shims = append(c.shims, shim)
}

func (c *Collection) RegisterDefaults() {
	c.RegisterShim(NewShim([]string{
		"pip", "pip2", "pip3",
	}))
}

func (c *Collection) Shims() []Shimable {
	return c.shims
}

func (c *Collection) BinariesToShim(relativePaths []string) map[string]string {
	result := map[string]string{}
	for _, shim := range c.shims {
		binaries := shim.BinariesToShim(relativePaths)
		if len(binaries) > 0 {
			for k, v := range binaries {
				if _, exists := result[k]; exists {
					logging.Warning("Shim binary: %s matches multiple binaries", k)
				}
				result[k] = v
			}
		}
	}

	return result
}

func (c *Collection) ShimBinaries(relativePaths []string) (string, *failures.Failure) {
	binariesToShim := c.BinariesToShim(relativePaths)
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", failures.FailIO.Wrap(err)
	}

	if len(binariesToShim) == 0 {
		return "", nil
	}

	for name, path := range binariesToShim {
		data := fmt.Sprintf(`%s%s shim %s %s`, forwardHeader, os.Args[0], path, forwardArgs)
		binPath := filepath.Join(dir, name+forwardSuffix)
		if fail := fileutils.WriteFile(binPath, []byte(data)); fail != nil {
			return "", fail
		}

		if fail := fileutils.MakeExecutable(binPath); fail != nil {
			return "", fail
		}
	}

	return dir, nil
}
