package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/ActiveState/cli/internal/ipc"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/test/pseudo/cmd/internal/serve"
	intsvcctl "github.com/ActiveState/cli/test/pseudo/cmd/internal/svcctl"
)

type namedClose struct {
	name string
	io.Closer
}

type gracefulShutdowner interface {
	Shutdown() error
	Wait() error
}

type gracefulShutdownerWrap struct {
	gracefulShutdowner
}

func (s gracefulShutdownerWrap) Close() error {
	if err := s.gracefulShutdowner.Shutdown(); err != nil {
		return err
	}

	return s.gracefulShutdowner.Wait()
}

func main() {
	if exitCode, err := run(); err != nil {
		cmd := path.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "%s: %s\n", cmd, err)

		if errors.Is(err, ipc.ErrInUse) {
			exitCode = 7
		}
		os.Exit(exitCode)
	}
}

func run() (int, error) {
	var (
		channel = "default"
		svcName = "svc"
	)

	defer fmt.Printf("%s: goodbye\n", svcName)

	flag.StringVar(&channel, "c", channel, "channel name")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpSrv := serve.New()
	addr, err := httpSrv.Run()
	if err != nil {
		return 1, err
	}

	spath := intsvcctl.NewIPCSockPath()
	spath.AppChannel = channel
	reqHandlers := []ipc.RequestHandler{
		svcctl.HTTPAddrHandler(addr),
	}
	ipcSrv := ipc.NewServer(ctx, spath, reqHandlers...)
	ipcClient := ipc.NewClient(spath)
	if err := ipcSrv.Start(); err != nil {
		return 2, err
	}

	errs := make(chan error)

	callOnSysSigs(ctx, svcName, cancel)
	callWhenNotVerified(ctx, errs, svcName, addr, ipcClient, cancel)

	return closeOnCancel(
		ctx,
		svcName,
		namedClose{"http", httpSrv},
		namedClose{"ipc", gracefulShutdownerWrap{ipcSrv}},
	)
}

func closeOnCancel(ctx context.Context, svcName string, ncs ...namedClose) (int, error) {
	<-ctx.Done()

	var exitCode int
	var retErr error

	for _, nc := range ncs {
		fmt.Printf("%s: closing %s\n", svcName, nc.name)
		if err := nc.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", svcName, err)
			if retErr != nil {
				exitCode = 3
				retErr = err
			}
		}
	}

	return exitCode, retErr
}

func callOnSysSigs(ctx context.Context, svcName string, fn func()) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		defer close(sigs)

		select {
		case <-ctx.Done():
			return
		case sig, ok := <-sigs:
			if !ok {
				return
			}

			fmt.Printf("%s: handling signal: %s\n", svcName, sig)
			fn()
		}
	}()
}

func callWhenNotVerified(ctx context.Context, errs chan error, svcName, addr string, ipComm svcctl.IPCommunicator, fn func()) {
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 3):
			checkedAddr, err := svcctl.LocateHTTP(ipComm)
			if err == nil && checkedAddr != addr {
				err = fmt.Errorf("checked addr %q does not match current %q", checkedAddr, addr)
			}
			if err != nil {
				errs <- err
				fn()
			}
		}
	}()
}
