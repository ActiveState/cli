package authentication

import (
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/rtutils/ptr"
)

func TestAuth_cutoffReached(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name   string
		auth   *Auth
		cutoff time.Time
		want   bool
	}{
		{
			"Cutoff one second beyond keepalive",
			&Auth{
				lastRenewal: ptr.To(now),
				jwtLifetime: time.Second,
			},
			now.Add(2 * time.Second),
			true,
		},
		{
			"Cutoff one second before keepalive",
			&Auth{
				lastRenewal: ptr.To(now),
				jwtLifetime: 2 * time.Second,
			},
			now.Add(time.Second),
			false,
		},
		{
			"Cutoff equal to keepalive",
			&Auth{
				lastRenewal: ptr.To(now),
				jwtLifetime: time.Second,
			},
			now.Add(time.Second),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.auth
			if got := a.cutoffReached(tt.cutoff); got != tt.want {
				t.Errorf("cutoffReached() = %v, want %v", got, tt.want)
			}
		})
	}
}
