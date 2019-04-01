package mock

import (
	"reflect"

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
func (m *Mock) Input(message string, response string) (string, *failures.Failure) {
	args := m.Called(message, response)
	return args.String(0), failure(args.Get(1))
}

// Select prompts the user to select one entry from multiple choices
func (m *Mock) Select(message string, choices []string, response string) (string, *failures.Failure) {
	args := m.Called(message, choices, response)
	return args.String(0), failure(args.Get(1))
}

// OnMethod behaves like mock.On but disregards whether arguments match or not
func (m *Mock) OnMethod(methodName string) *tmock.Call {
	methodType := reflect.ValueOf(m).MethodByName(methodName).Type()
	anyArgs := []interface{}{}
	for i := 0; i < methodType.NumIn(); i++ {
		anyArgs = append(anyArgs, tmock.MatchedBy(MatchAny))
	}
	return m.On(methodName, anyArgs...)
}

func failure(arg interface{}) *failures.Failure {
	if arg == nil {
		return nil
	}
	return arg.(*failures.Failure)
}

func MatchAny(a interface{}) bool {
	return true
}
