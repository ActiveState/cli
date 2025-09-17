package envdef

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/fileutils"
)

// EnvironmentDefinition provides all the information needed to set up an
// environment in which the packaged artifact contents can be used.
type EnvironmentDefinition struct {
	// Env is a list of environment variables to be set
	Env []EnvironmentVariable `json:"env"`

	// Transforms is a list of file transformations
	Transforms []FileTransform `json:"file_transforms"`

	// InstallDir is the directory (inside the artifact tarball) that needs to be installed on the user's computer
	InstallDir string `json:"installdir"`
}

func (e *EnvironmentDefinition) Save(path string) error {
	path = filepath.Join(path, EnvironmentDefinitionFilename)
	b, err := json.Marshal(e)
	if err != nil {
		return errs.Wrap(err, "Could not marshal environment definition")
	}
	if err := fileutils.WriteFile(path, b); err != nil {
		return errs.Wrap(err, "Could not write environment definition file")
	}
	return nil
}

// EnvironmentVariable defines a single environment variable and its values
type EnvironmentVariable struct {
	Name      string       `json:"env_name"`
	Values    []string     `json:"values"`
	Join      VariableJoin `json:"join"`
	Inherit   bool         `json:"inherit"`
	Separator string       `json:"separator"`
}

// VariableJoin defines a strategy to join environment variables together
type VariableJoin int

const (
	// Prepend indicates that new variables should be prepended
	Prepend VariableJoin = iota
	// Append indicates that new variables should be prepended
	Append
	// Disallowed indicates that there must be only one value for an environment variable
	Disallowed
)

// MarshalText marshals a join directive for environment variables
func (j *VariableJoin) MarshalText() ([]byte, error) {
	if j == nil {
		return nil, errs.New("MarshalText called on nil pointer")
	}

	var res string
	switch *j {
	default:
		res = "prepend"
	case Append:
		res = "append"
	case Disallowed:
		res = "disallowed"
	}
	return []byte(res), nil
}

func (j VariableJoin) MarshalJSON() ([]byte, error) {
	var res string
	switch j {
	default:
		res = "prepend"
	case Append:
		res = "append"
	case Disallowed:
		res = "disallowed"
	}
	return json.Marshal(res)
}

// UnmarshalText un-marshals a join directive for environment variables
func (j *VariableJoin) UnmarshalText(text []byte) error {
	switch string(text) {
	case "prepend":
		*j = Prepend
	case "append":
		*j = Append
	case "disallowed":
		*j = Disallowed
	default:
		return fmt.Errorf("Invalid join directive %s", string(text))
	}
	return nil
}

// UnmarshalJSON unmarshals an environment variable
// It sets default values for Inherit, Join and Separator if they are not specified
func (ev *EnvironmentVariable) UnmarshalJSON(data []byte) error {
	type evAlias EnvironmentVariable
	v := &evAlias{
		Inherit:   true,
		Separator: ":",
		Join:      Prepend,
	}

	err := json.Unmarshal(data, v)
	if err != nil {
		return err
	}

	*ev = EnvironmentVariable(*v)
	return nil
}

// NewEnvironmentDefinition returns an environment definition unmarshaled from a
// file
func NewEnvironmentDefinition(fp string) (*EnvironmentDefinition, error) {
	blob, err := os.ReadFile(fp)
	if err != nil {
		return nil, errs.Wrap(err, "could not read environment definition file: %s", fp)
	}
	ed := &EnvironmentDefinition{}
	err = json.Unmarshal(blob, ed)
	if err != nil {
		return nil, errs.Wrap(err, "could not unmarshal environment definition file: %s", fp)
	}

	if ignores := os.Getenv(constants.IgnoreEnvEnvVarName); ignores != "" {
		ignore := make(map[string]bool)
		for _, name := range strings.Split(ignores, ",") {
			ignore[name] = true
		}

		// Remove any environment variables to ignore.
		ed.Env = sliceutils.Filter(ed.Env, func(e EnvironmentVariable) bool {
			_, exists := ignore[e.Name]
			return !exists
		})
	}

	return ed, nil
}

