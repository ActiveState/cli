package retryhttp

import (
	"errors"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal-as/testhelpers/srvstatus"
)

func TestClient_Get(t *testing.T) {
	tests := []struct {
		name        string
		client      *Client
		path        string
		wantErrCode int
	}{
		{
			"Produces timeout error",
			NewClient(1*time.Millisecond, 0),
			"/200?sleep=30000",
			-1,
		},
		{
			"Produces server timeout error",
			DefaultClient,
			"/408",
			408,
		},
		{
			"Produces too early error",
			DefaultClient,
			"/425",
			425,
		},
		{
			"Produces too many error",
			NewClient(10*time.Second, 0), // 429 causes retries
			"/429",
			429,
		},
	}

	srv := srvstatus.New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			c.HTTPClient.Transport = srv.Client().Transport

			resp, err := c.Get(srv.URL + tt.path)
			if err == nil {
				t.Errorf("Error should not be nil")
				if resp.StatusCode != tt.wantErrCode {
					t.Errorf("server status code: got %d, want %d", resp.StatusCode, tt.wantErrCode)
				}
				return
			}
			werr := &UserNetworkError{}
			if !errors.As(err, &werr) {
				t.Fatalf("Error cannot be unwrapped to UserNetworkError")
			}
			if werr._testCode != tt.wantErrCode {
				t.Errorf("Wrapped error codes not equal:\n1: %d\n2: %d", werr._testCode, tt.wantErrCode)
			}
		})
	}
}
