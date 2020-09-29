package updater

import (
	"os"
	"testing"
	"time"
)

func Test_exeOverDayOld(t *testing.T) {
	tests := []struct {
		name       string
		setExeTime time.Time
		want       bool
	}{
		{
			"Exe is less than a day old",
			time.Now(),
			false,
		},
		{
			"Exe is over a day old",
			time.Now().Add(-25 * time.Hour),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exe, _ := os.Executable()
			os.Chtimes(exe, tt.setExeTime, tt.setExeTime)
			if got := exeOverDayOld(); got != tt.want {
				t.Errorf("exeOverDayOld() = %v, want %v", got, tt.want)
			}
		})
	}
}