// WriteFile marshals an environment definition to a file
func (ed *EnvironmentDefinition) WriteFile(filepath string) error {
	blob, err := ed.Marshal()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath, blob, 0666)
}

// Marshal marshals an environment definition to a file
func (ed *EnvironmentDefinition) Marshal() ([]byte, error) {
	blob, err := json.MarshalIndent(ed, "", "  ")
	if err != nil {
		return []byte(""), err
	}
	return blob, nil
}

// ExpandVariables expands substitution strings specified in the environment variable values.
// Right now, the only valid substition string is `${INSTALLDIR}` which is being replaced
// with the base of the installation directory for a given project
func (ed *EnvironmentDefinition) ExpandVariables(constants Constants) *EnvironmentDefinition {
	res := ed
	for k, v := range constants {
		res = ed.ReplaceString(fmt.Sprintf("${%s}", k), v)
	}
	return res
}

// ReplaceString replaces the string `from` with its `replacement` value
// in every environment variable value
func (ed *EnvironmentDefinition) ReplaceString(from string, replacement string) *EnvironmentDefinition {
	res := ed
	newEnv := make([]EnvironmentVariable, 0, len(ed.Env))
	for _, ev := range ed.Env {
		newEnv = append(newEnv, ev.ReplaceString(from, replacement))
	}
	res.Env = newEnv
	return res
}

// Merge merges two environment definitions according to the join strategy of
// the second one.
//   - Environment variables that are defined in both definitions, are merged with
//     EnvironmentVariable.Merge() and added to the result
//   - Environment variables that are defined in only one of the two definitions,
//     are added to the result directly
func (ed *EnvironmentDefinition) Merge(other *EnvironmentDefinition) (*EnvironmentDefinition, error) {
	res := *ed
	if other == nil {
		return &res, nil
	}

	newEnv := []EnvironmentVariable{}

	thisEnvNames := funk.Map(
		ed.Env,
		func(x EnvironmentVariable) string { return x.Name },
	).([]string)

	newKeys := make([]string, 0, len(other.Env))
	otherEnvMap := map[string]EnvironmentVariable{}
	for _, ev := range other.Env {
		if !funk.ContainsString(thisEnvNames, ev.Name) {
			newKeys = append(newKeys, ev.Name)
		}
		otherEnvMap[ev.Name] = ev
	}

	// add new keys to environment
	for _, k := range newKeys {
		oev := otherEnvMap[k]
		newEnv = append(newEnv, oev)
	}

	// merge keys
	for _, ev := range ed.Env {
		otherEv, ok := otherEnvMap[ev.Name]
		if !ok {
			// if key exists only in this variable, use it
			newEnv = append(newEnv, ev)
		} else {
			// otherwise: merge this variable and the other environment variable
			mev, err := ev.Merge(otherEv)
			if err != nil {
				return &res, err
			}
			newEnv = append(newEnv, *mev)
		}
	}
	res.Env = newEnv
	return &res, nil
}

// ReplaceString replaces the string 'from' with 'replacement' in
// environment variable values
func (ev *EnvironmentVariable) ReplaceString(from string, replacement string) EnvironmentVariable {
	res := *ev
	values := make([]string, 0, len(ev.Values))

	for _, v := range ev.Values {
		values = append(values, strings.ReplaceAll(v, from, replacement))
	}
	res.Values = values
	return res
}

