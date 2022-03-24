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
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/ActiveState/cli/internal/ipc"
	"github.com/ActiveState/cli/internal/svccomm"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/test/pseudo/cmd/internal/serve"
)

type namedClose struct {
	name string
	io.Closer
}

func main() {
	if err := run(); err != nil {
		cmd := path.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "%s: %s\n", cmd, err)

		exitCode := 1
		if errors.Is(err, ipc.ErrInUse) {
			exitCode = 7
		}
		os.Exit(exitCode)
	}
}

func run() error {
	var (
		rootDir = filepath.Join(os.TempDir(), "svccomm")
		name    = "state"
		version = "default"
		hash    = "DEADBEEF"
		svcName = "svc"
	)

	defer fmt.Printf("%s: goodbye\n", svcName)

	flag.StringVar(&version, "v", version, "version id")
	flag.Parse()

	httpSrv := serve.New()
	addr, err := httpSrv.Run()
	if err != nil {
		return err
	}

	n := &ipc.Namespace{
		RootDir:    rootDir,
		AppName:    name,
		AppVersion: version,
		AppHash:    hash,
	}
	mhs := []ipc.MatchedHandler{
		svccomm.HTTPAddrMHandler(addr),
	}
	ipcSrv := ipc.New(n, mhs...)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errs := make(chan error)

	callOnSysSigs(ctx, svcName, cancel)
	callWhenNotVerified(ctx, errs, svcName, addr, n, cancel)
	shutDownOnDone(
		ctx,
		svcName,
		namedClose{"ipc", ipcSrv},
		namedClose{"http", httpSrv},
	)

	go func() {
		defer close(errs)

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cancel()

			if err = ipcSrv.ListenAndServe(); err != nil {
				errs <- err
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()

			if err = httpSrv.Wait(); err != nil {
				errs <- err
			}
		}()

		fmt.Printf("%s: waiting\n", svcName)
		wg.Wait()
	}()

	var reportErr error
	for err := range errs {
		if reportErr == nil {
			cancel()
			reportErr = err
		}

		fmt.Fprintf(os.Stderr, "%s (outputing all): %s\n", svcName, err)
	}

	return reportErr
}

func shutDownOnDone(ctx context.Context, svcName string, ncs ...namedClose) {
	go func() {
		<-ctx.Done()

		for _, nc := range ncs {
			fmt.Printf("%s: closing %s\n", svcName, nc.name)
			if err := nc.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s\n", svcName, err)
			}
		}
	}()
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

func callWhenNotVerified(ctx context.Context, errs chan error, svcName, addr string, n *ipc.Namespace, fn func()) {
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 3):
			checkedAddr, err := svcctl.LocateHTTP(n)
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
