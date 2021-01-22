package models

import (
	"encoding/json"
	"fmt"
)

//go:generate enumer -type=BuildType -transform=kebab $GOFILE

// BuildType represents what kind of build process is performed on this
// artifact. An artifact's type has implications on how the build wrapper is
// invoked (such as what contextual information is passed to it).
type BuildType int

const (
	_ BuildType = iota
	// Builder is an artifact which is built by compiling or otherwise
	// processing source code into an executable format.
	Builder
	// Packager is an artifact which doesn't actually take in any new source
	// code, but rather takes the built output of one or more Builder artifacts
	// and packages it together into a single artifact, such as an installer.
	Packager
)

// Scan implements the sql.Scanner interface so string values coming from a SQL
// result set can be ready directly into a BuildType
func (bt *BuildType) Scan(value interface{}) error {
	strValue, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("Cannot convert value of type %T to a build type", value)
	}

	parsed, err := BuildTypeString(string(strValue))
	if err != nil {
		return err
	}

	*bt = parsed
	return nil
}

// MarshalJSON serializes this enum using its string representation rather than
// as an integer.
func (bt *BuildType) MarshalJSON() ([]byte, error) {
	return json.Marshal(bt.String())
}

const _BuildTypeName = "builderpackager"

var _BuildTypeIndex = [...]uint8{0, 7, 15}

func (i BuildType) String() string {
	i -= 1
	if i < 0 || i >= BuildType(len(_BuildTypeIndex)-1) {
		return fmt.Sprintf("BuildType(%d)", i+1)
	}
	return _BuildTypeName[_BuildTypeIndex[i]:_BuildTypeIndex[i+1]]
}

var _BuildTypeValues = []BuildType{1, 2}

var _BuildTypeNameToValueMap = map[string]BuildType{
	_BuildTypeName[0:7]:  1,
	_BuildTypeName[7:15]: 2,
}

// BuildTypeString retrieves an enum value from the enum constants string name.
// Throws an error if the param is not part of the enum.
func BuildTypeString(s string) (BuildType, error) {
	if val, ok := _BuildTypeNameToValueMap[s]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("%s does not belong to BuildType values", s)
}

// BuildTypeValues returns all values of the enum
func BuildTypeValues() []BuildType {
	return _BuildTypeValues
}

// IsABuildType returns "true" if the value is listed in the enum definition. "false" otherwise
func (i BuildType) IsABuildType() bool {
	for _, v := range _BuildTypeValues {
		if i == v {
			return true
		}
	}
	return false
}
