package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"google.golang.org/grpc"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/idl"
	"github.com/ActiveState/cli/internal/logging"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	idl.UnimplementedDaemonServer
}

// Ensure server implements DaemonServer
var _ idl.DaemonServer = &server{}

func (s *server) HelloWorld(ctx context.Context, in *idl.Empty) (*idl.StringMessage, error) {
	logging.Debug("Received SayHello")
	return &idl.StringMessage{Value: "Hello World!"}, nil
}

func main() {
	useNetworkSocket := flag.Bool("network", false, "Force running idl with network socket (otherwise uses unix socket).")
	flag.Parse()

	cfg, err := config.New()
	if err != nil {
		logging.Error("Could not initialize config: %v", err)
		os.Exit(1)
	}

	network := "unix"
	address := filepath.Join(cfg.ConfigPath(), constants.DaemonFile)
	if useNetworkSocket != nil && *useNetworkSocket {
		network = "tcp"
		address = ":" + constants.DaemonPort
	}

	logging.Debug("starting idl on " + address)
	lis, err := net.Listen(network, address)
	if err != nil {
		logging.Error("failed to listen: %v", err)
		os.Exit(1)
	}

	if network == "tcp" {
		fileutils.WriteFile(idl.NetworkPortFile(cfg.ConfigPath()), []byte(address))
	}

	s := grpc.NewServer()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		oscall := <-c
		logging.Debug("system call:%+v", oscall)
		s.GracefulStop()
	}()

	idl.RegisterDaemonServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		logging.Error("failed to serve: %v", err)
		os.Exit(1)
	}

}
