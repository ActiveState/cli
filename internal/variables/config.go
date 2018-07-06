package variables

import (
	"flag"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/surveyor"
	"github.com/spf13/viper"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

// Struct for marshalling/unmarshalling user-defined variable values from/into
// via viper.
type configVariable struct {
	Name    string
	Value   string
	Project string
}

var testValue string // for unit tests

// ConfigValue returns the value stored in the user config file for the variable
// with the given name and project. If no value exists, the user is prompted for
// one and the result is stored in the user config file.
func ConfigValue(name string, project string) string {
	config := []configVariable{}
	if err := viper.UnmarshalKey("variables", &config); err != nil {
		logging.Errorf("Unable to read user-configured variables: %s", err)
		return "" // this should not happen
	}
	// Lookup existing variable value and return it.
	for _, variable := range config {
		if variable.Name == name && variable.Project == project {
			return variable.Value
		}
	}
	// Prompt the user for a variable value and save it.
	var value string
	if flag.Lookup("test.v") != nil {
		value = testValue
	}
	if value == "" {
		prompt := &survey.Input{Message: locale.Tt("config_variable_prompt_value", map[string]string{"Name": name})}
		if err := survey.AskOne(prompt, &value, surveyor.ValidateRequired); err != nil {
			return "" // do not save if cancelled
		}
	}
	config = append(config, configVariable{Name: name, Value: value, Project: project})
	viper.Set("variables", config)
	viper.WriteConfig()
	return value
}
