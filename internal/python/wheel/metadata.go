package wheel

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/BurntSushi/toml"
)

// MissingMetadataError reports which of name and version neither the caller nor
// pyproject.toml supplied, and what pyproject.toml said about them, so the user
// can be told how to provide them.
type MissingMetadataError struct {
	Fields []string
	// Pyproject is the pyproject.toml that was read, or empty when there is none.
	Pyproject string
	// Dynamic holds the missing fields pyproject.toml lists in [project].dynamic,
	// which a build backend computes and so cannot be read from the file.
	Dynamic []string
}

func (e *MissingMetadataError) Error() string {
	return "package " + strings.Join(e.Fields, " and ") + " could not be determined"
}

func (e *MissingMetadataError) Is(target error) bool {
	return target == ErrMissingMetadata
}

// ResolveMetadata reads srcDir's pyproject.toml [project] table and applies the
// non-empty fields of override on top, producing the metadata to pack with. It
// returns a *MissingMetadataError when neither source supplies a name or
// version. Requires always comes from pyproject.toml's dependencies.
func ResolveMetadata(srcDir string, override Metadata) (*Metadata, error) {
	path := filepath.Join(srcDir, "pyproject.toml")
	proj, found, err := readPyProject(path)
	if err != nil {
		return nil, err
	}

	meta := Metadata{
		Name:     firstNonEmpty(override.Name, proj.Name),
		Version:  firstNonEmpty(override.Version, proj.Version),
		Summary:  firstNonEmpty(override.Summary, proj.Description),
		Requires: proj.Dependencies,
	}
	missing := &MissingMetadataError{}
	if found {
		missing.Pyproject = path
	}
	for field, value := range map[string]string{"name": meta.Name, "version": meta.Version} {
		if value != "" {
			continue
		}
		missing.Fields = append(missing.Fields, field)
		if slices.Contains(proj.Dynamic, field) {
			missing.Dynamic = append(missing.Dynamic, field)
		}
	}
	if len(missing.Fields) > 0 {
		slices.Sort(missing.Fields) // "name" before "version", regardless of map order
		slices.Sort(missing.Dynamic)
		return nil, missing
	}
	return &meta, nil
}

type pyProject struct {
	Name         string
	Version      string
	Description  string
	Dependencies []string
	Dynamic      []string
}

// readPyProject reads the [project] table from a pyproject.toml and reports
// whether the file exists. A missing file is not an error; caller-supplied
// metadata may stand in for it.
func readPyProject(path string) (pyProject, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return pyProject{}, false, nil
		}
		return pyProject{}, false, errs.Wrap(err, "could not read %s", path)
	}

	var parsed struct {
		Project struct {
			Name         string   `toml:"name"`
			Version      string   `toml:"version"`
			Description  string   `toml:"description"`
			Dependencies []string `toml:"dependencies"`
			Dynamic      []string `toml:"dynamic"`
		} `toml:"project"`
	}
	if err := toml.Unmarshal(data, &parsed); err != nil {
		return pyProject{}, true, errs.Wrap(err, "could not parse %s", path)
	}
	return pyProject{
		Name:         parsed.Project.Name,
		Version:      parsed.Project.Version,
		Description:  parsed.Project.Description,
		Dependencies: parsed.Project.Dependencies,
		Dynamic:      parsed.Project.Dynamic,
	}, true, nil
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
