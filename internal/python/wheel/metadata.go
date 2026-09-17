package wheel

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/BurntSushi/toml"
)

// ResolveMetadata reads srcDir's pyproject.toml [project] table and applies the
// non-empty fields of override on top, producing the metadata to pack with. It
// errors when neither source supplies a name or version.
func ResolveMetadata(srcDir string, override Metadata) (*Metadata, error) {
	proj, err := readPyProject(filepath.Join(srcDir, "pyproject.toml"))
	if err != nil {
		return nil, err
	}

	meta := Metadata{
		Name:    firstNonEmpty(override.Name, proj.Name),
		Version: firstNonEmpty(override.Version, proj.Version),
		Summary: firstNonEmpty(override.Summary, proj.Description),
	}
	if meta.Name == "" || meta.Version == "" {
		return nil, ErrMissingMetadata
	}
	return &meta, nil
}

type pyProject struct {
	Name        string
	Version     string
	Description string
}

// readPyProject reads the [project] table from a pyproject.toml. A missing file
// is not an error; caller-supplied metadata may stand in for it.
func readPyProject(path string) (pyProject, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return pyProject{}, nil
		}
		return pyProject{}, errs.Wrap(err, "could not read pyproject.toml")
	}

	var parsed struct {
		Project struct {
			Name        string `toml:"name"`
			Version     string `toml:"version"`
			Description string `toml:"description"`
		} `toml:"project"`
	}
	if err := toml.Unmarshal(data, &parsed); err != nil {
		return pyProject{}, errs.Wrap(err, "could not parse pyproject.toml")
	}
	return pyProject{
		Name:        parsed.Project.Name,
		Version:     parsed.Project.Version,
		Description: parsed.Project.Description,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

var (
	nameRunRe    = regexp.MustCompile(`[-_.]+`)
	versionRunRe = regexp.MustCompile(`[^A-Za-z0-9.]+`)
)

// normalizeName converts a distribution name to its wheel-filename form: runs of
// [-_.] collapse to a single underscore and the result is lowercased.
func normalizeName(name string) string {
	return strings.ToLower(nameRunRe.ReplaceAllString(name, "_"))
}

// escapeVersion replaces runs of characters not allowed in a wheel-filename
// version component with a single underscore.
func escapeVersion(version string) string {
	return versionRunRe.ReplaceAllString(version, "_")
}

func wheelFilename(name, version string) string {
	return normalizeName(name) + "-" + escapeVersion(version) + "-py3-none-any.whl"
}

func distInfoDir(name, version string) string {
	return normalizeName(name) + "-" + escapeVersion(version) + ".dist-info"
}
