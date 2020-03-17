package envdef

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// EnvironmentDefinition defines environment variables that need to be set for a
// runtime to work
type EnvironmentDefinition struct {
	Env        []EnvironmentVariable `json:env`
	InstallDir string                `json:installdir`
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
	Prepend VariableJoin = iota
	Append
	Disallowed
)

func (j VariableJoin) MarshalText() ([]byte, error) {
	var res string
	switch j {
	default:
		res = "prepend"
	case Append:
		res = "append"
	case Disallowed:
		res = "disallowed"
	}
	return []byte(res), nil
}

func (j *VariableJoin) UnmarshalText(text []byte) error {
	switch string(text) {
	case "prepend":
		*j = Prepend
	case "append":
		*j = Append
	case "disallowed":
		*j = Disallowed
	default:
		return fmt.Errorf("invalid join directive `%s`", string(text))
	}
	return nil
}

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

	blob, err := ioutil.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	ed := &EnvironmentDefinition{}
	err = json.Unmarshal(blob, ed)
	if err != nil {
		return nil, err
	}
	return ed, nil
}

// WriteFile marshals an environment definition to a file
func (ed *EnvironmentDefinition) WriteFile(filepath string) error {
	blob, err := json.MarshalIndent(ed, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath, blob, 0666)
}

// ReplaceInstallDir replaces the string '${INSTALLDIR}' with the actual
// installation directory in every environment variable value
func (ed *EnvironmentDefinition) ReplaceInstallDir(replacement string) *EnvironmentDefinition {
	res := ed
	newEnv := make([]EnvironmentVariable, 0, len(ed.Env))
	for _, ev := range ed.Env {
		newEnv = append(newEnv, ev.ReplaceInstallDir(replacement))
	}
	res.Env = newEnv
	return res
}

// Merge merges two environment definitions according to the join strategy of
// the second one.
func (ed *EnvironmentDefinition) Merge(other *EnvironmentDefinition) (*EnvironmentDefinition, error) {
	res := ed
	if other == nil {
		return res, nil
	}

	newEnv := []EnvironmentVariable{}

	thisEnvHas := map[string]struct{}{}

	for _, ev := range ed.Env {
		thisEnvHas[ev.Name] = struct{}{}
	}

	newKeys := make([]string, 0, len(other.Env))
	otherEnvMap := map[string]EnvironmentVariable{}
	for _, ev := range other.Env {
		if _, ok := thisEnvHas[ev.Name]; !ok {
			newKeys = append(newKeys, ev.Name)
		}
		otherEnvMap[ev.Name] = ev
	}

	// add new keys to environment
	for _, k := range newKeys {
		oev, ok := otherEnvMap[k]
		if !ok {
			panic("This should not happen")
		}
		newEnv = append(newEnv, oev)
	}

	// merge keys
	for _, ev := range ed.Env {
		otherEv, ok := otherEnvMap[ev.Name]
		if !ok {
			newEnv = append(newEnv, ev)
		} else {
			mev, err := ev.Merge(otherEv)
			if err != nil {
				return res, err
			}
			newEnv = append(newEnv, *mev)
		}
	}
	res.Env = newEnv
	return res, nil
}

// ReplaceInstallDir replaces the string '${INSTALLDIR}' with the actual
// installation directory
func (ev EnvironmentVariable) ReplaceInstallDir(replacement string) EnvironmentVariable {
	res := ev
	values := make([]string, 0, len(ev.Values))

	for _, v := range ev.Values {
		values = append(values, strings.ReplaceAll(v, "${INSTALLDIR}", replacement))
	}
	res.Values = values
	return res
}

// Merges two environment variables according to the join strategy defined by
// the second environment variable
func (ev *EnvironmentVariable) Merge(other EnvironmentVariable) (*EnvironmentVariable, error) {
	res := ev
	if ev.Separator != other.Separator || ev.Inherit != other.Inherit {
		return nil, fmt.Errorf("could not join environment variable %s, conflicting directives `inherit` or `separator`", ev.Name)
	}

	if (ev.Join == Disallowed || other.Join == Disallowed) && ev.Join != other.Join {
		return nil, fmt.Errorf("could not join environment variable %s, with conflicting `join` strategies %v and %v", ev.Name, ev.Join, other.Join)
	}

	switch other.Join {
	case Prepend:
		res.Values = append(other.Values, ev.Values...)
	case Append:
		res.Values = append(ev.Values, other.Values...)
	case Disallowed:
		if len(ev.Values) > 1 || len(other.Values) > 1 || (ev.Values[0] != other.Values[0]) {
			sep := string(ev.Separator)
			return nil, fmt.Errorf(
				"could not join environment variable %s: no join strategy with values %s and %s",
				ev.Name,
				strings.Join(ev.Values, sep), strings.Join(other.Values, sep))

		}
	default:
		return nil, fmt.Errorf("could not join environment variable %s: invalid `join` directive %v", ev.Name, other.Join)
	}
	res.Join = other.Join
	return res, nil
}

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
		string(ev.Separator))
}

// GetEnvBasedOn returns the environment variable names and values defined by
// the EnvironmentDefinition.
// If an environment variable is configured to inherit from the base
// environment (`Inherit==true`), the base environment defined by the
// `envLookup` method is joined with these environment variables.
// This function is mostly used for testing. Use GetEnv() in production.
func (ed *EnvironmentDefinition) GetEnvBasedOn(envLookup func(string) (string, bool)) (map[string]string, error) {
	res := map[string]string{}

	for _, ev := range ed.Env {
		pev := &ev
		if pev.Inherit {
			osValue, hasOsValue := envLookup(pev.Name)
			if hasOsValue {
				osEv := ev
				osEv.Values = []string{osValue}
				var err error
				pev, err = osEv.Merge(ev)
				if err != nil {
					return nil, err

				}
			}
		}
		res[pev.Name] = pev.ValueString()
	}
	return res, nil
}

// GetEnv returns the environment variable names and values defined by
// the EnvironmentDefinition.
// If an environment variable is configured to inherit from the OS
// environment (`Inherit==true`), the base environment defined by the
// `envLookup` method is joined with these environment variables.
func (ed *EnvironmentDefinition) GetEnv() map[string]string {
	res, err := ed.GetEnvBasedOn(os.LookupEnv)
	if err != nil {
		panic(fmt.Sprintf("Could not inherit OS environment variable: %v", err))
	}
	return res
}
