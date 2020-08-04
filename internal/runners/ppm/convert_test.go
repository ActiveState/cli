package ppm

import (
	"testing"

	"github.com/ActiveState/cli/internal/analytics"
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
	action string
	label  string
}

func TestConversionFlow(t *testing.T) {
	out := outputhelper.NewCatcher()

	tests := []struct {
		name           string
		selections     []int
		expectedResult conversionResult
		expectedEvents []testEvent
		wantErr        bool
	}{
		{
			"accept immediately", []int{0},
			accepted,
			[]testEvent{},
			false,
		},
		{
			"accept after message", []int{1, 2},
			accepted,
			[]testEvent{{"click", "asked-why"}},
			false,
		},
		{
			"accept after state tool info", []int{1, 0, 1},
			accepted,
			[]testEvent{{"click", "asked-why"}, {"click", "state-tool-info"}},
			false,
		},
		{
			"accept after all info", []int{1, 0, 0, 0},
			accepted,
			[]testEvent{{"click", "asked-why"}, {"click", "state-tool-info"}, {"click", "platform-info"}},
			false,
		},
		{
			"accept eventually", []int{1, 3, 0},
			accepted,
			[]testEvent{{"click", "asked-why"}, {"click", "still-wants-ppm"}},
			false,
		},
		{
			"reject", []int{1, 3, 1},
			rejected,
			[]testEvent{{"click", "asked-why"}, {"click", "still-wants-ppm"}},
			false,
		},
		{
			"canceled with error", []int{1, -1},
			canceled,
			[]testEvent{{"click", "asked-why"}},
			true,
		},
	}

	for _, run := range tests {
		t.Run(run.name, func(tt *testing.T) {
			events := []testEvent{}
			eventFunc := func(cat, action, label string) {
				assert.Equal(tt, analytics.CatPpmConversion, cat)
				events = append(events, testEvent{
					action, label,
				})
			}
			cf := newConversionFlow(surveyReturnOptions(run.selections), out, func(string) error { return nil }, eventFunc)

			r, err := cf.runSurvey()
			if run.wantErr != (err != nil) {
				tt.Fatalf("unexpected err value %v", err)
			}
			assert.Equal(tt, r, run.expectedResult)

			assert.Len(tt, events, len(run.expectedEvents))
			assert.Equal(tt, run.expectedEvents, events)
		})
	}
}
