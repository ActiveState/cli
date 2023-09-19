package target

import "testing"

func TestTrigger_IndicatesUsage(t *testing.T) {
	tests := []struct {
		name string
		t    Trigger
		want bool
	}{
		{
			"Activate counts as usage",
			TriggerActivate,
			true,
		},
		{
			"Reset exec does not count as usage",
			TriggerResetExec,
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
