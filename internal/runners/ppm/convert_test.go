package ppm

import (
	"os"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/go/src/context"
	"github.com/autarch/testify/assert"
)

var FailPromptTest = failures.Type("test.prompt")

type testSignal struct{}

func (ts testSignal) String() string { return "testSignal" }
func (ts testSignal) Signal()        {}

var closedFailure = FailPromptTest.New("closed")
var forcedFailure = FailPromptTest.New("error")
var interruptedFailure = FailPromptTest.New("interrupted")

func surveyReturnOptions(nums <-chan int) surveySelectFunc {
	return func(message string, choices []string, defaultResponse string) (string, *failures.Failure) {
		num, ok := <-nums
		if !ok {
			return "", closedFailure
		}
		if num == -1 {
			return "", forcedFailure
		}
		if num == -2 {
			panic("interrupted")
		}
		return choices[num], nil
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
		wantPanic       bool
	}{
		{
			"accept immediately", []int{0},
			accepted,
			"",
			false, false,
		},
		{
			"accept after message", []int{1, 2},
			accepted,
			"asked why",
			false, false,
		},
		{
			"accept after state tool info", []int{1, 0, 1},
			accepted,
			"asked why,visited state tool info",
			false, false,
		},
		{
			"accept after all info", []int{1, 0, 0, 0},
			accepted,
			"asked why,visited state tool info,visited platform info",
			false, false,
		},
		{
			"accept eventually", []int{1, 3, 0},
			accepted,
			"asked why,still wanted ppm",
			false, false,
		},
		{
			"reject", []int{1, 3, 1},
			rejected,
			"asked why,still wanted ppm",
			false, false,
		},
		{
			"canceled with error", []int{1, -1},
			canceled,
			"asked why",
			true, false,
		},
		{
			"canceled by ctrl-c", []int{1, 0},
			canceled,
			"asked why,visited state tool info",
			false, true,
		},
	}

	for _, run := range tests {
		t.Run(run.name, func(tt *testing.T) {
			numsC := make(chan int)
			eventsC := make(chan testEvent, 2)
			ctx, cancel := context.WithCancel(context.Background())
			exitFunc := func() {
				select {
				case <-ctx.Done():
				case numsC <- -2:
				}
			}

			go func() {
				for _, num := range run.selections {
					select {
					case <-ctx.Done():
						return
					case numsC <- num:
					}
				}
			}()

			func() {
				defer close(numsC)
				defer close(eventsC)
				eventFunc := func(cat, action, label string) {
					eventsC <- testEvent{
						cat, action, label,
					}
				}
				cf := newConversionFlow(surveyReturnOptions(numsC), out, func(string) error { return nil })
				c := make(chan os.Signal)
				var timeout time.Duration = 30 * time.Second
				if run.wantPanic {
					timeout = 200 * time.Millisecond
				}
				go func(waitTillCtrlC time.Duration) {
					select {
					case <-ctx.Done():
					case <-time.After(waitTillCtrlC):
						select {
						case <-ctx.Done():
						case c <- testSignal{}:
						}
					}
				}(timeout)
				defer close(c)
				defer cancel()

				defer func(wantPanic bool) {
					r := recover()
					if r != nil && !wantPanic {
						// forward panic
						panic(r)
					}
				}(run.wantPanic)

				r, err := cf.run(c, eventFunc, exitFunc)
				if run.wantPanic {
					tt.Fatalf("should have panicked")
				}
				if run.wantErr != (err != nil) {
					tt.Fatalf("unexpected err value %v", err)
				}
				assert.Equal(tt, r, run.expectedResult)
			}()

			var events []testEvent
			for e := range eventsC {
				events = append(events, e)
			}
			assert.Len(tt, events, 1)
			assert.Equal(tt, testEvent{"ppm_conversion", run.expectedResult.String(), run.expectedVisited}, events[0])
		})
	}
}
