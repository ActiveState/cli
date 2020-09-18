package retryhttp

import (
	"errors"
	"testing"
	"time"
)

func TestClient_Get(t *testing.T) {
	tests := []struct {
		name        string
		client      *Client
		url         string
		wantErrCode int
	}{
		{
			"Produces timeout error",
			NewClient(1*time.Microsecond, 0),
			"https://httpstat.us/200?sleep=30000",
			-1,
		},
		{
			"Produces server timeout error",
			DefaultClient,
			"https://httpstat.us/408",
			408,
		},
		{
			"Produces too early error",
			DefaultClient,
			"https://httpstat.us/425",
			425,
		},
		{
			"Produces too many error",
			NewClient(10*time.Second, 0), // 429 causes retries
			"https://httpstat.us/429",
			429,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			_, err := c.Get(tt.url)
			if (err != nil) != true {
				t.Errorf("Error should not be nil")
				return
			}
			werr := &UserNetworkError{}
			if !errors.As(err, &werr) {
				t.Errorf("Error cannot be unwrapped to UserNetworkError")
			}
			if werr._testCode != tt.wantErrCode {
				t.Errorf("Error codes not equal:\n1: %d\n2: %d", werr._testCode, tt.wantErrCode)
				return
			}
		})
	}
}
