package variables

import (
	"strings"

	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/projectfile"

	funk "github.com/thoas/go-funk"
)

// This is mostly a clone of the hooks cmdlet. Any bugfixes and changes to that
// file should be applied here and vice-versa.

// HashedVariable to easily associate a Variable struct to a hash of itself
type HashedVariable struct {
	Variable projectfile.Variable
	Hash     string
}

// GetEffectiveVariables returns effective variables, meaning only the ones that apply to the current runtime environment
func GetEffectiveVariables() []*projectfile.Variable {
	project := projectfile.Get()
	variables := []*projectfile.Variable{}

	for _, variable := range project.Variables {
		if !constraints.IsConstrained(variable.Constraints) {
			variables = append(variables, &variable)
		}
	}

	return variables
}

// HashVariables returns a map of all the variables with the keys being a hash of that variable
func HashVariables(variables []projectfile.Variable) (map[string]projectfile.Variable, error) {
	hashedVariables := make(map[string]projectfile.Variable)
	for _, variable := range variables {
		hash, err := variable.Hash()
		// If we can't hash, something is really wrong so fail gracefully
		if err != nil {
			return nil, err
		}
		hashedVariables[hash] = variable
	}
	return hashedVariables, nil
}

// VariableExists Returns true if this variable is already defined
func VariableExists(variable projectfile.Variable, project *projectfile.Project) (bool, error) {
	newVariableHash, err := variable.Hash()
	if err != nil {
		return false, err
	}
	variables, err := HashVariables(project.Variables)
	if err != nil {
		return false, err
	}
	_, exists := variables[newVariableHash]
	return exists, nil
}

// HashVariablesFiltered is identical to HashVariables except that it takes a slice of names to be used as a filter
// If no variable provided does the same as MapVariables
// If no variables found for given variable names, returns nil
func HashVariablesFiltered(variables []projectfile.Variable, variableNames []string) (map[string]projectfile.Variable, error) {
	hashedVariables, err := HashVariables(variables)
	if err != nil {
		return nil, err
	}
	if len(variableNames) == 0 {
		return hashedVariables, err
	}

	hashedVariablesFiltered := make(map[string]projectfile.Variable)
	for hash, variable := range hashedVariables {
		if funk.Contains(variableNames, variable.Name) {
			hashedVariablesFiltered[hash] = variable
		}
	}

	return hashedVariablesFiltered, nil
}

// PromptOptions returns an array of strings that can be consumed by the survey library we use,
// the second return argument contains a map that connects each item to a hash
func PromptOptions(filter string) ([]string, map[string]string, error) {
	project := projectfile.Get()
	optionsMap := make(map[string]string)
	options := []string{}

	filters := []string{}
	if filter != "" {
		filters = append(filters, filter)
	}

	hashedVariables, err := HashVariablesFiltered(project.Variables, filters)
	if err != nil {
		return options, optionsMap, err
	}

	if len(hashedVariables) == 0 {
		return options, optionsMap, failures.FailUserInput.New("err_env_cannot_find")
	}

	for hash, variable := range hashedVariables {
		value := strings.Replace(variable.Value, "\n", " ", -1)
		if len(value) > 50 {
			value = value[0:50] + ".."
		}

		constraints := []string{}
		if variable.Constraints.Environment != "" {
			constraints = append(constraints, variable.Constraints.Environment)
		}
		if variable.Constraints.Platform != "" {
			constraints = append(constraints, variable.Constraints.Platform)
		}

		var constraintString string
		if len(constraints) > 0 {
			constraintString = strings.Join(constraints, ", ") + ", "
		}

		option := locale.T("prompt_env_option", map[string]interface{}{
			"Hash":        hash,
			"Variable":    variable.Name,
			"Value":       value,
			"Constraints": constraintString,
		})
		options = append(options, option)
		optionsMap[option] = hash
	}

	return options, optionsMap, nil
}
