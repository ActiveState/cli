package daemon

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"google.golang.org/grpc"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/idl"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

type ErrDaemonNotRunning struct{ error }

func IsNotRunningError(err error) bool {
	return errs.Matches(err, &ErrDaemonNotRunning{})
}

type Client struct {
	idl.DaemonClient
	conn *grpc.ClientConn
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func New(address AddressedDaemon) (*Client, error) {
	// Set up a connection to the server.
	logging.Debug("Connecting to idl: %s", address)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, string(address), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, locale.WrapError(err, "err_daemon_dial", "Could not connect to idl process. Error received: {{.V0}}. Please try again or contact support.", err.Error())
	}

	return &Client{idl.NewDaemonClient(conn), conn}, nil
}

type AddressedDaemon string

func NewAddressedDaemon(basePath string) (AddressedDaemon, error) {
	address, err := detectAddress(basePath)
	if err == nil {
		return address, nil
	}
	if !IsNotRunningError(err) {
		return "", errs.Wrap(err, "Could not detect idl address")
	}

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	executable := filepath.Join(basePath, "idl"+ext)
	executable = "/tmp/idl"
	cmd := exec.Command(executable)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return "", errs.Wrap(err, "Could not start idl process")
	}

	if err := cmd.Process.Release(); err != nil {
		return "", errs.Wrap(err, "Could not release idl process")
	}

	sleepTime := time.Millisecond * 100
	for i := 0; i < 5; i++ {
		address, err = detectAddress(basePath)
		if err == nil {
			break
		}
		logging.Debug("Daemon not running after %d tries, waiting %d milliseconds", i, sleepTime/time.Millisecond)
		time.Sleep(sleepTime)
		sleepTime = sleepTime * 2
	}
	return address, err
}

func detectAddress(basePath string) (AddressedDaemon, error) {
	df := daemonFile(basePath)
	if fileutils.TargetExists(df) {
		logging.Debug("Detected running unix idl: %v", df)
		return AddressedDaemon("unix:" + df), nil
	}
	dpf := daemonPortFile(basePath)
	if fileutils.FileExists(dpf) {
		b, err := fileutils.ReadFile(dpf)
		if err != nil {
			return "", errs.Wrap(err, "Could not read idl port file")
		}
		logging.Debug("Detected running network idl: %v", dpf)
		return AddressedDaemon(b), nil
	}
	return "", &ErrDaemonNotRunning{errs.New("Daemon is not running.")}
}

func daemonFile(basePath string) string {
	return idl.UnixSocketFile(basePath)
}

func daemonPortFile(basePath string) string {
	return idl.NetworkPortFile(basePath)
}
