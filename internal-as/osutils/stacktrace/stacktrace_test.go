package stacktrace

import (
	"testing"

	"github.com/ActiveState/cli/internal-as/rtutils"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name          string
		wantFirstFile string
	}{
		{
			"Stacktrace",
			rtutils.CurrentFile(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Get().Frames[0].Path; got != tt.wantFirstFile {
				t.Errorf("Get() first file = %s, want %s", got, tt.wantFirstFile)
			}
		})
	}
}
