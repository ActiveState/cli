package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/ActiveState/cli/exp/pm/cmd/internal/serve"
	"github.com/ActiveState/cli/exp/pm/internal/ipc"
	"github.com/ActiveState/cli/exp/pm/internal/svccomm"
)

func main() {
	if err := run(); err != nil {
		cmd := path.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "%s: %s\n", cmd, err)
		os.Exit(1)
	}
}

func run() error {
	var (
		rootDir = "/tmp/svccomm"
		name    = "state"
		version = "default"
		hash    = "DEADBEEF"
		svcName = "svc"
	)

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

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	defer close(sigs)

	go func() {
		sig, ok := <-sigs
		if !ok {
			return
		}
		fmt.Printf("%s: handling signal: %s\n", svcName, sig)

		fmt.Printf("%s: closing ipc\n", svcName)
		if err := ipcSrv.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", svcName, err)
		}

		fmt.Printf("%s: closing server\n", svcName)
		if err := httpSrv.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", svcName, err)
		}
	}()
	time.Sleep(time.Millisecond)

	var wg sync.WaitGroup

	errs := make(chan error)
	wg.Add(1)
	go func() {
		defer wg.Done()

		for err := range errs {
			if errors.Is(err, ipc.ErrInUse) {
				httpSrv.Close() // TODO: make this less gross
			}
			fmt.Fprintf(os.Stderr, "%s: errored early: %s\n", svcName, err)
		}
	}()

	wg.Add(1)
	go func() {
		defer close(errs)
		defer wg.Done()

		// TODO: work out the best order of closing/cleanup
		if err = ipcSrv.ListenAndServe(); err != nil {
			errs <- err
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		httpSrv.Wait() //nolint // add error handling
	}()

	fmt.Printf("%s: waiting\n", svcName)
	wg.Wait()

	return nil
}
