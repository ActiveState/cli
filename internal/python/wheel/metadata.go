package wheel

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/BurntSushi/toml"
)

// resolvedMetadata is the metadata after merging caller overrides with
// pyproject.toml, with name and version guaranteed non-empty.
type resolvedMetadata struct {
	Name    string
	Version string
	Summary string
}

// resolveMetadata fills empty fields of override from srcDir's pyproject.toml and
// returns the result, erroring if name or version is set by neither source.
func resolveMetadata(srcDir string, override Metadata) (resolvedMetadata, error) {
	proj, err := readPyProject(filepath.Join(srcDir, "pyproject.toml"))
	if err != nil {
		return resolvedMetadata{}, err
	}

	res := resolvedMetadata{
		Name:    firstNonEmpty(override.Name, proj.Name),
		Version: firstNonEmpty(override.Version, proj.Version),
		Summary: firstNonEmpty(override.Summary, proj.Description),
	}
	if res.Name == "" || res.Version == "" {
		return resolvedMetadata{}, ErrMissingMetadata
	}
	return res, nil
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
