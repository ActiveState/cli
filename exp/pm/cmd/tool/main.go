package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ActiveState/cli/exp/pm/cmd/internal/serve"
	"github.com/ActiveState/cli/exp/pm/internal/ipc"
	"github.com/ActiveState/cli/exp/pm/internal/svccomm"
	"github.com/ActiveState/cli/exp/pm/internal/svcctl"
)

func main() {
	start := time.Now()

	var (
		rootDir = "/tmp/svccomm"
		name    = "state"
		version = "default"
		hash    = "DEADBEEF"
	)

	flag.StringVar(&version, "v", version, "version id")
	flag.Parse()
	fmt.Println("parsed flags", time.Since(start))

	n := &ipc.Namespace{
		RootDir:    rootDir,
		AppName:    name,
		AppVersion: version,
		AppHash:    hash,
	}
	sc := ipc.NewClient(n)
	pc := svccomm.NewClient(sc)
	fmt.Println("setup svccomm client", time.Since(start))

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*2)
	defer cancel()
	addr, err := pc.GetHTTPAddr(ctx)
	fmt.Println("got http addr", time.Since(start))

	if err != nil {
		if !errors.Is(err, ipc.ErrServerDown) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Println("starting service")
		svcCtl := svcctl.New(sc)
		ctx1, cancel1 := context.WithTimeout(context.Background(), time.Millisecond*2)
		defer cancel1()
		if err := svcCtl.Start(ctx1); err != nil {
			fmt.Println("starting")
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		ctx2, cancel2 := context.WithTimeout(context.Background(), time.Millisecond*2)
		defer cancel2()
		addr, err = pc.GetHTTPAddr(ctx2)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	c := serve.NewClient(addr)
	info, err := c.GetInfo()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("got info via http", time.Since(start))
	fmt.Print(info)
}
