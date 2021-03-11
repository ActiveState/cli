package svc

import (
	"fmt"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/idl"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

type Client struct {
	idl.VersionSvcClient
	conn *grpc.ClientConn
}

// New will create a new API client using default settings (for an authenticated version use the NewWithAuth version)
func New(cfg *config.Instance) (*Client, error) {
	port := cfg.GetInt("port")
	if port <= 0 {
		return nil, locale.NewError("err_svc_no_port", "The State Tool service does not appear to be running (no local port was configured).")
	}
	address := fmt.Sprintf("127.0.0.1:%d", port)
	// Set up a connection to the server.
	logging.Debug("Connecting to service: %s", address)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, string(address), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, locale.WrapError(err, "err_svc_dial", "Could not connect to service process. Error received: {{.V0}}. Please try again or contact support.", err.Error())
	}

	return &Client{
		idl.NewVersionSvcClient(conn),
		conn,
	}, nil
}
