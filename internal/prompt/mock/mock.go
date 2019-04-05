package mock

import (
	"reflect"

	"github.com/ActiveState/cli/internal/prompt"

	"github.com/ActiveState/cli/internal/failures"
	tmock "github.com/stretchr/testify/mock"
)

type Mock struct {
	tmock.Mock
}

func Init() *Mock {
	return &Mock{}
}

func (m *Mock) Close() {
}

// Input prompts the user for input
func (m *Mock) Input(message, defaultResponse string, flags ...prompt.Flag) (string, *failures.Failure) {
	args := m.Called(message, defaultResponse, flags)
	return args.String(0), failure(args.Get(1))
}

// Select prompts the user to select one entry from multiple choices
func (m *Mock) InputAndValidate(message, defaultResponse string, validator func(val interface{}) error) (response string, fail *failures.Failure) {
	args := m.Called(message, defaultResponse, validator)
	return args.String(0), failure(args.Get(1))
}

// Select prompts the user to select one entry from multiple choices
func (m *Mock) Select(message string, choices []string, defaultChoice string) (string, *failures.Failure) {
	args := m.Called(message, choices, defaultChoice)
	return args.String(0), failure(args.Get(1))
}

// Confirm prompts user for yes or no response.
func (m *Mock) Confirm(message string, defaultChoice bool) (bool, *failures.Failure) {
	args := m.Called(message, defaultChoice)
	return args.Bool(0), failure(args.Get(1))
}

// InputPassword prompts the user for input and obfuscates the text in stdout.
// Will fail if empty.
func (m *Mock) InputPassword(message string) (response string, fail *failures.Failure) {
	args := m.Called(message)
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

func failure(arg interface{}) *failures.Failure {
	if arg == nil {
		return nil
	}
	return arg.(*failures.Failure)
}