// Merge merges two environment variables according to the join strategy defined by
// the second environment variable
// If join strategy of the second variable is "prepend" or "append", the values
// are prepended or appended to the first variable.
// If join strategy is set to "disallowed", the variables need to have exactly
// one value, and both merged values need to be identical, otherwise an error is
// returned.
func (ev *EnvironmentVariable) Merge(other EnvironmentVariable) (*EnvironmentVariable, error) {
	res := *ev

	// separators and inherit strategy always need to match for two merged variables
	if ev.Separator != other.Separator || ev.Inherit != other.Inherit {
		return nil, fmt.Errorf("cannot merge environment definitions: incompatible `separator` or `inherit` directives")
	}

	// 'disallowed' join strategy needs to be set for both or none of the variables
	if (ev.Join == Disallowed || other.Join == Disallowed) && ev.Join != other.Join {
		return nil, fmt.Errorf("cannot merge environment definitions: incompatible `join` directives")
	}

	switch other.Join {
	case Prepend:
		res.Values = filterValuesUniquely(append(other.Values, ev.Values...), true)
	case Append:
		res.Values = filterValuesUniquely(append(ev.Values, other.Values...), false)
	case Disallowed:
		if len(ev.Values) != 1 || len(other.Values) != 1 || (ev.Values[0] != other.Values[0]) {
			sep := ev.Separator
			// It's possible that the merged env vars will still be equal, so only error if they are not.
			if strings.Join(ev.Values, sep) != strings.Join(other.Values, sep) {
				return nil, fmt.Errorf(
					"cannot merge environment definitions: no join strategy for variable %s with values %s and %s",
					ev.Name,
					strings.Join(ev.Values, sep), strings.Join(other.Values, sep),
				)
			}

		}
	default:
		return nil, fmt.Errorf("could not join environment variable %s: invalid `join` directive %v", ev.Name, other.Join)
	}
	res.Join = other.Join
	return &res, nil
}

// filterValuesUniquely removes duplicate entries from a list of strings
// If `keepFirst` is true, only the first occurrence is kept, otherwise the last
// one.
func filterValuesUniquely(values []string, keepFirst bool) []string {
	nvs := make([]*string, len(values))
	posMap := map[string][]int{}

	for i, v := range values {
		pmv, ok := posMap[v]
		if !ok {
			pmv = []int{}
		}
		pmv = append(pmv, i)
		posMap[v] = pmv
	}

	var getPos func([]int) int
	if keepFirst {
		getPos = func(x []int) int { return x[0] }
	} else {
		getPos = func(x []int) int { return x[len(x)-1] }
	}

	for v, positions := range posMap {
		pos := getPos(positions)
		cv := v
		nvs[pos] = &cv
	}

	res := make([]string, 0, len(values))
	for _, nv := range nvs {
		if nv != nil {
			res = append(res, *nv)
		}
	}
	return res
}

// ValueString joins the environment variable values into a single string
// If duplicate values are found, only one of them is considered: for join
// strategy `prepend` only the first occurrence, for join strategy `append` only
// the last one.
func (ev *EnvironmentVariable) ValueString() string {
	return strings.Join(
		filterValuesUniquely(ev.Values, ev.Join == Prepend),
		ev.Separator)
}

// getEnvBasedOn returns the environment variable names and values defined by
// the EnvironmentDefinition.
// If an environment variable is configured to inherit from the base
// environment (`Inherit==true`), the base environment defined by the
// `envLookup` method is joined with these environment variables.
// This function is mostly used for testing. Use GetEnv() in production.
func (ed *EnvironmentDefinition) getEnvBasedOn(envLookup map[string]string) (map[string]string, error) {
	res := maps.Clone(envLookup)

	// On Windows, environment variable names are case-insensitive.
	// For example, it uses "Path", but responds to "PATH" as well.
	// This causes trouble with our environment merging, which will end up adding "PATH" (with the
	// correct value) alongside "Path" (with the old value).
	// In order to remedy this, track the OS-specific environment variable name and if it's
	// modified/merged, replace it with our version (e.g. "Path" -> "PATH"). We do not use the OS name
	// because we assume ours is the one that's used elsewhere in the codebase, and Windows will
	// properly respond to a changed-case name anyway.
	osEnvNames := map[string]string{}
	if runtime.GOOS == "windows" {
		for k := range envLookup {
			osEnvNames[strings.ToLower(k)] = k
		}
	}

	for _, ev := range ed.Env {
		pev := &ev
		osName := pev.Name
		if runtime.GOOS == "windows" {
			if name, ok := osEnvNames[strings.ToLower(osName)]; ok {
				osName = name
			}
		}
		osValue, hasOsValue := envLookup[osName]
		if pev.Inherit {
			if hasOsValue {
				osEv := ev
				osEv.Values = []string{osValue}
				var err error
				pev, err = osEv.Merge(ev)
				if err != nil {
					return nil, err
				}
			}
		} else if hasOsValue {
			res[pev.Name] = "" // unset
		}
		// only add environment variable if at least one value is set (This allows us to remove variables from the environment.)
		if len(ev.Values) > 0 {
			res[pev.Name] = pev.ValueString()
			if pev.Name != osName {
				// On Windows, delete the redundant (case-insensitive) version that our case-sensitive
				// version could conflict with. (Our version has already processed the value of the
				// redundant version.)
				// For example, delete "Path" while preserving our "PATH".
				delete(res, osName)
			}
		}
	}
	return res, nil
}

