package target

import (
	"testing"

	"github.com/ActiveState/cli/internal/runbits/runtime"
)

func TestTrigger_IndicatesUsage(t *testing.T) {
	tests := []struct {
		name string
		t    runtime_runbit.Trigger
		want bool
	}{
		{
			"Activate counts as usage",
			runtime_runbit.TriggerActivate,
			true,
		},
		{
			"Reset exec does not count as usage",
			runtime_runbit.TriggerResetExec,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.t.IndicatesUsage(); got != tt.want {
				t.Errorf("IndicatesUsage() = %v, want %v", got, tt.want)
			}
		})
	}
}
