package projectfile

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/logging"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/mitchellh/mapstructure"
)

var (
	// FailParseVar describes failures due to parsing the value of a variable
	FailParseVar = failures.Type("projectfile.fail.parsevar")

	// FailValidateVarPullFrom describes a failure due to an invalid value for the pullfrom field
	FailValidateVarPullFrom = failures.Type("projectfile.fail.varpull", FailValidate)

	// FailValidateVarShare describes a failure due to an invalid value for the share field
	FailValidateVarShare = failures.Type("projectfile.fail.varshare", FailValidate)

	// FailValidateStaticValueWithPull described a failure due to the pullfrom and/or share fields being defined AS WELL AS a static value
	FailValidateStaticValueWithPull = failures.Type("projectfile.fail.varstaticwithpull", FailValidate)

	// FailValidateValueEmpty described a failure due to the pullfrom and/or share fields being defined AS WELL AS a static value
	FailValidateValueEmpty = failures.Type("projectfile.fail.varempty", FailValidate)
)

// VariablePullFrom records the scope of a variable, variables won't be exposed if we're not in this scope
type VariablePullFrom string

const (
	// VariablePullFromProject indicates that the value for a variable is tied to the project
	VariablePullFromProject VariablePullFrom = "project"

	// VariablePullFromOrg indicates that the value for a variable is tied to the organization
	VariablePullFromOrg = "organization"
)

// AcceptedValues is a helper method that returns all possible values for this type
func (v VariablePullFrom) AcceptedValues() []string {
	return []string{string(VariablePullFromProject), string(VariablePullFromOrg)}
}

// Validate checks whether we hold a string value that's actually valid for this type, because Go is not smart enough to
// assert this itself apparently
func (v VariablePullFrom) Validate() *failures.Failure {
	switch v {
	case VariablePullFromProject, VariablePullFromOrg:
		return nil
	default:
		return FailValidateVarPullFrom.New(locale.Tr("variables_err_invalid_pullfrom", string(v), strings.Join(v.AcceptedValues(), ", ")))
	}
}

// String returns a formatted representation of the underlying VariablePullFrom
// value. If the underlying value is nil, an empty string is returned.
func (v *VariablePullFrom) String() string {
	if v == nil {
		return ""
	}
	return string(*v)
}

// VariableShare records the owner of the variable, this determines who a variable might be shared with
type VariableShare string

const (
	// VariableShareOrg indicates that a variable can be shared at the organization level
	VariableShareOrg VariableShare = "organization"
)

// AcceptedValues is a helper method that returns all possible values for this type
func (v VariableShare) AcceptedValues() []string {
	return []string{string(VariableShareOrg)}
}

// Validate checks whether we hold a string value that's actually valid for this type, because Go is not smart enough to
// assert this itself apparently
func (v VariableShare) Validate() *failures.Failure {
	switch v {
	case VariableShareOrg:
		return nil
	default:
		return FailValidateVarShare.New(locale.Tr("variables_err_invalid_share", string(v), strings.Join(v.AcceptedValues(), ", ")))
	}
}

// String returns a formatted representation of the underlying VariableShare
// value. If the underlying value is nil, an empty string is returned.
func (v *VariableShare) String() string {
	if v == nil {
		return ""
	}
	return string(*v)
}

// VariableValue holds the value of the variable, since variables can have complex value (eg. they can be secrets), this
// needs a more complex type
type VariableValue struct {
	StaticValue *string
	PullFrom    *VariablePullFrom
	Share       *VariableShare
}

// Variable covers the variable structure, which goes under Project
type Variable struct {
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	ValueRaw    interface{}   `yaml:"value"`
	Value       VariableValue `yaml:"-"`
	Constraints Constraint    `yaml:"constraints"`
}

// Parse is called right after the yaml is unmarshalled, it serves to infer the true value of the "value" property
func (v *Variable) Parse() *failures.Failure {
	switch v.ValueRaw.(type) {
	case string, bool, int:
		pointableValue := fmt.Sprintf("%v", v.ValueRaw)
		v.Value.StaticValue = &pointableValue
	default:
		err := mapstructure.Decode(v.ValueRaw, &v.Value)
		if err != nil {
			logging.Warning("mapstructure decode failed with: %v", err)
			return FailParseVar.New(locale.Tr("variables_err_invalid_value", v.Name, fmt.Sprintf("%v", v.ValueRaw)))
		}
	}

	return v.Validate()
}

// Validate asserts that the values used are actually valid
func (v *Variable) Validate() *failures.Failure {
	if v.Value.PullFrom != nil {
		if fail := v.Value.PullFrom.Validate(); fail != nil {
			return fail
		}
	}

	if v.Value.Share != nil {
		if fail := v.Value.Share.Validate(); fail != nil {
			return fail
		}
	}

	if v.Value.StaticValue != nil && (v.Value.Share != nil || v.Value.PullFrom != nil) {
		return FailValidateStaticValueWithPull.New(locale.Tr("variables_err_value_with_pull", v.Name))
	}

	if v.Value.StaticValue == nil && v.Value.Share == nil && v.Value.PullFrom == nil {
		return FailValidateValueEmpty.New(locale.Tr("variables_err_value_empty", v.Name))
	}

	return nil
}
