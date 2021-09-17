package analytics

import (
	"reflect"
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/stretchr/testify/require"
)

func Test_sendEvent(t *testing.T) {
	deferValue := deferAnalytics
	defer func() {
		deferAnalytics = deferValue
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
			deferAnalytics = tt.deferValue
			if err := sendEvent(tt.values[0], tt.values[1], tt.values[2], map[string]string{}); err != nil {
				t.Errorf("sendEvent() error = %s", errs.JoinMessage(err))
			}
			got, _ := loadDeferred(DeferrerFilePath())
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
				got, _ = loadDeferred(DeferrerFilePath())
				if len(got) > 0 {
					t.Errorf("Deferred events not cleared after sending, got: %v", got)
				}
			}
			require.NoFileExists(t, DeferrerFilePath(), "deferrer file should have been cleaned up")
		})
	}
}
