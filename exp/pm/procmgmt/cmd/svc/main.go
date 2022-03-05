package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/ActiveState/cli/exp/pm/internal/proccomm"
	"github.com/ActiveState/cli/exp/pm/internal/serve"
	"github.com/ActiveState/cli/exp/pm/internal/socket"
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
		rootDir = "/tmp/proccomm"
		name    = "state"
		version = "default"
		hash    = "DEADBEEF"
	)

	flag.StringVar(&version, "v", version, "version id")
	flag.Parse()

	srv := serve.New()
	addr, err := srv.Run()
	if err != nil {
		return err
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan struct{})
	errs := make(chan error)

	go func() {
		defer close(done)

		select {
		case sig := <-sigs:
			fmt.Printf("handling signal: %s\n", sig)
		case err = <-errs:
			fmt.Fprintf(os.Stderr, "errored early: %s\n", err)
		}

		fmt.Println("closing server")
		if err := srv.Close(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()
	time.Sleep(time.Millisecond)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		n := &socket.Namespace{
			RootDir:    rootDir,
			AppName:    name,
			AppVersion: version,
			AppHash:    hash,
		}
		sock := socket.New(n, proccomm.HTTPAddrMHandler(addr))

		if err = sock.ListenAndServe(); err != nil {
			select {
			case errs <- err:
			default:
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		srv.Wait() //nolint // add error handling
	}()

	wg.Wait()
	return nil
}
