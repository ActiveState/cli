package ppm

import (
	"testing"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/autarch/testify/assert"
)

var FailPromptTest = failures.Type("test.prompt")

type testSignal struct{}

func (ts testSignal) String() string { return "testSignal" }
func (ts testSignal) Signal()        {}

var overflowFailure = FailPromptTest.New("overflow")
var forcedFailure = FailPromptTest.New("error")

func surveyReturnOptions(selections []int) surveySelectFunc {
	i := 0

	return func(message string, choices []string, defaultResponse string) (string, *failures.Failure) {
		if i >= len(selections) {
			return "", overflowFailure
		}
		c := selections[i]
		i++
		if c == -1 {
			return "", forcedFailure
		}
		return choices[c], nil
	}
}

type testEvent struct {
	category string
	action   string
	visited  string
}

func TestConversionFlow(t *testing.T) {
	out := outputhelper.NewCatcher()

	tests := []struct {
		name            string
		selections      []int
		expectedResult  conversionResult
		expectedVisited string
		wantErr         bool
	}{
		{
			"accept immediately", []int{0},
			accepted,
			"",
			false,
		},
		{
			"accept after message", []int{1, 2},
			accepted,
			"asked why",
			false,
		},
		{
			"accept after state tool info", []int{1, 0, 1},
			accepted,
			"asked why,visited state tool info",
			false,
		},
		{
			"accept after all info", []int{1, 0, 0, 0},
			accepted,
			"asked why,visited state tool info,visited platform info",
			false,
		},
		{
			"accept eventually", []int{1, 3, 0},
			accepted,
			"asked why,still wanted ppm",
			false,
		},
		{
			"reject", []int{1, 3, 1},
			rejected,
			"asked why,still wanted ppm",
			false,
		},
		{
			"canceled with error", []int{1, -1},
			canceled,
			"asked why",
			true,
		},
	}

	for _, run := range tests {
		t.Run(run.name, func(tt *testing.T) {
			var events []testEvent
			eventFunc := func(cat, action, label string) {
				events = append(events, testEvent{
					cat, action, label,
				})
			}
			cf := newConversionFlow(surveyReturnOptions(run.selections), out, func(string) error { return nil })

			r, err := cf.run(eventFunc)
			if run.wantErr != (err != nil) {
				tt.Fatalf("unexpected err value %v", err)
			}
			assert.Equal(tt, r, run.expectedResult)

			assert.Len(tt, events, 1)
			assert.Equal(tt, testEvent{"ppm_conversion", run.expectedResult.String(), run.expectedVisited}, events[0])
		})
	}
}
