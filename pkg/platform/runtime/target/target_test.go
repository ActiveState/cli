package target

import (
	"testing"

	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
)

func TestTrigger_IndicatesUsage(t *testing.T) {
	tests := []struct {
		name string
		t    trigger.Trigger
		want bool
	}{
		{
			"Activate counts as usage",
			trigger.TriggerActivate,
			true,
		},
		{
			"Reset exec does not count as usage",
			trigger.TriggerResetExec,
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
