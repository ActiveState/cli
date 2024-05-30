package target

import (
	"testing"

	"github.com/ActiveState/cli/internal/runbits/runtime/target"
)

func TestTrigger_IndicatesUsage(t *testing.T) {
	tests := []struct {
		name string
		t    target.Trigger
		want bool
	}{
		{
			"Activate counts as usage",
			target.TriggerActivate,
			true,
		},
		{
			"Reset exec does not count as usage",
			target.TriggerResetExec,
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