// GetEnv returns the environment variable names and values defined by
// the EnvironmentDefinition.
// If an environment variable is configured to inherit from the OS
// environment (`Inherit==true`), the base environment defined by the
// `envLookup` method is joined with these environment variables.
func (ed *EnvironmentDefinition) GetEnv(inherit bool) map[string]string {
	lookupEnv := map[string]string{}
	if inherit {
		lookupEnv = osutils.EnvSliceToMap(os.Environ())
	}
	res, err := ed.getEnvBasedOn(lookupEnv)
	if err != nil {
		panic(fmt.Sprintf("Could not inherit OS environment variable: %v", err))
	}
	return res
}

func FilterPATH(env map[string]string, excludes ...string) {
	PATH, exists := env["PATH"]
	if !exists {
		return
	}

	newPaths := []string{}
	paths := strings.Split(PATH, string(os.PathListSeparator))
	for _, p := range paths {
		pc := filepath.Clean(p)
		includePath := true
		for _, exclude := range excludes {
			if pc == filepath.Clean(exclude) {
				includePath = false
				break
			}
		}
		if includePath {
			newPaths = append(newPaths, p)
		}
	}

	env["PATH"] = strings.Join(newPaths, string(os.PathListSeparator))
}

type ExecutablePaths []string

func (ed *EnvironmentDefinition) ExecutablePaths() (ExecutablePaths, error) {
	env := ed.GetEnv(false)

	// Retrieve artifact binary directory
	var bins []string
	if p, ok := env["PATH"]; ok {
		bins = strings.Split(p, string(os.PathListSeparator))
	}

	exes, err := osutils.Executables(bins)
	if err != nil {
		return nil, errs.Wrap(err, "Could not detect executables")
	}

	// Remove duplicate executables as per PATH and PATHEXT
	exes, err = osutils.UniqueExes(exes, os.Getenv("PATHEXT"))
	if err != nil {
		return nil, errs.Wrap(err, "Could not detect unique executables, make sure your PATH and PATHEXT environment variables are properly configured.")
	}

	return exes, nil
}

func (ed *EnvironmentDefinition) ExecutableDirs() (ExecutablePaths, error) {
	exes, err := ed.ExecutablePaths()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get executable paths")
	}

	var dirs ExecutablePaths
	for _, p := range exes {
		dirs = append(dirs, filepath.Dir(p))
	}
	dirs = funk.UniqString(dirs)

	return dirs, nil
}

// FindBinPathFor returns the PATH directory in which the executable can be found.
// If the executable cannot be found, an empty string is returned.
// This function should be called after variables names are expanded with ExpandVariables()
func (ed *EnvironmentDefinition) FindBinPathFor(executable string) string {
	for _, ev := range ed.Env {
		if ev.Name == "PATH" {
			for _, dir := range ev.Values {
				if fileutils.TargetExists(filepath.Join(dir, executable)) {
					return filepath.Clean(filepath.FromSlash(dir))
				}
			}
		}
	}
	return ""
}

func (ed *EnvironmentDefinition) NeedsTransforms() bool {
	return len(ed.Transforms) > 0
}

func (ed *EnvironmentDefinition) InstallationDir() string {
	return ed.InstallDir
}
