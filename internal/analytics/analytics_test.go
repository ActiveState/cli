package analytics

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"

	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
)

const CatTest = "tests"

type AnalyticsTestSuite struct {
	suite.Suite

	authMock *authMock.Mock
}

func (suite *AnalyticsTestSuite) BeforeTest(suiteName, testName string) {
	suite.authMock = authMock.Init()
	suite.authMock.MockLoggedin()
}

func (suite *AnalyticsTestSuite) TestSetup() {
	setup()
	suite.Require().NotNil(client, "Client is set")
}

func (suite *AnalyticsTestSuite) TestEvent() {
	err := event(CatTest, "TestEvent")
	suite.Require().NoError(err, "Should send event without causing an error")
}

func TestAnalyticsTestSuite(t *testing.T) {
	suite.Run(t, new(AnalyticsTestSuite))
}

func Test_sendEvent(t *testing.T) {
	deferValue := Defer
	defer func() {
		Defer = deferValue
	}()

	tests := []struct {
		name       string
		deferValue bool
		values     []string
		want       []string
	}{
		{
			"Deferred",
			true,
			[]string{"category", "action", "label"},
			[]string{"category", "action", "label"},
		},
		{
			"Not Deferred",
			false,
			[]string{"category", "action", "label"},
			[]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Defer = tt.deferValue
			if err := sendEvent(tt.values[0], tt.values[1], tt.values[2], map[string]string{}); err != nil {
				t.Errorf("sendEvent() error = %v", err)
			}
			got := loadDeferred()
			gotSlice := []string{}
			if len(got) > 0 {
				gotSlice = []string{got[0].Category, got[0].Action, got[0].Label}
			}
			if !reflect.DeepEqual(gotSlice, tt.want) {
				t.Errorf("deferredEvents() = %v, want %v", gotSlice, tt.want)
			}
			if len(got) > 0 {
				called := false
				sendDeferred(func(category string, action string, label string, _ map[string]string) error {
					called = true
					gotSlice := []string{category, action, label}
					if !reflect.DeepEqual(gotSlice, tt.want) {
						t.Errorf("sendDeferred() = %v, want %v", gotSlice, tt.want)
					}
					return nil
				})
				if !called {
					t.Errorf("sendDeferred not called")
				}
				got = loadDeferred()
				if len(got) > 0 {
					t.Errorf("Deferred events not cleared after sending, got: %v", got)
				}
			}
			saveDeferred([]deferredData{}) // Ensure cleanup
		})
	}
}