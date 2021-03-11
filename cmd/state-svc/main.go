package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"google.golang.org/grpc"

	"github.com/ActiveState/cli/cmd/state-svc/internal/services"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/idl"
	"github.com/ActiveState/cli/internal/logging"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, errs.Join(err, ": ").Error())
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not initialize config")
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return errs.Wrap(err, "Failed to listen")
	}

	// todo: we may be able to use the service manager for this, once we implement it
	address := strings.Split(lis.Addr().String(), ":")
	port, err := strconv.Atoi(address[len(address)-1])
	if err != nil {
		return errs.Wrap(err, "Could not parse port from address: %v", address)
	}
	if err := cfg.Set("port", port); err != nil {
		return errs.Wrap(err, "Could not save config")
	}
	fmt.Println("Listening on " + lis.Addr().String())

	s := grpc.NewServer()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		oscall := <-c
		logging.Debug("system call:%+v", oscall)
		s.GracefulStop()
	}()

	idl.RegisterVersionSvcServer(s, services.NewVersion())
	if err := s.Serve(lis); err != nil {
		return errs.Wrap(err, "failed to serve: %v", err)
	}

	return nil
}
