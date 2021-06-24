package mock

import (
	"reflect"

	"github.com/ActiveState/cli/internal/prompt"

	tmock "github.com/stretchr/testify/mock"
)

var _ prompt.Prompter = &Mock{}

// Mock the struct to mock the Prompt struct
type Mock struct {
	tmock.Mock
}

// Init an object
func Init() *Mock {
	return &Mock{}
}

// Close it
func (m *Mock) Close() {
}

// Input prompts the user for input
func (m *Mock) Input(title, message string, defaultResponse *string, flags ...prompt.ValidatorFlag) (string, error) {
	args := m.Called(title, message, defaultResponse, flags)
	return args.String(0), failure(args.Get(1))
}

// InputAndValidate prompts the user for input witha  customer validator and validation flags
func (m *Mock) InputAndValidate(title, message string, defaultResponse *string, validator prompt.ValidatorFunc, flags ...prompt.ValidatorFlag) (response string, err error) {
	args := m.Called(message, message, defaultResponse, validator)
	return args.String(0), failure(args.Get(1))
}

// Select prompts the user to select one entry from multiple choices
func (m *Mock) Select(title, message string, choices []string, defaultChoice *string) (string, error) {
	args := m.Called(title, message, choices, defaultChoice)
	return args.String(0), failure(args.Get(1))
}

// Confirm prompts user for yes or no response.
func (m *Mock) Confirm(title, message string, defaultChoice *bool) (bool, error) {
	args := m.Called(title, message, defaultChoice)
	return args.Bool(0), failure(args.Get(1))
}

// InputSecret prompts the user for input and obfuscates the text in stdout.
// Will fail if empty.
func (m *Mock) InputSecret(title, message string, flags ...prompt.ValidatorFlag) (response string, err error) {
	args := m.Called(title, message)
	return args.String(0), failure(args.Get(1))
}

// OnMethod behaves like mock.On but disregards whether arguments match or not
func (m *Mock) OnMethod(methodName string) *tmock.Call {
	methodType := reflect.ValueOf(m).MethodByName(methodName).Type()
	anyArgs := []interface{}{}
	for i := 0; i < methodType.NumIn(); i++ {
		anyArgs = append(anyArgs, tmock.Anything)
	}
	return m.On(methodName, anyArgs...)
}

// IsInteractive always returns true.
func (m *Mock) IsInteractive() bool {
	return true
}

// IsPromptable always returns true.
func (m *Mock) IsPromptable() bool {
	return true
}

// IsPromptableOnce always returns true.
func (m *Mock) IsPromptableOnce(cfg prompt.Configurer, key prompt.OnceKey) bool {
	return true
}

func failure(arg interface{}) error {
	if arg == nil {
		return nil
	}
	return arg.(error)
}
