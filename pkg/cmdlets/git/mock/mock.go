package git

import (
	"fmt"
	"reflect"

	tmock "github.com/stretchr/testify/mock"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/cmdlets/git"
	"github.com/ActiveState/cli/pkg/projectfile"
)

var _ git.Repository = (*Mock)(nil)

// Mock the struct to mock the ProjectRepository struct
type Mock struct {
	tmock.Mock
}

// Init the mock
func Init() *Mock {
	return &Mock{}
}

// CloneProject will attempt to clone the associalted public git repository
// for the project identified by <owner>/<name> to the given directory
func (m *Mock) CloneProject(owner, name, path string, out output.Outputer, an analytics.AnalyticsDispatcher) error {
	args := m.Called(path)

	dummyID := "00010001-0001-0001-0001-000100010001"
	projectURL := fmt.Sprintf("https://%s/%s/%s?commitID=%s", constants.PlatformURL, owner, name, dummyID)
	_, err := projectfile.TestOnlyCreateWithProjectURL(projectURL, path)
	if err != nil {
		return err
	}

	return failure(args.Get(0))
}

// Close the mock
func (m *Mock) Close() {
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

func failure(arg interface{}) error {
	if arg == nil {
		return nil
	}
	return arg.(error)
}
